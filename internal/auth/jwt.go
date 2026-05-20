package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	typAccess  = "access"
	typRefresh = "refresh"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrWrongType    = errors.New("wrong token type")
)

type Claims struct {
	Typ string `json:"typ"`
	jwt.RegisteredClaims
}

type Issuer struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewIssuer(secret string, accessTTL, refreshTTL time.Duration) *Issuer {
	return &Issuer{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (i *Issuer) AccessTTL() time.Duration  { return i.accessTTL }
func (i *Issuer) RefreshTTL() time.Duration { return i.refreshTTL }

func (i *Issuer) IssueAccess(sessionID string) (string, error) {
	now := time.Now()
	claims := Claims{
		Typ: typAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sessionID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(i.accessTTL)),
		},
	}
	return i.sign(claims)
}

func (i *Issuer) IssueRefresh(sessionID string) (token, jti string, err error) {
	now := time.Now()
	jti = uuid.NewString()
	claims := Claims{
		Typ: typRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sessionID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(i.refreshTTL)),
		},
	}
	token, err = i.sign(claims)
	return token, jti, err
}

func (i *Issuer) ParseAccess(token string) (*Claims, error) {
	return i.parse(token, typAccess)
}

func (i *Issuer) ParseRefresh(token string) (*Claims, error) {
	return i.parse(token, typRefresh)
}

func (i *Issuer) sign(claims Claims) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	return signed, nil
}

func (i *Issuer) parse(token, expectedTyp string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return i.secret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	if claims.Typ != expectedTyp {
		return nil, ErrWrongType
	}
	return claims, nil
}
