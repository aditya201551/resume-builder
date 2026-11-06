package ai

import (
	"context"
	"github.com/cloudwego/eino/schema"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAnthropicStreamToolArguments(t *testing.T) {
	events := []string{
		`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":5,"output_tokens":0}}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"call_1","name":"check","input":{}}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"value\":"}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"ok\"}"}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tokens":12}}`,
		`{"type":"message_stop"}`,
	}
	types := []string{"message_start", "content_block_start", "content_block_delta", "content_block_delta", "content_block_stop", "message_delta", "message_stop"}
	var streamBody strings.Builder
	for i, event := range events {
		streamBody.WriteString("event: " + types[i] + "\ndata: " + event + "\n\n")
	}
	client := &http.Client{Transport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != BaseURL(ProviderAnthropic)+"/messages" || r.Header.Get("x-api-key") != "synthetic-key" {
			t.Error("wrong Anthropic endpoint or credentials")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(streamBody.String()))}, nil
	})}
	m, err := newChatModel(context.Background(), &Config{Provider: ProviderAnthropic, APIKey: "synthetic-key", Model: "claude-sonnet-4"}, 512, client)
	if err != nil {
		t.Fatal(err)
	}
	m, err = m.WithTools([]*schema.ToolInfo{{Name: "check", ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{"value": {Type: schema.String}})}})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := m.Stream(context.Background(), []*schema.Message{schema.UserMessage("synthetic check")})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var parts []*schema.Message
	for {
		part, e := stream.Recv()
		if e == io.EOF {
			break
		}
		if e != nil {
			t.Fatal(e)
		}
		parts = append(parts, part)
	}
	msg, err := schema.ConcatMessages(parts)
	if err != nil || len(msg.ToolCalls) != 1 || msg.ToolCalls[0].Function.Arguments != `{"value":"ok"}` {
		t.Fatal("Anthropic tool argument reconstruction failed", err)
	}
}
