package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string

	AppEnv               string
	FrontendURL          string
	OAuthRedirectBaseURL string
	JWTSecret            string
	JWTTTL               time.Duration
	GoogleClientID       string
	GoogleClientSecret   string
	GithubClientID       string
	GithubClientSecret   string
}

func (c *Config) CookieSecure() bool {
	return c.AppEnv == "production"
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	redirectBaseURL := os.Getenv("OAUTH_REDIRECT_BASE_URL")
	if redirectBaseURL == "" {
		redirectBaseURL = "http://localhost:" + port
	}

	ttlHours := 168
	if v := os.Getenv("JWT_TTL_HOURS"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("JWT_TTL_HOURS must be an integer: %w", err)
		}
		ttlHours = parsed
	}

	return &Config{
		Port:                 port,
		DatabaseURL:          dbURL,
		AppEnv:               appEnv,
		FrontendURL:          frontendURL,
		OAuthRedirectBaseURL: redirectBaseURL,
		JWTSecret:            jwtSecret,
		JWTTTL:               time.Duration(ttlHours) * time.Hour,
		GoogleClientID:       os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:   os.Getenv("GOOGLE_CLIENT_SECRET"),
		GithubClientID:       os.Getenv("GITHUB_CLIENT_ID"),
		GithubClientSecret:   os.Getenv("GITHUB_CLIENT_SECRET"),
	}, nil
}
