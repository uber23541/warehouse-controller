package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"warehouse-controller/internal/auth"
	cachemock "warehouse-controller/internal/mocks/cache"
	"warehouse-controller/internal/platform/cache"
	sessioncache "warehouse-controller/internal/platform/cache/session"
	"warehouse-controller/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newAuthService(t *testing.T) (*service.AuthService, *cachemock.MockCache, *auth.Issuer) {
	t.Helper()
	c := cachemock.NewMockCache(t)
	issuer := auth.NewIssuer("secret", 15*time.Minute, time.Hour)
	svc := service.NewAuthService(issuer, sessioncache.New(c))
	return svc, c, issuer
}

func keyPrefix(p string) func(string) bool {
	return func(key string) bool { return len(key) >= len(p) && key[:len(p)] == p }
}

func TestAuthService_IssuePair_StoresBothJTI(t *testing.T) {
	svc, c, _ := newAuthService(t)
	ctx := context.Background()

	c.EXPECT().Set(ctx, mock.MatchedBy(keyPrefix("access:")), mock.Anything, mock.Anything).Return(nil).Once()
	c.EXPECT().Set(ctx, mock.MatchedBy(keyPrefix("refresh:")), mock.Anything, mock.Anything).Return(nil).Once()

	pair, err := svc.IssuePair(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.Access)
	assert.NotEmpty(t, pair.Refresh)
}

// stored == nil → Get не ожидается (токен невалиден, сессия не запрашивается).
// wantErr == nil → успех: проверяется перевыпуск пары и сохранение новых JTI.
func TestAuthService_Refresh(t *testing.T) {
	storeErr := errors.New("redis down")

	tests := []struct {
		name    string
		token   func(iss *auth.Issuer) (token, sub, jti string)
		stored  func(jti string) ([]byte, error)
		wantErr error
	}{
		{
			name: "valid",
			token: func(i *auth.Issuer) (string, string, string) {
				tok, jti, _ := i.IssueRefresh("s1")
				return tok, "s1", jti
			},
			stored:  func(jti string) ([]byte, error) { return []byte(jti), nil },
			wantErr: nil,
		},
		{
			name:    "invalid token",
			token:   func(*auth.Issuer) (string, string, string) { return "not-a-jwt", "", "" },
			wantErr: service.ErrRefreshRejected,
		},
		{
			name: "jti not found",
			token: func(i *auth.Issuer) (string, string, string) {
				tok, jti, _ := i.IssueRefresh("s2")
				return tok, "s2", jti
			},
			stored:  func(string) ([]byte, error) { return nil, cache.ErrNotFound },
			wantErr: service.ErrRefreshRejected,
		},
		{
			name: "jti mismatch",
			token: func(i *auth.Issuer) (string, string, string) {
				tok, jti, _ := i.IssueRefresh("s3")
				return tok, "s3", jti
			},
			stored:  func(string) ([]byte, error) { return []byte("other-jti"), nil },
			wantErr: service.ErrRefreshRejected,
		},
		{
			name: "store error",
			token: func(i *auth.Issuer) (string, string, string) {
				tok, jti, _ := i.IssueRefresh("s4")
				return tok, "s4", jti
			},
			stored:  func(string) ([]byte, error) { return nil, storeErr },
			wantErr: storeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, c, issuer := newAuthService(t)
			ctx := context.Background()

			token, sub, jti := tt.token(issuer)
			if tt.stored != nil {
				c.EXPECT().Get(ctx, "refresh:"+sub).Return(tt.stored(jti)).Once()
			}
			if tt.wantErr == nil {
				c.EXPECT().Set(ctx, "access:"+sub, mock.Anything, mock.Anything).Return(nil).Once()
				c.EXPECT().Set(ctx, "refresh:"+sub, mock.Anything, mock.Anything).Return(nil).Once()
			}

			pair, err := svc.Refresh(ctx, token)

			if tt.wantErr == nil {
				require.NoError(t, err)
				assert.NotEmpty(t, pair.Access)
				assert.NotEmpty(t, pair.Refresh)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
			if !errors.Is(tt.wantErr, service.ErrRefreshRejected) {
				assert.NotErrorIs(t, err, service.ErrRefreshRejected)
			}
		})
	}
}

func TestAuthService_ValidateAccess(t *testing.T) {
	storeErr := errors.New("redis down")

	tests := []struct {
		name    string
		token   func(iss *auth.Issuer) (token, sub, jti string)
		stored  func(jti string) ([]byte, error)
		wantSub string
		wantErr error
	}{
		{
			name: "valid",
			token: func(i *auth.Issuer) (string, string, string) {
				tok, jti, _ := i.IssueAccess("s5")
				return tok, "s5", jti
			},
			stored:  func(jti string) ([]byte, error) { return []byte(jti), nil },
			wantSub: "s5",
		},
		{
			name:    "invalid token",
			token:   func(*auth.Issuer) (string, string, string) { return "garbage", "", "" },
			wantErr: service.ErrAccessRejected,
		},
		{
			name: "jti mismatch",
			token: func(i *auth.Issuer) (string, string, string) {
				tok, jti, _ := i.IssueAccess("s6")
				return tok, "s6", jti
			},
			stored:  func(string) ([]byte, error) { return []byte("stale-jti"), nil },
			wantErr: service.ErrAccessRejected,
		},
		{
			name: "not found",
			token: func(i *auth.Issuer) (string, string, string) {
				tok, jti, _ := i.IssueAccess("s7")
				return tok, "s7", jti
			},
			stored:  func(string) ([]byte, error) { return nil, cache.ErrNotFound },
			wantErr: service.ErrAccessRejected,
		},
		{
			name: "store error",
			token: func(i *auth.Issuer) (string, string, string) {
				tok, jti, _ := i.IssueAccess("s8")
				return tok, "s8", jti
			},
			stored:  func(string) ([]byte, error) { return nil, storeErr },
			wantErr: storeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, c, issuer := newAuthService(t)
			ctx := context.Background()

			token, sub, jti := tt.token(issuer)
			if tt.stored != nil {
				c.EXPECT().Get(ctx, "access:"+sub).Return(tt.stored(jti)).Once()
			}

			gotSub, err := svc.ValidateAccess(ctx, token)

			if tt.wantErr == nil {
				require.NoError(t, err)
				assert.Equal(t, tt.wantSub, gotSub)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
			if !errors.Is(tt.wantErr, service.ErrAccessRejected) {
				assert.NotErrorIs(t, err, service.ErrAccessRejected)
			}
		})
	}
}
