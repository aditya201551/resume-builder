package ai

import (
	"context"
	"testing"
)

func TestNewChatModel(t *testing.T) {
	for _, provider := range []Provider{ProviderAnthropic, ProviderOpenRouter} {
		t.Run(string(provider), func(t *testing.T) {
			cfg := &Config{Provider: provider, APIKey: "test-key", Model: "test-model"}
			chatModel, err := NewChatModel(context.Background(), cfg, 1024)
			if err != nil {
				t.Fatalf("NewChatModel: %v", err)
			}
			if chatModel == nil {
				t.Fatal("NewChatModel returned a nil model")
			}
		})
	}
}

func TestNewChatModel_UnsupportedProvider(t *testing.T) {
	cfg := &Config{Provider: Provider("openai"), APIKey: "test-key", Model: "test-model"}
	if _, err := NewChatModel(context.Background(), cfg, 1024); err == nil {
		t.Error("expected an error for an unsupported provider, got nil")
	}
}
