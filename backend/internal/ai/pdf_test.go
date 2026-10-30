package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPDFClientInlineDocuments(t *testing.T) {
	for _, provider := range []Provider{ProviderAnthropic, ProviderOpenRouter} {
		t.Run(string(provider), func(t *testing.T) {
			pdf := []byte("%PDF-1.7\nprivate resume")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["tools"] != nil {
					t.Error("import must not have tools")
				}
				if body["model"] != "chosen-model" {
					t.Error("model not preserved")
				}
				messages := body["messages"].([]any)
				var source string
				if provider == ProviderAnthropic {
					if r.Header.Get("x-api-key") != "test-key" || r.Header.Get("anthropic-version") == "" {
						t.Error("missing Anthropic auth/version")
					}
					if body["system"] != "trusted instructions" {
						t.Error("missing system instructions")
					}
					blocks := messages[0].(map[string]any)["content"].([]any)
					document := blocks[0].(map[string]any)
					if document["type"] != "document" {
						t.Error("not a PDF document block")
					}
					source = document["source"].(map[string]any)["data"].(string)
					json.NewEncoder(w).Encode(map[string]any{"stop_reason": "end_turn", "content": []any{map[string]string{"type": "text", "text": "{\"document_type\":\"resume\"}"}}})
				} else {
					if r.Header.Get("Authorization") != "Bearer test-key" {
						t.Error("missing OpenRouter auth")
					}
					if messages[0].(map[string]any)["role"] != "system" {
						t.Error("missing trusted system message")
					}
					blocks := messages[1].(map[string]any)["content"].([]any)
					source = strings.TrimPrefix(blocks[0].(map[string]any)["file"].(map[string]any)["file_data"].(string), "data:application/pdf;base64,")
					if body["plugins"] == nil {
						t.Error("native PDF engine must be explicit")
					}
					json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]string{"content": "{\"document_type\":\"resume\"}"}}}})
				}
				decoded, err := base64.StdEncoding.DecodeString(source)
				if err != nil || string(decoded) != string(pdf) {
					t.Error("PDF bytes changed")
				}
			}))
			defer server.Close()
			client := NewPDFClient(&Config{Provider: provider, APIKey: "test-key", Model: "chosen-model"})
			client.endpoint = server.URL
			result, err := client.ParsePDF(context.Background(), "trusted instructions", pdf)
			if err != nil || !strings.Contains(result, "resume") {
				t.Fatalf("result %q: %v", result, err)
			}
		})
	}
}

func TestPDFClientRejectsTruncationAndHidesProviderBody(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{
		{200, `{"stop_reason":"max_tokens","content":[{"type":"text","text":"partial"}]}`},
		{200, `{"stop_reason":"end_turn","content":[]}`},
		{429, `sensitive resume content and credentials`},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); w.Write([]byte(tc.body)) }))
		client := NewPDFClient(&Config{Provider: ProviderAnthropic, APIKey: "test-key", Model: "model"})
		client.endpoint = server.URL
		_, err := client.ParsePDF(context.Background(), "instructions", []byte("pdf"))
		server.Close()
		if err == nil || strings.Contains(err.Error(), "sensitive") {
			t.Fatalf("unsafe/incomplete response accepted: %v", err)
		}
		if tc.status == 429 {
			var e *PDFProviderError
			if !errors.As(err, &e) || e.Status != 429 {
				t.Fatalf("lost status: %v", err)
			}
		}
	}
}
