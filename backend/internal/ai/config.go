package ai

import (
	"fmt"
	"os"
)

type Provider string

const (
	ProviderAnthropic  Provider = "anthropic"
	ProviderOpenRouter Provider = "openrouter"
)

type Config struct {
	Provider Provider
	APIKey   string
	Model    string
}

func LoadConfig() (*Config, error) {
	provider := Provider(os.Getenv("AI_PROVIDER"))
	switch provider {
	case ProviderAnthropic, ProviderOpenRouter:
	case "":
		return nil, fmt.Errorf("AI_PROVIDER is not set (want %q or %q)", ProviderAnthropic, ProviderOpenRouter)
	default:
		return nil, fmt.Errorf("AI_PROVIDER %q is not supported (want %q or %q)", provider, ProviderAnthropic, ProviderOpenRouter)
	}

	apiKey := os.Getenv("AI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("AI_API_KEY is not set")
	}

	model := os.Getenv("AI_MODEL")
	if model == "" {
		return nil, fmt.Errorf("AI_MODEL is not set")
	}

	return &Config{Provider: provider, APIKey: apiKey, Model: model}, nil
}
