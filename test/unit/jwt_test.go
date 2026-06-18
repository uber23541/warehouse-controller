package unit

import (
	"testing"
	"time"

	"warehouse-controller/internal/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const jwtSecret = "test-secret"

func newIssuer() *auth.Issuer {
	return auth.NewIssuer(jwtSecret, 15*time.Minute, 168*time.Hour)
}

func TestIssuer_RoundTrip(t *testing.T) {
	iss := newIssuer()

	tests := []struct {
		name  string
		issue func(sessionID string) (string, string, error)
		parse func(token string) (*auth.Claims, error)
	}{
		{"access", iss.IssueAccess, iss.ParseAccess},
		{"refresh", iss.IssueRefresh, iss.ParseRefresh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, jti, err := tt.issue("session-1")
			require.NoError(t, err)
			require.NotEmpty(t, token)
			require.NotEmpty(t, jti)

			claims, err := tt.parse(token)
			require.NoError(t, err)
			assert.Equal(t, "session-1", claims.Subject)
			assert.Equal(t, jti, claims.ID)
		})
	}
}

func TestIssuer_ParseErrors(t *testing.T) {
	iss := newIssuer()
	foreign := auth.NewIssuer("another-secret", 15*time.Minute, time.Hour)
	expired := auth.NewIssuer(jwtSecret, -time.Minute, -time.Minute)

	noneToken := func() string {
		claims := auth.Claims{Typ: "access", RegisteredClaims: jwt.RegisteredClaims{Subject: "s"}}
		signed, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)
		return signed
	}
	issue := func(i *auth.Issuer, access bool) string {
		var token string
		var err error
		if access {
			token, _, err = i.IssueAccess("s")
		} else {
			token, _, err = i.IssueRefresh("s")
		}
		require.NoError(t, err)
		return token
	}

	tests := []struct {
		name    string
		token   func() string
		parse   func(string) (*auth.Claims, error)
		wantErr error
	}{
		{"access parsed as refresh", func() string { return issue(iss, true) }, iss.ParseRefresh, auth.ErrWrongType},
		{"refresh parsed as access", func() string { return issue(iss, false) }, iss.ParseAccess, auth.ErrWrongType},
		{"foreign secret", func() string { return issue(foreign, true) }, iss.ParseAccess, auth.ErrInvalidToken},
		{"none alg", noneToken, iss.ParseAccess, auth.ErrInvalidToken},
		{"expired", func() string { return issue(expired, true) }, iss.ParseAccess, auth.ErrInvalidToken},
		{"garbage", func() string { return "not-a-jwt" }, iss.ParseAccess, auth.ErrInvalidToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.parse(tt.token())
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
