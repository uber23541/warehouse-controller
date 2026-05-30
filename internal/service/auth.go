package service

import (
	"context"
	"errors"
	"fmt"

	"warehouse-controller/internal/auth"
	"warehouse-controller/internal/cache"
	sessioncache "warehouse-controller/internal/cache/session"

	"github.com/google/uuid"
)

var (
	ErrRefreshRejected = errors.New("refresh rejected")
	ErrAccessRejected  = errors.New("access rejected")
)

type TokenPair struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

type AuthService struct {
	issuer   *auth.Issuer
	sessions *sessioncache.Store
}

func NewAuthService(issuer *auth.Issuer, sessions *sessioncache.Store) *AuthService {
	return &AuthService{issuer: issuer, sessions: sessions}
}

func (s *AuthService) IssuePair(ctx context.Context) (TokenPair, error) {
	return s.issueFor(ctx, uuid.NewString())
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	claims, err := s.issuer.ParseRefresh(refreshToken)
	if err != nil {
		return TokenPair{}, ErrRefreshRejected
	}

	storedJTI, err := s.sessions.GetRefreshJTI(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return TokenPair{}, ErrRefreshRejected
		}
		return TokenPair{}, fmt.Errorf("refresh: %w", err)
	}
	if storedJTI != claims.ID {
		return TokenPair{}, ErrRefreshRejected
	}

	return s.issueFor(ctx, claims.Subject)
}

func (s *AuthService) issueFor(ctx context.Context, sessionID string) (TokenPair, error) {
	access, accessJTI, err := s.issuer.IssueAccess(sessionID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("issue access: %w", err)
	}
	refresh, refreshJTI, err := s.issuer.IssueRefresh(sessionID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("issue refresh: %w", err)
	}
	// Перезаписываем access JTI — старый access-токен сессии становится недействительным.
	if err := s.sessions.SetAccessJTI(ctx, sessionID, accessJTI, s.issuer.AccessTTL()); err != nil {
		return TokenPair{}, fmt.Errorf("store access jti: %w", err)
	}
	if err := s.sessions.SetRefreshJTI(ctx, sessionID, refreshJTI, s.issuer.RefreshTTL()); err != nil {
		return TokenPair{}, fmt.Errorf("store refresh jti: %w", err)
	}
	return TokenPair{Access: access, Refresh: refresh}, nil
}

func (s *AuthService) ValidateAccess(ctx context.Context, accessToken string) (string, error) {
	claims, err := s.issuer.ParseAccess(accessToken)
	if err != nil {
		return "", ErrAccessRejected
	}

	storedJTI, err := s.sessions.GetAccessJTI(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return "", ErrAccessRejected
		}
		return "", fmt.Errorf("validate access: %w", err)
	}
	if storedJTI != claims.ID {
		return "", ErrAccessRejected
	}

	return claims.Subject, nil
}
