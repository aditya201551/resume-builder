package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"resume-builder/backend/internal/ai"
	"resume-builder/backend/internal/repository"
	"testing"
	"time"
)

type memoryAIStore struct {
	connections    map[string]repository.AIConnection
	prefs          map[string]*repository.AIPreference
	allowed        map[string]bool
	allowlistReads int
}

func (s *memoryAIStore) List(_ context.Context, u string) ([]repository.AIConnection, error) {
	out := []repository.AIConnection{}
	for _, c := range s.connections {
		if c.UserID == u {
			out = append(out, c)
		}
	}
	return out, nil
}
func (s *memoryAIStore) Get(_ context.Context, u, p string) (*repository.AIConnection, error) {
	c, ok := s.connections[u+"/"+p]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &c, nil
}
func (s *memoryAIStore) Save(_ context.Context, c repository.AIConnection) error {
	c.CredentialVersion = s.connections[c.UserID+"/"+c.Provider].CredentialVersion + 1
	s.connections[c.UserID+"/"+c.Provider] = c
	return nil
}
func (s *memoryAIStore) Delete(_ context.Context, u, p string) error {
	delete(s.connections, u+"/"+p)
	if pref := s.prefs[u]; pref != nil && pref.Provider == p {
		pref.Provider = ""
	}
	return nil
}
func (s *memoryAIStore) Validate(_ context.Context, u, p string, v int64) error {
	c := s.connections[u+"/"+p]
	if c.CredentialVersion != v {
		return repository.ErrNotFound
	}
	now := time.Now()
	c.ValidatedAt = &now
	s.connections[u+"/"+p] = c
	return nil
}
func (s *memoryAIStore) Preference(_ context.Context, u string) (*repository.AIPreference, error) {
	return s.prefs[u], nil
}
func (s *memoryAIStore) Select(_ context.Context, u, source, p string) error {
	s.prefs[u] = &repository.AIPreference{Source: source, Provider: p}
	return nil
}
func (s *memoryAIStore) Allowed(_ context.Context, u string) (bool, error) {
	s.allowlistReads++
	return s.allowed[u], nil
}
func newTestAIService(t *testing.T) (*AISettingsService, *memoryAIStore) {
	t.Helper()
	t.Setenv("AI_ENCRYPTION_KEYS", `{"v1":"`+base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))+`"}`)
	t.Setenv("AI_ENCRYPTION_ACTIVE_KEY", "v1")
	vault, err := ai.LoadVault()
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryAIStore{connections: map[string]repository.AIConnection{}, prefs: map[string]*repository.AIPreference{}, allowed: map[string]bool{}}
	svc := NewAISettingsService(store, vault, &ai.Config{Provider: ai.ProviderAnthropic, APIKey: "platform-synthetic", Model: "claude-sonnet-4"}, "")
	svc.discover = func(context.Context, *ai.Config) ([]ai.ModelInfo, error) {
		return []ai.ModelInfo{{ID: "chat", PDF: false}, {ID: "pdf", PDF: true}}, nil
	}
	svc.test = func(context.Context, *ai.Config) error { return nil }
	return svc, store
}
func TestAIFlagAndInviteCombinations(t *testing.T) {
	svc, store := newTestAIService(t)
	ctx := context.Background()
	for _, platform := range []bool{false, true} {
		for _, byok := range []bool{false, true} {
			for _, allowed := range []bool{false, true} {
				store.allowed["owner"] = allowed
				for _, public := range []bool{false, true} {
					flags := AIAvailability{Platform: platform, BYOK: byok, PlatformPublic: public}
					settings, err := svc.Settings(ctx, "owner", flags)
					if err != nil {
						t.Fatal(err)
					}
					expected := platform && (public || allowed)
					if settings.PlatformAvailable != expected || settings.BYOKAvailable != byok || settings.Ready != expected {
						t.Fatalf("wrong access: platform=%v public=%v byok=%v invited=%v: %+v", platform, public, byok, allowed, settings)
					}
					for _, pdf := range []bool{false, true} {
						_, err = svc.Resolve(ctx, "owner", flags, pdf)
						if (err == nil) != expected {
							t.Fatalf("request gate disagrees with availability: flags=%+v invited=%v pdf=%v err=%v", flags, allowed, pdf, err)
						}
					}
				}
			}
		}
	}
}
func TestBYOKLifecycleIsolationAndNoFallback(t *testing.T) {
	svc, store := newTestAIService(t)
	ctx := context.Background()
	flags := AIAvailability{Platform: true, BYOK: true}
	store.allowed["owner"] = true
	if err := svc.Save(ctx, "owner", "anthropic", "personal-synthetic-key", "pdf", flags); err != nil {
		t.Fatal(err)
	}
	if err := svc.Select(ctx, "owner", "byok", "anthropic", flags); err == nil {
		t.Fatal("untested connection selected")
	}
	if err := svc.Test(ctx, "owner", "anthropic", flags); err != nil {
		t.Fatal(err)
	}
	if err := svc.Select(ctx, "owner", "byok", "anthropic", flags); err != nil {
		t.Fatal(err)
	}
	cfg, err := svc.Resolve(ctx, "owner", flags, true)
	if err != nil || cfg.APIKey != "personal-synthetic-key" {
		t.Fatal("personal key not resolved")
	}
	if _, err = svc.Resolve(ctx, "owner", AIAvailability{Platform: true}, false); err == nil {
		t.Fatal("disabled BYOK fell back to platform")
	}
	store.allowed["owner"] = false
	if _, err = svc.Resolve(ctx, "owner", flags, false); err != nil {
		t.Fatal("BYOK depends on invitation")
	}
	if _, _, err := svc.connection(ctx, "other-user", "anthropic"); err == nil {
		t.Fatal("another account's connection exposed")
	}
	safe, _ := svc.Settings(ctx, "owner", flags)
	encoded, _ := json.Marshal(safe)
	if bytes.Contains(encoded, []byte("personal-synthetic-key")) || bytes.Contains(encoded, []byte("encrypted_key")) {
		t.Fatal("secret leaked in settings")
	}
	if err = svc.Save(ctx, "owner", "anthropic", "", "chat", flags); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Resolve(ctx, "owner", flags, false); err == nil {
		t.Fatal("model change retained validation")
	}
	if err = svc.Test(ctx, "owner", "anthropic", flags); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Resolve(ctx, "owner", flags, true); err == nil {
		t.Fatal("chat-only model accepted PDF")
	}
	store.allowed["owner"] = true
	if err = svc.Delete(ctx, "owner", "anthropic"); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Resolve(ctx, "owner", flags, false); err == nil {
		t.Fatal("deletion fell back to platform")
	}
	settings, _ := svc.Settings(ctx, "owner", flags)
	if settings.Source != "byok" || settings.Ready {
		t.Fatal("explicit source lost after deletion")
	}
}

func TestPublicPlatformPolicyPreservesPreferencesAndRestoresInvites(t *testing.T) {
	svc, store := newTestAIService(t)
	ctx := context.Background()
	flags := AIAvailability{Platform: true, PlatformPublic: true}
	if _, err := svc.Resolve(ctx, "", flags, false); err == nil {
		t.Fatal("public access permitted an unauthenticated caller")
	}
	settings, err := svc.Settings(ctx, "new-user", flags)
	if err != nil || !settings.Ready || settings.Source != "platform" || store.allowlistReads != 0 {
		t.Fatal("public access did not default to platform without an invitation", err)
	}
	if err = svc.Select(ctx, "new-user", "platform", "", flags); err != nil {
		t.Fatal(err)
	}
	flags.PlatformPublic = false
	if _, err = svc.Resolve(ctx, "new-user", flags, false); err == nil {
		t.Fatal("public access survived invite-only policy")
	}
	store.allowed["new-user"] = true
	if _, err = svc.Resolve(ctx, "new-user", flags, false); err != nil {
		t.Fatal("restoring invite-only lost invitation", err)
	}
	flags.PlatformPublic = true
	store.prefs["new-user"] = &repository.AIPreference{Source: "byok"}
	settings, err = svc.Settings(ctx, "new-user", flags)
	if err != nil || settings.Source != "byok" || settings.Ready {
		t.Fatal("public access overwrote explicit BYOK preference", err)
	}
	svc.platform = nil
	delete(store.prefs, "new-user")
	settings, err = svc.Settings(ctx, "new-user", flags)
	if err != nil || settings.PlatformAvailable || settings.Ready {
		t.Fatal("public access bypassed missing platform configuration", err)
	}
}
func TestDisabledBYOKNeverCallsProviderAndStaleTestCannotValidate(t *testing.T) {
	svc, store := newTestAIService(t)
	ctx := context.Background()
	svc.discover = func(context.Context, *ai.Config) ([]ai.ModelInfo, error) {
		t.Fatal("disabled BYOK contacted provider")
		return nil, nil
	}
	if err := svc.Save(ctx, "owner", "openai", "synthetic-key", "model", AIAvailability{}); err == nil {
		t.Fatal("disabled save allowed")
	}
	svc, store = newTestAIService(t)
	flags := AIAvailability{BYOK: true}
	if err := svc.Save(ctx, "owner", "openai", "synthetic-key", "pdf", flags); err != nil {
		t.Fatal(err)
	}
	svc.test = func(context.Context, *ai.Config) error {
		c := store.connections["owner/openai"]
		c.CredentialVersion++
		store.connections["owner/openai"] = c
		return nil
	}
	var failure *AIError
	if err := svc.Test(ctx, "owner", "openai", flags); !errors.As(err, &failure) || failure.Code != "connection_changed" {
		t.Fatal("stale test validated a rotated key", err)
	}
}
