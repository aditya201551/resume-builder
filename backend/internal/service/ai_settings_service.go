package service

import (
	"context"
	"errors"
	"resume-builder/backend/internal/ai"
	"resume-builder/backend/internal/repository"
	"strings"
)

type AIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *AIError) Error() string                     { return e.Message }
func aiError(code, message string, status int) error { return &AIError{code, message, status} }

type AIAvailability struct {
	Platform       bool
	BYOK           bool
	PlatformPublic bool
}
type AISettings struct {
	PlatformAvailable bool                      `json:"platform_available"`
	BYOKAvailable     bool                      `json:"byok_available"`
	Connections       []repository.AIConnection `json:"connections"`
	Source            string                    `json:"source"`
	Provider          string                    `json:"provider"`
	Model             string                    `json:"model"`
	Ready             bool                      `json:"ready"`
	PDFSupported      bool                      `json:"pdf_supported"`
	Reason            string                    `json:"reason"`
}
type aiSettingsStore interface {
	List(context.Context, string) ([]repository.AIConnection, error)
	Get(context.Context, string, string) (*repository.AIConnection, error)
	Save(context.Context, repository.AIConnection) error
	Delete(context.Context, string, string) error
	Validate(context.Context, string, string, int64) error
	Preference(context.Context, string) (*repository.AIPreference, error)
	Select(context.Context, string, string, string) error
	Allowed(context.Context, string) (bool, error)
}
type AISettingsService struct {
	store      aiSettingsStore
	vault      *ai.Vault
	platform   *ai.Config
	agentModel string
	discover   func(context.Context, *ai.Config) ([]ai.ModelInfo, error)
	test       func(context.Context, *ai.Config) error
}

func NewAISettingsService(store aiSettingsStore, vault *ai.Vault, platform *ai.Config, agentModel string) *AISettingsService {
	return &AISettingsService{store: store, vault: vault, platform: platform, agentModel: agentModel, discover: ai.DiscoverModels, test: ai.TestConnection}
}
func (s *AISettingsService) Settings(ctx context.Context, user string, flags AIAvailability) (*AISettings, error) {
	if user == "" {
		return nil, aiError("unauthorized", "Sign in to use AI connections.", 401)
	}
	connections, err := s.store.List(ctx, user)
	if err != nil {
		return nil, err
	}
	pref, err := s.store.Preference(ctx, user)
	if err != nil {
		return nil, err
	}
	allowed := flags.PlatformPublic
	if flags.Platform && s.platform != nil && !allowed {
		allowed, err = s.store.Allowed(ctx, user)
		if err != nil {
			return nil, err
		}
	}
	out := &AISettings{Connections: connections, PlatformAvailable: flags.Platform && allowed && s.platform != nil, BYOKAvailable: flags.BYOK && s.vault != nil, Reason: "setup_required"}
	if pref == nil {
		if out.PlatformAvailable {
			out.Source = "platform"
		}
	} else {
		out.Source = pref.Source
		out.Provider = pref.Provider
	}
	if out.Source == "platform" {
		if out.PlatformAvailable {
			out.Ready = true
			out.PDFSupported = ai.KnownNativePDF(s.platform.Provider, s.platform.Model)
			out.Provider = string(s.platform.Provider)
			out.Model = s.platform.Model
			if s.agentModel != "" {
				out.Model = s.agentModel
			}
			out.Reason = ""
		} else {
			out.Reason = "platform_unavailable"
		}
	} else if out.Source == "byok" {
		if !out.BYOKAvailable {
			out.Reason = "byok_unavailable"
			return out, nil
		}
		for _, c := range connections {
			if c.Provider == out.Provider {
				out.Model = c.Model
				out.PDFSupported = c.PDFSupported
				out.Ready = c.Model != "" && c.ValidatedAt != nil
				if out.Ready {
					out.Reason = ""
				}
			}
		}
	}
	if !out.PlatformAvailable && !out.BYOKAvailable {
		out.Reason = "ai_unavailable"
	}
	return out, nil
}
func (s *AISettingsService) RequireBYOK(flags AIAvailability) error {
	if !flags.BYOK {
		return aiError("byok_disabled", "Personal AI connections are currently disabled.", 403)
	}
	if s.vault == nil {
		return aiError("byok_unavailable", "Personal AI connections are not configured on this server.", 503)
	}
	return nil
}
func (s *AISettingsService) connection(ctx context.Context, user, provider string) (*ai.Config, *repository.AIConnection, error) {
	if !ai.ValidProvider(ai.Provider(provider)) {
		return nil, nil, aiError("invalid_provider", "Choose a supported AI provider.", 400)
	}
	c, err := s.store.Get(ctx, user, provider)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, aiError("connection_missing", "Configure this AI connection first.", 409)
	}
	if err != nil {
		return nil, nil, err
	}
	key, err := s.vault.Open(c.EncryptedKey, c.EncryptionVersion, user, provider)
	if err != nil {
		return nil, nil, aiError("connection_unavailable", "This key cannot be opened. Replace it in AI connections.", 503)
	}
	return &ai.Config{Provider: ai.Provider(provider), APIKey: key, Model: c.Model}, c, nil
}
func (s *AISettingsService) Resolve(ctx context.Context, user string, flags AIAvailability, pdf bool) (*ai.Config, error) {
	settings, err := s.Settings(ctx, user, flags)
	if err != nil {
		return nil, err
	}
	if !settings.Ready {
		status := 409
		if settings.Reason == "platform_unavailable" || settings.Reason == "byok_unavailable" || settings.Reason == "ai_unavailable" {
			status = 403
		}
		return nil, aiError(settings.Reason, "Choose an available AI connection in AI connections.", status)
	}
	if settings.Source == "platform" {
		if pdf && !settings.PDFSupported {
			return nil, aiError("model_unsupported_for_pdf", "The platform model does not have supported native PDF input.", 422)
		}
		cfg := *s.platform
		if !pdf && s.agentModel != "" {
			cfg.Model = s.agentModel
		}
		return &cfg, nil
	}
	cfg, c, err := s.connection(ctx, user, settings.Provider)
	if err != nil {
		return nil, err
	}
	if c.ValidatedAt == nil || cfg.Model == "" {
		return nil, aiError("setup_required", "Test this connection before using it.", 409)
	}
	if pdf && !c.PDFSupported {
		return nil, aiError("model_unsupported_for_pdf", "Choose a model with native PDF support to import a resume.", 422)
	}
	return cfg, nil
}
func (s *AISettingsService) Save(ctx context.Context, user, provider, key, model string, flags AIAvailability) error {
	if err := s.RequireBYOK(flags); err != nil {
		return err
	}
	if !ai.ValidProvider(ai.Provider(provider)) {
		return aiError("invalid_provider", "Choose a supported AI provider.", 400)
	}
	key = strings.TrimSpace(key)
	model = strings.TrimSpace(model)
	if len(key) > 4096 || len(model) > 200 || strings.ContainsAny(key, "\r\n") {
		return aiError("invalid_connection", "Invalid API key or model.", 400)
	}
	if key == "" {
		cfg, _, err := s.connection(ctx, user, provider)
		if err != nil {
			return err
		}
		key = cfg.APIKey
	}
	if len(key) < 8 {
		return aiError("invalid_key", "Enter a valid API key.", 400)
	}
	pdf := false
	if model != "" {
		models, err := s.discover(ctx, &ai.Config{Provider: ai.Provider(provider), APIKey: key})
		if err != nil {
			return aiError("discovery_failed", "Could not verify this key and model. Check provider access and balance.", 422)
		}
		found := false
		for _, m := range models {
			if m.ID == model {
				found = true
				pdf = m.PDF
				break
			}
		}
		if !found {
			return aiError("model_unsupported", "Choose a compatible model from the model list.", 422)
		}
	}
	encrypted, version, err := s.vault.Seal(key, user, provider)
	if err != nil {
		return err
	}
	return s.store.Save(ctx, repository.AIConnection{UserID: user, Provider: provider, Model: model, EncryptedKey: encrypted, EncryptionVersion: version, KeySuffix: key[len(key)-4:], PDFSupported: pdf})
}
func (s *AISettingsService) Models(ctx context.Context, user, provider string, flags AIAvailability) ([]ai.ModelInfo, error) {
	if err := s.RequireBYOK(flags); err != nil {
		return nil, err
	}
	cfg, _, err := s.connection(ctx, user, provider)
	if err != nil {
		return nil, err
	}
	models, err := s.discover(ctx, cfg)
	if err != nil {
		return nil, aiError("discovery_failed", "Could not load models. Check the key, provider access and balance.", 422)
	}
	return models, nil
}
func (s *AISettingsService) Test(ctx context.Context, user, provider string, flags AIAvailability) error {
	if err := s.RequireBYOK(flags); err != nil {
		return err
	}
	cfg, c, err := s.connection(ctx, user, provider)
	if err != nil {
		return err
	}
	if cfg.Model == "" {
		return aiError("model_required", "Choose and save a model first.", 400)
	}
	if err = s.test(ctx, cfg); err != nil {
		return aiError("connection_test_failed", "Connection test failed. Check the key, model tool support and provider balance.", 422)
	}
	if err = s.store.Validate(ctx, user, provider, c.CredentialVersion); errors.Is(err, repository.ErrNotFound) {
		return aiError("connection_changed", "The connection changed during the test. Test it again.", 409)
	}
	return err
}
func (s *AISettingsService) Select(ctx context.Context, user, source, provider string, flags AIAvailability) error {
	if source == "platform" {
		settings, err := s.Settings(ctx, user, flags)
		if err != nil {
			return err
		}
		if !settings.PlatformAvailable {
			return aiError("platform_unavailable", "Platform AI is not available to this account.", 403)
		}
	} else if source == "byok" {
		if err := s.RequireBYOK(flags); err != nil {
			return err
		}
		_, c, err := s.connection(ctx, user, provider)
		if err != nil {
			return err
		}
		if c.ValidatedAt == nil || c.Model == "" {
			return aiError("setup_required", "Choose a model and test the connection first.", 409)
		}
	} else {
		return aiError("invalid_source", "Choose Platform AI or a personal connection.", 400)
	}
	return s.store.Select(ctx, user, source, provider)
}
func (s *AISettingsService) Delete(ctx context.Context, user, provider string) error {
	return s.store.Delete(ctx, user, provider)
}
