package auth

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

// SupportedProviders is the full set of SSO providers the schema's
// auth_identities.provider CHECK constraint allows — adding a new one here
// still requires that constraint to be widened via a migration.
var SupportedProviders = []string{"google", "github", "linkedin"}

type ProviderConfig struct {
	Name   string
	OAuth2 *oauth2.Config
}

// Providers holds only the providers with credentials configured — an entry
// missing from this map means that provider isn't set up yet, not an error.
type Providers map[string]*ProviderConfig

func NewProviders(redirectBaseURL, googleID, googleSecret, githubID, githubSecret string) Providers {
	providers := Providers{}

	if googleID != "" && googleSecret != "" {
		providers["google"] = &ProviderConfig{
			Name: "google",
			OAuth2: &oauth2.Config{
				ClientID:     googleID,
				ClientSecret: googleSecret,
				RedirectURL:  redirectBaseURL + "/api/auth/google/callback",
				Scopes:       []string{"openid", "email", "profile"},
				Endpoint:     endpoints.Google,
			},
		}
	}

	if githubID != "" && githubSecret != "" {
		providers["github"] = &ProviderConfig{
			Name: "github",
			OAuth2: &oauth2.Config{
				ClientID:     githubID,
				ClientSecret: githubSecret,
				RedirectURL:  redirectBaseURL + "/api/auth/github/callback",
				Scopes:       []string{"read:user", "user:email"},
				Endpoint:     endpoints.GitHub,
			},
		}
	}

	return providers
}
