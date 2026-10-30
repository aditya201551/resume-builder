package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"resume-builder/backend/internal/ai"
	"resume-builder/backend/internal/repository"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const MaxResumePDFBytes = 10 * 1024 * 1024
const MaxResumePDFPages = 10

type ImportError struct{ Code, Message string }

func (e *ImportError) Error() string         { return e.Message }
func importError(code, message string) error { return &ImportError{Code: code, Message: message} }

type PDFParser interface {
	ParsePDF(context.Context, string, []byte) (string, error)
}
type ImportedResumeWriter interface {
	Create(context.Context, string, repository.ImportedResume) (*repository.Resume, error)
}

type ResumeImportService struct {
	parser PDFParser
	writer ImportedResumeWriter
	slots  chan struct{}
	mu     sync.Mutex
	active map[string]bool
}

func NewResumeImportService(parser PDFParser, writer ImportedResumeWriter) *ResumeImportService {
	return &ResumeImportService{parser: parser, writer: writer, slots: make(chan struct{}, 2), active: map[string]bool{}}
}

type ResumeImportResult struct {
	Resume   *repository.Resume `json:"resume"`
	Warnings []string           `json:"warnings"`
}

func (s *ResumeImportService) Import(ctx context.Context, userID, filename string, pdf []byte) (*ResumeImportResult, error) {
	if userID == "" {
		return nil, importError("unauthorized", "Sign in to import a resume.")
	}
	if len(pdf) > MaxResumePDFBytes {
		return nil, importError("file_too_large", "Choose a PDF smaller than 10 MB.")
	}
	if !strings.EqualFold(filepath.Ext(filename), ".pdf") || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return nil, importError("invalid_pdf", "Choose a valid PDF file.")
	}
	s.mu.Lock()
	if s.active[userID] {
		s.mu.Unlock()
		return nil, importError("import_busy", "Your resume is already being imported. Please wait.")
	}
	select {
	case s.slots <- struct{}{}:
		s.active[userID] = true
	default:
		s.mu.Unlock()
		return nil, importError("import_busy", "Resume import is busy. Please try again shortly.")
	}
	s.mu.Unlock()
	defer func() { <-s.slots; s.mu.Lock(); delete(s.active, userID); s.mu.Unlock() }()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()
	if err := validateResumePDF(ctx, pdf); err != nil {
		return nil, err
	}
	raw, err := s.parser.ParsePDF(ctx, importInstructions, pdf)
	if err != nil {
		if ctx.Err() != nil {
			return nil, importError("import_timeout", "Import took too long. Please try again with a smaller PDF.")
		}
		var providerErr *ai.PDFProviderError
		if errors.As(err, &providerErr) && (providerErr.Status == 429 || providerErr.Status == 529 || providerErr.Status == 503) {
			return nil, importError("provider_busy", "The AI service is busy. Please try again shortly.")
		}
		// Never send provider response bodies, credentials or extracted PII to the client/log.
		return nil, importError("parse_failed", "We couldn't read this PDF with the configured AI service. Try again or upload a clearer copy.")
	}
	extracted, err := decodeResumeImport(raw, filename)
	if err != nil {
		return nil, err
	}
	resume, err := s.writer.Create(ctx, userID, extracted.Content)
	if err != nil {
		return nil, fmt.Errorf("store imported resume: %w", err)
	}
	return &ResumeImportResult{Resume: resume, Warnings: extracted.Warnings}, nil
}

func validateResumePDF(ctx context.Context, pdf []byte) (err error) {
	defer func() {
		if recover() != nil {
			err = importError("invalid_pdf", "This PDF could not be read. Export a new PDF and try again.")
		}
	}()
	conf := model.NewStatelessConfiguration()
	// Parsing only: never fetch external resources or load attachments. Bound PDF
	// objects/decompression independently of the compressed upload size.
	conf.Limits.MaxDecodeBytes = 32 * 1024 * 1024
	conf.Limits.MaxObjectBytes = 16 * 1024 * 1024
	conf.Limits.MaxStreamBytes = 16 * 1024 * 1024
	conf.Limits.MaxObjectCount = 50000
	conf.Limits.MaxXRefEntries = 50000
	conf.Limits.MaxObjectStreamCount = 10000
	conf.Limits.MaxRecursionDepth = 50
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	parsed, e := api.ReadAndValidate(checkCtx, bytes.NewReader(pdf), conf)
	if e != nil {
		return importError("invalid_pdf", "This PDF is unreadable or password protected. Export an unlocked PDF and try again.")
	}
	if parsed.Encrypt != nil {
		return importError("invalid_pdf", "Password-protected PDFs aren't supported. Upload an unlocked copy.")
	}
	if parsed.PageCount < 1 || parsed.PageCount > MaxResumePDFPages {
		return importError("page_limit", "Upload a resume PDF with 1 to 10 pages.")
	}
	return nil
}

type resumeExtraction struct {
	DocumentType string                    `json:"document_type"`
	Evidence     []string                  `json:"evidence"`
	Warnings     []string                  `json:"warnings"`
	Content      repository.ImportedResume `json:"content"`
}

func decodeResumeImport(raw, filename string) (*resumeExtraction, error) {
	if len(raw) > 512*1024 {
		return nil, importError("invalid_extraction", "The extracted resume is too large. Try a shorter PDF.")
	}
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```json") {
		raw = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw, "```json"), "```"))
	}
	var result resumeExtraction
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&result) != nil {
		return nil, importError("invalid_extraction", "We couldn't extract reliable resume sections. Please try another PDF.")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, importError("invalid_extraction", "The extraction was incomplete. Please try again.")
	}
	if result.DocumentType == "uncertain" || result.DocumentType == "unreadable" {
		return nil, importError("uncertain_document", "We couldn't confidently identify a resume. Upload a clearer resume or CV PDF.")
	}
	if result.DocumentType != "resume" && result.DocumentType != "cv" {
		return nil, importError("not_resume", "This PDF doesn't appear to be a resume or CV. Upload a resume to continue.")
	}
	if len(result.Evidence) == 0 || len(result.Evidence) > 5 {
		return nil, importError("uncertain_document", "We couldn't confidently identify a resume. Please try another PDF.")
	}
	for _, evidence := range result.Evidence {
		if strings.TrimSpace(evidence) == "" || len(evidence) > 1000 {
			return nil, importError("uncertain_document", "We couldn't confidently identify a resume. Please try another PDF.")
		}
	}
	if len(result.Warnings) > 7 {
		result.Warnings = result.Warnings[:7]
	}
	for i, warning := range result.Warnings {
		if len([]rune(warning)) > 500 {
			result.Warnings[i] = string([]rune(warning)[:500])
		}
	}
	if err := validateImportedContent(&result.Content); err != nil {
		return nil, err
	}
	label := strings.TrimSpace(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	if label == "" {
		label = "Imported resume"
	}
	if len([]rune(label)) > 120 {
		label = string([]rune(label)[:120])
	}
	result.Content.Resume.Label = label
	result.Content.Resume.PhotoURL = nil
	// Warnings are informational; provider text is never rendered as HTML.
	result.Warnings = append([]string{"Imported into the Vertical template. Review names, dates and section details before sharing."}, result.Warnings...)
	if len(result.Warnings) > 8 {
		result.Warnings = result.Warnings[:8]
	}
	return &result, nil
}
