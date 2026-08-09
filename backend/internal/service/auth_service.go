package service

import (
	"context"
	"errors"
	"fmt"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/repository"
)

type AuthService struct {
	users *repository.UserRepository
}

func NewAuthService(users *repository.UserRepository) *AuthService {
	return &AuthService{users: users}
}

func (s *AuthService) Login(ctx context.Context, provider string, profile *auth.Profile) (*repository.User, error) {
	if profile.ProviderUserID == "" {
		return nil, fmt.Errorf("provider %s returned no subject id", provider)
	}
	if profile.Email == "" {
		return nil, fmt.Errorf("provider %s returned no email", provider)
	}

	user, err := s.users.FindByIdentity(ctx, provider, profile.ProviderUserID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	if profile.EmailVerified {
		existing, err := s.users.FindByEmail(ctx, profile.Email)
		if err == nil {
			if err := s.users.LinkIdentity(ctx, existing.ID, provider, profile.ProviderUserID, profile.Email); err != nil {
				return nil, err
			}
			return existing, nil
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
	}

	newUser := repository.User{
		Email:         profile.Email,
		EmailVerified: profile.EmailVerified,
		Name:          profile.Name,
		AvatarURL:     profile.AvatarURL,
	}
	return s.users.CreateWithIdentity(ctx, newUser, provider, profile.ProviderUserID, profile.Email)
}
