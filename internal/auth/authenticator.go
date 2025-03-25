package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/computer-technology-team/go-judge/config"
)

const issuer = "gojudge"

var ErrSigningKeyNotFound = errors.New("signing key not found")

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type Authenticator interface {
	GenerateToken(context.Context, Claims) (string, error)
	VerifyDecodeToken(context.Context, string) (*Claims, error)
}

type AuthenticatorImpl struct {
	keys                map[string][]byte
	keyID               string
	tokenExpireDuration time.Duration
}

// GenerateToken implements Authenticator.
func (a *AuthenticatorImpl) GenerateToken(ctx context.Context, claims Claims) (string, error) {
	now := time.Now()

	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        uuid.NewString(),
		Issuer:    issuer,
		Subject:   claims.UserID,
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(a.tokenExpireDuration)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = a.keyID

	key, ok := a.keys[a.keyID]
	if !ok {
		return "", errors.New("active signing key not found")
	}

	return token.SignedString(key)
}

func (a *AuthenticatorImpl) VerifyDecodeToken(ctx context.Context, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, a.GetKey, jwt.WithValidMethods([]string{
		jwt.SigningMethodHS256.Name,
		jwt.SigningMethodES384.Name,
		jwt.SigningMethodES512.Name,
	}))
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func (a *AuthenticatorImpl) GetKey(token *jwt.Token) (any, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}

	kidRaw, ok := token.Header["kid"]
	if !ok {
		return nil, errors.New("kid not found")
	}

	kid, ok := kidRaw.(string)
	if !ok {
		return nil, errors.New("invalid key ID format")
	}

	key, ok := a.keys[kid]
	if !ok {
		return nil, ErrSigningKeyNotFound
	}

	return key, nil
}

func NewAuthenticator(cfg config.AuthenticationConfig) (Authenticator, error) {
	var errs error
	keys := lo.MapValues(cfg.Keys, func(v string, k string) []byte {
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			errs = errors.Join(errs, err)
			return nil
		}

		return decoded
	})
	if errs != nil {
		return nil, fmt.Errorf("failed to decode keys for authenticator: %w", errs)
	}

	if _, ok := keys[cfg.ActiveKeyID]; !ok {
		return nil, ErrSigningKeyNotFound
	}

	return &AuthenticatorImpl{
		keys:                keys,
		keyID:               cfg.ActiveKeyID,
		tokenExpireDuration: cfg.TokenExpiry,
	}, nil
}
