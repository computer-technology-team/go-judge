package auth

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/computer-technology-team/go-judge/config"
)

type AuthenticatorTestSuite struct {
	suite.Suite
	testKey     string
	testKeyID   string
	testKeyID2  string
	testKey2    string
	validClaims Claims
	ctx         context.Context
}

func (s *AuthenticatorTestSuite) SetupTest() {
	s.testKey = "test-secret-key-for-jwt-signing"
	s.testKeyID = "test-key-1"
	s.testKey2 = "test-secret-key-2-for-rotation"
	s.testKeyID2 = "test-key-2"
	s.validClaims = Claims{UserID: "user123"}
	s.ctx = context.Background()
}

func (s *AuthenticatorTestSuite) createAuthenticator(keyID string, expiry time.Duration) Authenticator {
	cfg := config.AuthenticationConfig{
		Keys: map[string]string{
			s.testKeyID:  base64.StdEncoding.EncodeToString([]byte(s.testKey)),
			s.testKeyID2: base64.StdEncoding.EncodeToString([]byte(s.testKey2)),
		},
		ActiveKeyID: keyID,
		TokenExpiry: expiry,
	}

	auth, err := NewAuthenticator(cfg)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), auth)

	return auth
}

func (s *AuthenticatorTestSuite) TestGenerateToken() {
	// Create authenticator with standard settings
	auth := s.createAuthenticator(s.testKeyID, 1*time.Hour)

	// Generate token
	tokenString, _, err := auth.GenerateToken(s.ctx, s.validClaims)

	// Assertions
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), tokenString)

	// Parse and verify the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.testKey), nil
	})

	require.NoError(s.T(), err)
	assert.True(s.T(), token.Valid)

	// Verify header contains kid
	kid, ok := token.Header["kid"]
	assert.True(s.T(), ok, "Token header should contain 'kid'")
	assert.Equal(s.T(), s.testKeyID, kid)

	// Verify signing method
	assert.Equal(s.T(), jwt.SigningMethodHS256.Alg(), token.Method.Alg())

	// Verify claims
	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(s.T(), ok)

	// Check user_id claim
	userID, ok := claims["user_id"]
	assert.True(s.T(), ok, "Token should contain 'user_id' claim")
	assert.Equal(s.T(), s.validClaims.UserID, userID)

	// Check standard claims
	s.verifyStandardClaims(claims)
}

func (s *AuthenticatorTestSuite) verifyStandardClaims(claims jwt.MapClaims) {
	// Check issuer
	issuerClaim, ok := claims["iss"]
	assert.True(s.T(), ok, "Token should contain 'iss' claim")
	assert.Equal(s.T(), issuer, issuerClaim)

	// Check subject
	subClaim, ok := claims["sub"]
	assert.True(s.T(), ok, "Token should contain 'sub' claim")
	assert.Equal(s.T(), s.validClaims.UserID, subClaim)

	// Check expiration
	_, ok = claims["exp"]
	assert.True(s.T(), ok, "Token should contain 'exp' claim")

	// Check other required claims
	_, ok = claims["iat"]
	assert.True(s.T(), ok, "Token should contain 'iat' claim")

	_, ok = claims["nbf"]
	assert.True(s.T(), ok, "Token should contain 'nbf' claim")

	_, ok = claims["jti"]
	assert.True(s.T(), ok, "Token should contain 'jti' claim")
}

func (s *AuthenticatorTestSuite) TestTokenExpiration() {
	// Create authenticator with custom expiration
	expiry := 2 * time.Hour
	auth := s.createAuthenticator(s.testKeyID, expiry)

	// Generate token
	tokenString, _, err := auth.GenerateToken(s.ctx, s.validClaims)
	require.NoError(s.T(), err)

	// Parse token to check expiration
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.testKey), nil
	})
	require.NoError(s.T(), err)

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(s.T(), ok)

	// Check expiration time
	expClaim, ok := claims["exp"]
	assert.True(s.T(), ok)

	// Convert exp to time.Time for comparison
	expFloat, ok := expClaim.(float64)
	assert.True(s.T(), ok, "exp claim should be a number")

	expTime := time.Unix(int64(expFloat), 0)
	expectedExpTime := time.Now().Add(expiry)

	// Allow for a small time difference due to test execution
	timeDiff := expectedExpTime.Sub(expTime)
	assert.Less(s.T(), timeDiff.Abs(), 5*time.Second, "Expiration time should be close to expected")
}

func (s *AuthenticatorTestSuite) TestVerifyDecodeToken() {
	// Create authenticator
	auth := s.createAuthenticator(s.testKeyID, 1*time.Hour)

	// Generate a valid token
	validToken, _, err := auth.GenerateToken(s.ctx, s.validClaims)
	require.NoError(s.T(), err)

	// Create an expired token
	expiredAuth := s.createAuthenticator(s.testKeyID, -1*time.Hour) // Negative duration to create expired token
	expiredToken, _, err := expiredAuth.GenerateToken(s.ctx, s.validClaims)
	require.NoError(s.T(), err)

	// Create a token with wrong signing method
	wrongMethodToken := s.createTokenWithWrongMethod()

	// Test cases
	testCases := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   validToken,
			wantErr: false,
		},
		{
			name:    "expired token",
			token:   expiredToken,
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "not.a.token",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "wrong signing method",
			token:   wrongMethodToken,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			claims, err := auth.VerifyDecodeToken(s.ctx, tc.token)

			if tc.wantErr {
				assert.Error(s.T(), err)
				assert.Nil(s.T(), claims)
				return
			}

			assert.NoError(s.T(), err)
			assert.NotNil(s.T(), claims)
			assert.Equal(s.T(), s.validClaims.UserID, claims.UserID)
			assert.Equal(s.T(), s.validClaims.UserID, claims.Subject)
			assert.Equal(s.T(), issuer, claims.Issuer)
		})
	}
}

func (s *AuthenticatorTestSuite) TestKeyRotation() {
	// Create authenticator with first key active
	auth1 := s.createAuthenticator(s.testKeyID, 1*time.Hour)

	// Generate token with first key
	token1, _, err := auth1.GenerateToken(s.ctx, s.validClaims)
	require.NoError(s.T(), err)

	// Parse token to verify kid
	parsedToken1, err := jwt.Parse(token1, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.testKey), nil
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), s.testKeyID, parsedToken1.Header["kid"])

	// Create authenticator with second key active
	auth2 := s.createAuthenticator(s.testKeyID2, 1*time.Hour)

	// Generate token with second key
	token2, _, err := auth2.GenerateToken(s.ctx, s.validClaims)
	require.NoError(s.T(), err)

	// Parse token to verify kid
	parsedToken2, err := jwt.Parse(token2, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.testKey2), nil
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), s.testKeyID2, parsedToken2.Header["kid"])

	// Verify both tokens can be decoded with the same authenticator (key rotation)
	decodedClaims1, err := auth2.VerifyDecodeToken(s.ctx, token1)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), s.validClaims.UserID, decodedClaims1.UserID)

	decodedClaims2, err := auth2.VerifyDecodeToken(s.ctx, token2)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), s.validClaims.UserID, decodedClaims2.UserID)
}

func (s *AuthenticatorTestSuite) TestNewAuthenticatorErrors() {
	testCases := []struct {
		name    string
		cfg     config.AuthenticationConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: config.AuthenticationConfig{
				Keys: map[string]string{
					"key1": base64.StdEncoding.EncodeToString([]byte("secret")),
				},
				ActiveKeyID: "key1",
				TokenExpiry: time.Hour,
			},
			wantErr: false,
		},
		{
			name: "missing active key",
			cfg: config.AuthenticationConfig{
				Keys: map[string]string{
					"key1": base64.StdEncoding.EncodeToString([]byte("secret")),
				},
				ActiveKeyID: "non-existent-key",
				TokenExpiry: time.Hour,
			},
			wantErr: true,
		},
		{
			name: "invalid base64 in key",
			cfg: config.AuthenticationConfig{
				Keys: map[string]string{
					"key1": "not-valid-base64!@#",
				},
				ActiveKeyID: "key1",
				TokenExpiry: time.Hour,
			},
			wantErr: true,
		},
		{
			name: "empty keys map",
			cfg: config.AuthenticationConfig{
				Keys:        map[string]string{},
				ActiveKeyID: "key1",
				TokenExpiry: time.Hour,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			auth, err := NewAuthenticator(tc.cfg)

			if tc.wantErr {
				assert.Error(s.T(), err)
				assert.Nil(s.T(), auth)
			} else {
				assert.NoError(s.T(), err)
				assert.NotNil(s.T(), auth)
			}
		})
	}
}

// Helper function to create a token with a wrong signing method
func (s *AuthenticatorTestSuite) createTokenWithWrongMethod() string {
	claims := jwt.MapClaims{
		"user_id": s.validClaims.UserID,
		"sub":     s.validClaims.UserID,
		"iss":     issuer,
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
		"nbf":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	token.Header["kid"] = s.testKeyID
	token.Header["alg"] = "none" // This will be rejected by our authenticator

	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(s.T(), err)

	return tokenString
}

// Run the test suite
func TestAuthenticatorSuite(t *testing.T) {
	suite.Run(t, new(AuthenticatorTestSuite))
}
