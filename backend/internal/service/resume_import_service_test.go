package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"resume-builder/backend/internal/ai"
	"resume-builder/backend/internal/repository"
)

// Small real PDFs with an xref/page tree, not signature-only mocks.
func testPDF(pages int) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	objects := []string{"<< /Type /Catalog /Pages 2 0 R >>", ""}
	kids := []string{}
	for i := 0; i < pages; i++ {
		kids = append(kids, fmt.Sprintf("%d 0 R", i+3))
		objects = append(objects, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>")
	}
	objects[1] = fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), pages)
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return b.Bytes()
}

type fakeImportParser struct {
	result           string
	err              error
	calls            atomic.Int32
	entered, release chan struct{}
}

func (p *fakeImportParser) ParsePDF(ctx context.Context, _ string, _ []byte) (string, error) {
	p.calls.Add(1)
	if p.entered != nil {
		close(p.entered)
		select {
		case <-p.release:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return p.result, p.err
}

type fakeImportWriter struct {
	calls   int
	user    string
	content repository.ImportedResume
}

func (w *fakeImportWriter) Create(_ context.Context, user string, in repository.ImportedResume) (*repository.Resume, error) {
	w.calls++
	w.user = user
	w.content = in
	return &repository.Resume{ID: "new-resume", UserID: user, FullName: in.Resume.FullName}, nil
}

const validExtraction = `{"document_type":"resume","evidence":["A student's education and projects"],"warnings":[],"content":{"resume":{"full_name":"Anika Rao","links":[]},"educations":[{"institution":"Example University","degree":"BSc"}],"projects":[{"name":"Campus app","content":"Built an app"}]}}`

func requireImportCode(t *testing.T, err error, code string) {
	t.Helper()
	var failure *ImportError
	if !errors.As(err, &failure) || failure.Code != code {
		t.Fatalf("got %v, want %s", err, code)
	}
}

func TestResumeImportCreatesOnlyAcceptedContent(t *testing.T) {
	for _, tc := range []struct{ name, output, code string }{
		{"student resume", validExtraction, ""},
		{"job posting", strings.Replace(validExtraction, `"document_type":"resume"`, `"document_type":"job_description"`, 1), "not_resume"},
		{"uncertain", `{"document_type":"uncertain"}`, "uncertain_document"},
		{"unreadable", `{"document_type":"unreadable"}`, "uncertain_document"},
		{"empty resume", `{"document_type":"resume","evidence":["A name"],"content":{"resume":{"full_name":"Name"}}}`, "uncertain_document"},
		{"no evidence", `{"document_type":"resume","content":{}}`, "uncertain_document"},
		{"malformed", `{"document_type":"resume"`, "invalid_extraction"},
		{"trailing payload", validExtraction + ` {}`, "invalid_extraction"},
		{"model supplied owner", strings.Replace(validExtraction, `"full_name":"Anika Rao"`, `"user_id":"someone-else","full_name":"Anika Rao"`, 1), "invalid_extraction"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parser := &fakeImportParser{result: tc.output}
			writer := &fakeImportWriter{}
			result, err := NewResumeImportService(parser, writer).Import(context.Background(), "signed-in-user", "My CV.PDF", testPDF(1))
			if tc.code != "" {
				requireImportCode(t, err, tc.code)
				if writer.calls != 0 {
					t.Fatal("rejected content was stored")
				}
				return
			}
			if err != nil || result.Resume.ID != "new-resume" {
				t.Fatalf("import failed: %v", err)
			}
			if writer.calls != 1 || writer.user != "signed-in-user" || writer.content.Resume.Label != "My CV" || len(writer.content.Educations) != 1 || len(writer.content.Projects) != 1 {
				t.Fatal("lost content or ownership")
			}
			if len(result.Warnings) == 0 {
				t.Fatal("missing review notice")
			}
		})
	}
}

func TestResumeImportValidatesPDFBeforeAI(t *testing.T) {
	for _, tc := range []struct {
		name string
		pdf  []byte
		code string
	}{
		{"resume.docx", testPDF(1), "invalid_pdf"},
		{"resume.pdf", []byte("not a pdf"), "invalid_pdf"},
		{"resume.pdf", []byte("%PDF-1.7\nbroken"), "invalid_pdf"},
		{"resume.pdf", testPDF(11), "page_limit"},
		{"resume.pdf", make([]byte, MaxResumePDFBytes+1), "file_too_large"},
	} {
		parser := &fakeImportParser{}
		writer := &fakeImportWriter{}
		_, err := NewResumeImportService(parser, writer).Import(context.Background(), "user", tc.name, tc.pdf)
		requireImportCode(t, err, tc.code)
		if parser.calls.Load() != 0 || writer.calls != 0 {
			t.Fatal("invalid file reached AI or DB")
		}
	}
	conf := model.NewStatelessConfiguration()
	conf.UserPW = "secret"
	conf.OwnerPW = "owner"
	var locked bytes.Buffer
	if err := api.Encrypt(context.Background(), bytes.NewReader(testPDF(1)), &locked, conf); err != nil {
		t.Fatal(err)
	}
	if err := validateResumePDF(context.Background(), locked.Bytes()); err == nil {
		t.Fatal("encrypted PDF accepted")
	}
}

func TestResumeImportRejectsConcurrentUserRequest(t *testing.T) {
	parser := &fakeImportParser{result: validExtraction, entered: make(chan struct{}), release: make(chan struct{})}
	writer := &fakeImportWriter{}
	svc := NewResumeImportService(parser, writer)
	done := make(chan error, 1)
	go func() { _, err := svc.Import(context.Background(), "user", "resume.pdf", testPDF(1)); done <- err }()
	<-parser.entered
	_, err := svc.Import(context.Background(), "user", "resume.pdf", testPDF(1))
	requireImportCode(t, err, "import_busy")
	close(parser.release)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if writer.calls != 1 {
		t.Fatal("concurrent import wrote twice")
	}
}

func TestResumeImportProviderFailureDoesNotStore(t *testing.T) {
	parser := &fakeImportParser{err: &ai.PDFProviderError{Status: 429}}
	writer := &fakeImportWriter{}
	_, err := NewResumeImportService(parser, writer).Import(context.Background(), "user", "resume.pdf", testPDF(1))
	requireImportCode(t, err, "provider_busy")
	if writer.calls != 0 {
		t.Fatal("failed extraction wrote to DB")
	}
}

func TestImportedDataPreservesCustomContentAndRemovesUnsafeURLs(t *testing.T) {
	var extraction resumeExtraction
	if err := json.Unmarshal([]byte(validExtraction), &extraction); err != nil {
		t.Fatal(err)
	}
	extraction.Content.Resume.Links = json.RawMessage(`[{"label":"Bad","url":"javascript:alert(1)"},{"label":"Profile","url":"https://example.com"}]`)
	description := "Volunteered at a library"
	extraction.Content.CustomSections = []repository.ImportedCustomSection{{Title: "Community", Entries: []repository.CustomSectionEntryInput{{Description: &description}}}}
	if err := validateImportedContent(&extraction.Content); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(extraction.Content.Resume.Links), "javascript") || !strings.Contains(string(extraction.Content.Resume.Links), "https://example.com") {
		t.Fatal("unsafe link or lost valid link")
	}
	if *extraction.Content.CustomSections[0].Entries[0].Description != description {
		t.Fatal("custom content changed")
	}
}
