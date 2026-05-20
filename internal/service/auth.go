package service

import (
	"context"
	"errors"
	"fmt"

	"warehouse-controller/internal/auth"
	"warehouse-controller/internal/repo"

	"github.com/google/uuid"
)

var ErrRefreshRejected = errors.New("refresh rejected")

type TokenPair struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

type AuthService struct {
	issuer   *auth.Issuer
	sessions *repo.SessionRepo
}

func NewAuthService(issuer *auth.Issuer, sessions *repo.SessionRepo) *AuthService {
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
		if errors.Is(err, repo.ErrSessionNotFound) {
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
	access, err := s.issuer.IssueAccess(sessionID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("issue access: %w", err)
	}
	refresh, jti, err := s.issuer.IssueRefresh(sessionID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("issue refresh: %w", err)
	}
	if err := s.sessions.SetRefreshJTI(ctx, sessionID, jti, s.issuer.RefreshTTL()); err != nil {
		return TokenPair{}, fmt.Errorf("store refresh jti: %w", err)
	}
	return TokenPair{Access: access, Refresh: refresh}, nil
}
