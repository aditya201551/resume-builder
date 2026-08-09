package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

type Profile struct {
	ProviderUserID string
	Email          string
	EmailVerified  bool
	Name           string
	AvatarURL      string
}

func FetchProfile(ctx context.Context, provider string, cfg *oauth2.Config, token *oauth2.Token) (*Profile, error) {
	client := cfg.Client(ctx, token)

	switch provider {
	case "google":
		return fetchGoogleProfile(ctx, client)
	case "github":
		return fetchGitHubProfile(ctx, client)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func fetchGoogleProfile(ctx context.Context, client *http.Client) (*Profile, error) {
	var body struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := getJSON(ctx, client, "https://openidconnect.googleapis.com/v1/userinfo", &body); err != nil {
		return nil, err
	}

	return &Profile{
		ProviderUserID: body.Sub,
		Email:          body.Email,
		EmailVerified:  body.EmailVerified,
		Name:           body.Name,
		AvatarURL:      body.Picture,
	}, nil
}

func fetchGitHubProfile(ctx context.Context, client *http.Client) (*Profile, error) {
	var user struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := getJSON(ctx, client, "https://api.github.com/user", &user); err != nil {
		return nil, err
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := getJSON(ctx, client, "https://api.github.com/user/emails", &emails); err != nil {
		return nil, err
	}

	var email string
	var verified bool
	for _, e := range emails {
		if e.Primary {
			email, verified = e.Email, e.Verified
			break
		}
	}
	if email == "" && len(emails) > 0 {
		email, verified = emails[0].Email, emails[0].Verified
	}

	name := user.Name
	if name == "" {
		name = user.Login
	}

	return &Profile{
		ProviderUserID: fmt.Sprintf("%d", user.ID),
		Email:          email,
		EmailVerified:  verified,
		Name:           name,
		AvatarURL:      user.AvatarURL,
	}, nil
}

func getJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
