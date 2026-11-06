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

func LoadMaxTokens() (int, error) {
	maxTokens := 16384
	if v := os.Getenv("AGENT_MAX_TOKENS"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("AGENT_MAX_TOKENS must be an integer: %w", err)
		}
		maxTokens = parsed
	}

	if maxTokens < 512 || maxTokens > 32768 {
		return 0, fmt.Errorf("AGENT_MAX_TOKENS must be between 512 and 32768")
	}
	return maxTokens, nil
}
