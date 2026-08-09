package ai

import "testing"

func setEnv(t *testing.T, provider, apiKey, model string) {
	t.Helper()
	t.Setenv("AI_PROVIDER", provider)
	t.Setenv("AI_API_KEY", apiKey)
	t.Setenv("AI_MODEL", model)
}

func TestLoadConfig(t *testing.T) {
	setEnv(t, "openrouter", "sk-or-v1-test", "anthropic/claude-sonnet-4")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Provider != ProviderOpenRouter {
		t.Errorf("Provider = %q, want %q", cfg.Provider, ProviderOpenRouter)
	}
	if cfg.APIKey != "sk-or-v1-test" {
		t.Errorf("APIKey = %q", cfg.APIKey)
	}
	if cfg.Model != "anthropic/claude-sonnet-4" {
		t.Errorf("Model = %q", cfg.Model)
	}
}

func TestLoadConfig_Errors(t *testing.T) {
	cases := map[string]struct{ provider, apiKey, model string }{
		"missing provider": {"", "key", "model"},
		"unknown provider": {"openai", "key", "model"},
		"missing api key":  {"anthropic", "", "model"},
		"missing model":    {"anthropic", "key", ""},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			setEnv(t, tc.provider, tc.apiKey, tc.model)
			if _, err := LoadConfig(); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func TestChatModelOptions(t *testing.T) {
	if got := (&Config{Provider: ProviderAnthropic}).ChatModelOptions(); len(got) != 1 {
		t.Errorf("anthropic: got %d options, want 1", len(got))
	}
	if got := (&Config{Provider: ProviderOpenRouter}).ChatModelOptions(); len(got) != 0 {
		t.Errorf("openrouter: got %d options, want 0", len(got))
	}
}
