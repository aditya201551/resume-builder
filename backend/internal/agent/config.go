package agent

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds settings for the AI assistant only. It is loaded
// independently of internal/config.Config — see doc.go's split-out plan —
// so this package never has to know about unrelated cmd/api settings
// (OAuth, JWT, Chrome path, ...) even after it moves into its own binary.
type Config struct {
	AnthropicAPIKey string
	ClaudeModel     string
	MaxTokens       int
}

// LoadConfig reads agent settings from the environment. It returns an error
// (rather than a fatal exit) when ANTHROPIC_API_KEY is unset, so callers can
// choose to run the API without the AI assistant enabled — the same
// optionality pattern main.go already uses for SSO providers.
func LoadConfig() (*Config, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}

	model := os.Getenv("AGENT_CLAUDE_MODEL")
	if model == "" {
		model = "claude-sonnet-5"
	}

	// The chat agent emits structured edits as tool calls, and a request like
	// "build me a resume end to end" produces a lot of them in one turn. When
	// the response hits this ceiling mid-tool-call, the provider truncates the
	// arguments JSON and the tool receives an unparseable fragment — so this
	// needs enough headroom for a full turn's worth of proposals, not just a
	// conversational reply. 2048 was not close.
	maxTokens := 16384
	if v := os.Getenv("AGENT_MAX_TOKENS"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("AGENT_MAX_TOKENS must be an integer: %w", err)
		}
		maxTokens = parsed
	}

	return &Config{AnthropicAPIKey: apiKey, ClaudeModel: model, MaxTokens: maxTokens}, nil
}
