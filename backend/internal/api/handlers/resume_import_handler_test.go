package handlers

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/repository"
	"resume-builder/backend/internal/service"
)

type fakeResumeImporter struct {
	calls          int
	user, filename string
	err            error
}

func (s *fakeResumeImporter) Import(_ context.Context, user, name string, pdf []byte) (*service.ResumeImportResult, error) {
	s.calls++
	s.user = user
	s.filename = name
	if s.err != nil {
		return nil, s.err
	}
	return &service.ResumeImportResult{Resume: &repository.Resume{ID: "imported", UserID: user}, Warnings: []string{}}, nil
}

func importRequest(t *testing.T, name string, content []byte, extra bool) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	file.Write(content)
	if extra {
		other, _ := form.CreateFormFile("file", "another.pdf")
		other.Write([]byte("%PDF"))
	}
	form.Close()
	r := httptest.NewRequest(http.MethodPost, "/api/resumes/import", &body)
	r.Header.Set("Content-Type", form.FormDataContentType())
	return r.WithContext(auth.ContextWithUserID(r.Context(), "signed-in-user"))
}

func TestResumeImportHTTP(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request func(*testing.T) *http.Request
		failure error
		status  int
		calls   int
	}{
		{"success", func(t *testing.T) *http.Request { return importRequest(t, "resume.pdf", []byte("%PDF"), false) }, nil, 201, 1},
		{"extra file", func(t *testing.T) *http.Request { return importRequest(t, "resume.pdf", []byte("%PDF"), true) }, nil, 400, 0},
		{"too large", func(t *testing.T) *http.Request {
			return importRequest(t, "resume.pdf", make([]byte, service.MaxResumePDFBytes+1), false)
		}, nil, 413, 0},
		{"not a resume", func(t *testing.T) *http.Request { return importRequest(t, "invoice.pdf", []byte("%PDF"), false) }, &service.ImportError{Code: "not_resume", Message: "This is not a resume."}, 422, 1},
		{"no auth", func(t *testing.T) *http.Request {
			return httptest.NewRequest("POST", "/api/resumes/import", strings.NewReader(""))
		}, nil, 401, 0},
		{"not multipart", func(t *testing.T) *http.Request {
			r := httptest.NewRequest("POST", "/api/resumes/import", strings.NewReader(`{}`))
			return r.WithContext(auth.ContextWithUserID(r.Context(), "user"))
		}, nil, 400, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeResumeImporter{err: tc.failure}
			response := httptest.NewRecorder()
			NewResumeImportHandler(svc).Import(response, tc.request(t))
			if response.Code != tc.status || svc.calls != tc.calls {
				t.Fatalf("HTTP %d, calls %d, body %s", response.Code, svc.calls, response.Body.String())
			}
			if tc.name == "success" && (svc.user != "signed-in-user" || svc.filename != "resume.pdf") {
				t.Fatal("lost ownership or filename")
			}
		})
	}
}
