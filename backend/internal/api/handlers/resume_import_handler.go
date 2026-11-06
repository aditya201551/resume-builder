package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"resume-builder/backend/internal/ai"
	"resume-builder/backend/internal/platform/httpresponse"
	"time"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/service"
)

type resumeImporter interface {
	Import(context.Context, string, string, []byte, service.PDFParser) (*service.ResumeImportResult, error)
}

type ResumeImportHandler struct {
	svc    resumeImporter
	access *AIHandler
}

func NewResumeImportHandler(svc resumeImporter, access *AIHandler) *ResumeImportHandler {
	return &ResumeImportHandler{svc: svc, access: access}
}

func (h *ResumeImportHandler) Import(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	r = r.WithContext(ctx)
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(30 * time.Second))
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	cfg, ok := h.access.Resolve(w, r, true)
	if !ok {
		return
	}
	release, ok := h.access.Begin(w, r)
	if !ok {
		return
	}
	defer release()
	r.Body = http.MaxBytesReader(w, r.Body, service.MaxResumePDFBytes+128*1024)
	reader, err := r.MultipartReader()
	if err != nil {
		writeImportError(w, "invalid_upload", "Choose a PDF to import.", http.StatusBadRequest)
		return
	}
	part, err := reader.NextPart()
	if err != nil || part.FormName() != "file" || part.FileName() == "" {
		writeImportError(w, "invalid_upload", "Choose one PDF file to import.", http.StatusBadRequest)
		return
	}
	filename := part.FileName()
	pdf, err := io.ReadAll(io.LimitReader(part, service.MaxResumePDFBytes+1))
	part.Close()
	var maxBytes *http.MaxBytesError
	if errors.As(err, &maxBytes) || len(pdf) > service.MaxResumePDFBytes {
		writeImportError(w, "file_too_large", "Choose a PDF smaller than 10 MB.", http.StatusRequestEntityTooLarge)
		return
	}
	if err != nil {
		writeImportError(w, "invalid_upload", "The upload couldn't be read. Please try again.", http.StatusBadRequest)
		return
	}
	if _, err = reader.NextPart(); err != io.EOF {
		writeImportError(w, "invalid_upload", "Upload one PDF at a time.", http.StatusBadRequest)
		return
	}
	result, err := h.svc.Import(r.Context(), userID, filename, pdf, ai.NewPDFClient(cfg))
	if err != nil {
		var failure *service.ImportError
		if errors.As(err, &failure) {
			status := http.StatusUnprocessableEntity
			switch failure.Code {
			case "file_too_large":
				status = http.StatusRequestEntityTooLarge
			case "import_busy":
				status = http.StatusTooManyRequests
			case "provider_busy":
				status = http.StatusServiceUnavailable
			case "import_timeout":
				status = http.StatusGatewayTimeout
			case "parse_failed", "invalid_extraction":
				status = http.StatusBadGateway
			case "unauthorized":
				status = http.StatusUnauthorized
			}
			writeImportError(w, failure.Code, failure.Message, status)
		} else {
			// Database errors may embed extracted PII; keep the public error and logs
			// independent of SQL parameters and provider output.
			writeImportError(w, "import_failed", "The resume couldn't be imported. Check your library before trying again.", http.StatusInternalServerError)
		}
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func writeImportError(w http.ResponseWriter, code, message string, status int) {
	level := slog.LevelWarn
	if status >= 500 {
		level = slog.LevelError
	}
	slog.Log(context.Background(), level, "resume_import_failed", "code", code, "status", status)
	httpresponse.JSON(w, status, map[string]string{"code": code, "message": message})
}
