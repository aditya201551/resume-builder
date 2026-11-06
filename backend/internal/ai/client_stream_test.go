package ai

import (
	"context"
	"encoding/json"
	"github.com/cloudwego/eino/schema"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fixtureTransport func(*http.Request) (*http.Response, error)

func (f fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestCompatibleAdaptersStreamToolsAndDetectEmbeddedErrors(t *testing.T) {
	for _, provider := range []Provider{ProviderOpenAI, ProviderOpenRouter, ProviderVercel} {
		t.Run(string(provider), func(t *testing.T) {
			failed := false
			client := &http.Client{Transport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != BaseURL(provider)+"/chat/completions" || r.Header.Get("Authorization") != "Bearer synthetic-key" {
					t.Error("wrong endpoint or credential")
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["max_completion_tokens"] != float64(512) || body["max_tokens"] != nil || body["store"] != false || body["tools"] == nil {
					t.Error("incorrect tool request", body)
				}
				if provider == ProviderOpenAI && body["reasoning_effort"] != "none" {
					t.Error("missing compatible reasoning setting")
				}
				data := `data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"check","arguments":"{\"value\":"}}]}}]}` + "\n\n" +
					`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"ok\"}"}}]},"finish_reason":"tool_calls"}]}` + "\n\n"
				if failed {
					data = `data: {"id":"1","error":{"message":"private provider body"},"choices":[{"index":0,"delta":{},"finish_reason":"error"}]}` + "\n\n"
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(data + "data: [DONE]\n\n"))}, nil
			})}
			m, err := newChatModel(context.Background(), &Config{Provider: provider, APIKey: "synthetic-key", Model: "gpt-5.4"}, 512, client)
			if err != nil {
				t.Fatal(err)
			}
			m, err = m.WithTools([]*schema.ToolInfo{{Name: "check", ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{"value": {Type: schema.String}})}})
			if err != nil {
				t.Fatal(err)
			}
			stream, err := m.Stream(context.Background(), []*schema.Message{schema.UserMessage("test")})
			if err != nil {
				t.Fatal(err)
			}
			var parts []*schema.Message
			for {
				msg, e := stream.Recv()
				if e == io.EOF {
					break
				}
				if e != nil {
					t.Fatal(e)
				}
				parts = append(parts, msg)
			}
			stream.Close()
			combined, err := schema.ConcatMessages(parts)
			if err != nil || len(combined.ToolCalls) != 1 || combined.ToolCalls[0].Function.Arguments != `{"value":"ok"}` {
				t.Fatal("tool stream lost arguments", err)
			}
			failed = true
			stream, err = m.Stream(context.Background(), []*schema.Message{schema.UserMessage("test")})
			if err != nil {
				t.Fatal(err)
			}
			defer stream.Close()
			_, err = stream.Recv()
			if err == nil || strings.Contains(err.Error(), "private provider body") {
				t.Fatal("embedded SSE failure not sanitized", err)
			}
		})
	}
}
