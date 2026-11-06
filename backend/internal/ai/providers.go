package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

func ValidProvider(p Provider) bool {
	return p == ProviderOpenAI || p == ProviderAnthropic || p == ProviderOpenRouter || p == ProviderVercel
}
func BaseURL(p Provider) string {
	switch p {
	case ProviderOpenAI:
		return "https://api.openai.com/v1"
	case ProviderAnthropic:
		return "https://api.anthropic.com/v1"
	case ProviderVercel:
		return "https://ai-gateway.vercel.sh/v1"
	case ProviderOpenRouter:
		return OpenRouterBaseURL
	default:
		return ""
	}
}

type ModelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	PDF  bool   `json:"pdf"`
}

// Direct OpenAI uses the existing Chat Completions tool loop. Responses-only
// and reasoning-only models must not be offered as compatible agent models.
func CompatibleOpenAI(id string) bool {
	for _, prefix := range []string{"gpt-4o", "gpt-4.1", "gpt-5.1", "gpt-5.2", "gpt-5.3", "gpt-5.4", "gpt-5.6"} {
		if id == prefix || strings.HasPrefix(id, prefix+"-") {
			return !strings.Contains(id, "realtime") && !strings.Contains(id, "audio") && !strings.Contains(id, "codex") && !strings.Contains(id, "image") && !strings.Contains(id, "pro") && !strings.Contains(id, "chat")
		}
	}
	return false
}

func KnownNativePDF(provider Provider, id string) bool {
	if provider == ProviderOpenRouter || provider == ProviderVercel {
		if rest, ok := strings.CutPrefix(id, "anthropic/"); ok {
			return KnownNativePDF(ProviderAnthropic, rest)
		}
		if rest, ok := strings.CutPrefix(id, "openai/"); ok {
			return KnownNativePDF(ProviderOpenAI, rest)
		}
		return false
	}
	if provider == ProviderOpenAI {
		return CompatibleOpenAI(id)
	}
	if provider == ProviderAnthropic {
		for _, name := range []string{"claude-sonnet-4", "claude-opus-4", "claude-haiku-4", "claude-sonnet-5", "claude-opus-5", "claude-haiku-5", "claude-3-5-sonnet", "claude-3-7-sonnet"} {
			if strings.HasPrefix(id, name) {
				return true
			}
		}
	}
	return false
}

func DiscoverModels(ctx context.Context, cfg *Config) ([]ModelInfo, error) {
	if !ValidProvider(cfg.Provider) {
		return nil, errors.New("unsupported AI provider")
	}
	url := BaseURL(cfg.Provider) + "/models"
	if cfg.Provider == ProviderOpenRouter {
		url += "/user"
	}
	// Vercel's model catalog is public; a separate authenticated call is needed.
	if cfg.Provider == ProviderVercel {
		if err := providerGET(ctx, cfg, BaseURL(cfg.Provider)+"/credits", nil); err != nil {
			return nil, err
		}
	}
	var response struct {
		Data []struct {
			ID           string   `json:"id"`
			Name         string   `json:"name"`
			DisplayName  string   `json:"display_name"`
			Type         string   `json:"type"`
			Tags         []string `json:"tags"`
			Supported    []string `json:"supported_parameters"`
			Architecture struct {
				Input []string `json:"input_modalities"`
			} `json:"architecture"`
			Capabilities map[string]json.RawMessage `json:"capabilities"`
		} `json:"data"`
	}
	if err := providerGET(ctx, cfg, url, &response); err != nil {
		return nil, err
	}
	models := make([]ModelInfo, 0)
	for _, item := range response.Data {
		m := ModelInfo{ID: item.ID, Name: item.Name}
		if m.Name == "" {
			m.Name = item.DisplayName
		}
		if m.Name == "" {
			m.Name = m.ID
		}
		switch cfg.Provider {
		case ProviderOpenAI:
			if !CompatibleOpenAI(m.ID) {
				continue
			}
			m.PDF = true
		case ProviderAnthropic:
			if !strings.HasPrefix(m.ID, "claude-") {
				continue
			}
			// Current Claude model API advertises native PDF input. Do not infer
			// it from a generic image-input capability on an unknown model.
			var cap struct {
				Supported bool `json:"supported"`
			}
			_ = json.Unmarshal(item.Capabilities["pdf_input"], &cap)
			var flag bool
			_ = json.Unmarshal(item.Capabilities["pdf_input"], &flag)
			m.PDF = cap.Supported || flag || KnownNativePDF(cfg.Provider, m.ID)
		case ProviderOpenRouter:
			if !contains(item.Supported, "tools") {
				continue
			}
			m.PDF = contains(item.Architecture.Input, "file") || KnownNativePDF(cfg.Provider, m.ID)
		case ProviderVercel:
			if item.Type != "language" || !contains(item.Tags, "tool-use") {
				continue
			}
			m.PDF = contains(item.Tags, "file-input")
		}
		models = append(models, m)
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func providerGET(ctx context.Context, cfg *Config, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	if cfg.Provider == ProviderAnthropic {
		req.Header.Del("Authorization")
		req.Header.Set("x-api-key", cfg.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	}
	res, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return errors.New("AI provider could not be reached")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return &PDFProviderError{Status: res.StatusCode}
	}
	if out == nil {
		return nil
	}
	if json.NewDecoder(io.LimitReader(res.Body, 4*1024*1024)).Decode(out) != nil {
		return errors.New("AI provider returned an invalid model catalog")
	}
	return nil
}

// TestConnection uses synthetic input only and verifies a real streamed tool
// round trip. Callers expose the potential provider charge before invoking it.
func TestConnection(ctx context.Context, cfg *Config) error {
	m, err := NewChatModel(ctx, cfg, 512)
	if err != nil {
		return errors.New("could not initialize AI connection")
	}
	m, err = m.WithTools([]*schema.ToolInfo{{Name: "connection_check", Desc: "Verify the connection. Call with value ok.", ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{"value": {Type: schema.String, Required: true}})}})
	if err != nil {
		return err
	}
	messages := []*schema.Message{schema.SystemMessage("Call connection_check with value ok exactly once. After its result, reply OK."), schema.UserMessage("Check the connection.")}
	stream, err := m.Stream(ctx, messages)
	if err != nil {
		return errors.New("AI connection test failed; check the key, model access and balance")
	}
	var parts []*schema.Message
	for {
		part, e := stream.Recv()
		if e == io.EOF {
			break
		}
		if e != nil {
			stream.Close()
			return errors.New("AI streaming test failed")
		}
		parts = append(parts, part)
	}
	stream.Close()
	msg, err := schema.ConcatMessages(parts)
	if err != nil || msg == nil || len(msg.ToolCalls) != 1 {
		return errors.New("selected model did not complete the tool-calling test")
	}
	call := msg.ToolCalls[0]
	var args struct {
		Value string `json:"value"`
	}
	if call.Function.Name != "connection_check" || json.Unmarshal([]byte(call.Function.Arguments), &args) != nil || args.Value != "ok" {
		return errors.New("selected model returned an invalid tool call")
	}
	messages = append(messages, msg, schema.ToolMessage("ok", call.ID))
	result, err := m.Generate(ctx, messages)
	if err != nil || result == nil || strings.TrimSpace(result.Content) == "" || len(result.ToolCalls) != 0 {
		return errors.New("selected model could not complete the tool round trip")
	}
	return nil
}
