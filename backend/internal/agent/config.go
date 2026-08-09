package agent

import (
	"fmt"
	"os"
	"strconv"

	"resume-builder/backend/internal/ai"
)

type Config struct {
	AI        *ai.Config
	MaxTokens int
}

func LoadConfig() (*Config, error) {
	aiCfg, err := ai.LoadConfig()
	if err != nil {
		return nil, err
	}

	if model := os.Getenv("AGENT_MODEL"); model != "" {
		aiCfg.Model = model
	}

	maxTokens := 16384
	if v := os.Getenv("AGENT_MAX_TOKENS"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("AGENT_MAX_TOKENS must be an integer: %w", err)
		}
		maxTokens = parsed
	}

	return &Config{AI: aiCfg, MaxTokens: maxTokens}, nil
}
