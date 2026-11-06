package ai

import (
	"fmt"
	"os"
)

type Provider string

const (
	ProviderAnthropic  Provider = "anthropic"
	ProviderOpenRouter Provider = "openrouter"
	ProviderOpenAI     Provider = "openai"
	ProviderVercel     Provider = "vercel"
)

type Config struct {
	Provider Provider
	APIKey   string `json:"-"`
	Model    string
}

func LoadConfig() (*Config, error) {
	provider := Provider(os.Getenv("AI_PROVIDER"))
	switch provider {
	case ProviderAnthropic, ProviderOpenRouter, ProviderOpenAI, ProviderVercel:
	case "":
		return nil, fmt.Errorf("AI_PROVIDER is not set")
	default:
		return nil, fmt.Errorf("AI_PROVIDER is not supported")
	}

	apiKey := os.Getenv("AI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("AI_API_KEY is not set")
	}

	model := os.Getenv("AI_MODEL")
	if model == "" {
		return nil, fmt.Errorf("AI_MODEL is not set")
	}
	if provider == ProviderOpenAI && !CompatibleOpenAI(model) {
		return nil, fmt.Errorf("AI_MODEL is not a supported Chat Completions tool model")
	}

	return &Config{Provider: provider, APIKey: apiKey, Model: model}, nil
}
