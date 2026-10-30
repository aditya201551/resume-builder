package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PDFClient sends a one-shot inline document request without Files API storage,
// tools or chat history. Eino's current chat adapter does not support PDF blocks.
type PDFClient struct {
	config   Config
	client   *http.Client
	endpoint string
}

type PDFProviderError struct{ Status int }

func (e *PDFProviderError) Error() string {
	return fmt.Sprintf("PDF provider returned HTTP %d", e.Status)
}

func NewPDFClient(cfg *Config) *PDFClient {
	endpoint := "https://api.anthropic.com/v1/messages"
	if cfg.Provider == ProviderOpenRouter {
		endpoint = OpenRouterBaseURL + "/chat/completions"
	}
	return &PDFClient{config: *cfg, client: &http.Client{Timeout: 110 * time.Second}, endpoint: endpoint}
}

func (c *PDFClient) ParsePDF(ctx context.Context, instructions string, pdf []byte) (string, error) {
	data := base64.StdEncoding.EncodeToString(pdf)
	var body map[string]any
	if c.config.Provider == ProviderAnthropic {
		body = map[string]any{"model": c.config.Model, "max_tokens": 12000, "system": instructions, "messages": []any{
			map[string]any{"role": "user", "content": []any{
				map[string]any{"type": "document", "source": map[string]string{"type": "base64", "media_type": "application/pdf", "data": data}},
				map[string]string{"type": "text", "text": "Classify this PDF and extract its resume content according to the supplied JSON contract."},
			}},
		}}
	} else if c.config.Provider == ProviderOpenRouter {
		body = map[string]any{"model": c.config.Model, "max_tokens": 12000, "messages": []any{
			map[string]any{"role": "system", "content": instructions},
			map[string]any{"role": "user", "content": []any{
				map[string]any{"type": "file", "file": map[string]string{"filename": "resume.pdf", "file_data": "data:application/pdf;base64," + data}},
				map[string]string{"type": "text", "text": "Classify this PDF and extract its resume content according to the supplied JSON contract."},
			}},
		}, "plugins": []any{map[string]any{"id": "file-parser", "pdf": map[string]string{"engine": "native"}}}}
	} else {
		return "", fmt.Errorf("unsupported PDF provider %q", c.config.Provider)
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(encoded))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.config.Provider == ProviderAnthropic {
		req.Header.Set("x-api-key", c.config.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("PDF request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", &PDFProviderError{Status: res.StatusCode}
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1024*1024+1))
	if err != nil {
		return "", err
	}
	if len(raw) > 1024*1024 {
		return "", fmt.Errorf("PDF extraction response too large")
	}
	if c.config.Provider == ProviderAnthropic {
		var result struct {
			StopReason string `json:"stop_reason"`
			Content    []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if err = json.Unmarshal(raw, &result); err != nil {
			return "", err
		}
		if result.StopReason != "end_turn" {
			return "", fmt.Errorf("PDF extraction did not complete")
		}
		var text strings.Builder
		for _, block := range result.Content {
			if block.Type == "text" {
				text.WriteString(block.Text)
			}
		}
		if text.Len() == 0 {
			return "", fmt.Errorf("empty PDF extraction")
		}
		return text.String(), nil
	}
	var result struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err = json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if len(result.Choices) != 1 || result.Choices[0].FinishReason != "stop" || result.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("PDF extraction did not complete")
	}
	return result.Choices[0].Message.Content, nil
}
