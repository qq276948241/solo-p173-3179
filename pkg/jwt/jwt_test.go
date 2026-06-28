package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const testSecret = "test-secret-key-for-jwt-unit-test"

func TestGenerateAndParseToken(t *testing.T) {
	svc := NewService(testSecret)

	token, err := svc.GenerateToken(42, "landlord")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := svc.ParseToken(token)
	assert.NoError(t, err)
	assert.Equal(t, uint(42), claims.UserID)
	assert.Equal(t, "landlord", claims.Role)
	assert.NotNil(t, claims.ExpiresAt)
	assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
}

func TestGenerateToken_TenantRole(t *testing.T) {
	svc := NewService(testSecret)
	token, err := svc.GenerateToken(99, "tenant")
	assert.NoError(t, err)

	claims, err := svc.ParseToken(token)
	assert.NoError(t, err)
	assert.Equal(t, uint(99), claims.UserID)
	assert.Equal(t, "tenant", claims.Role)
}

func TestParseToken_InvalidToken(t *testing.T) {
	svc := NewService(testSecret)

	_, err := svc.ParseToken("not.a.valid.token")
	assert.Error(t, err)
}

func TestParseToken_WrongSecret(t *testing.T) {
	svc1 := NewService("secret-one")
	svc2 := NewService("secret-two")

	token, err := svc1.GenerateToken(1, "landlord")
	assert.NoError(t, err)

	_, err = svc2.ParseToken(token)
	assert.Error(t, err)
}

func TestParseToken_ExpiredToken(t *testing.T) {
	svc := NewService(testSecret)

	claims := Claims{
		UserID: 1,
		Role:   "landlord",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(testSecret))
	assert.NoError(t, err)

	_, err = svc.ParseToken(tokenStr)
	assert.Error(t, err)
}

func TestParseToken_WrongSigningMethod(t *testing.T) {
	svc := NewService(testSecret)

	claims := Claims{
		UserID: 1,
		Role:   "landlord",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	assert.NoError(t, err)

	_, err = svc.ParseToken(tokenStr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected signing method")
}

func TestSetExpireHour_PositiveOnly(t *testing.T) {
	svc := NewService(testSecret)
	svc.SetExpireHour(0)
	svc.SetExpireHour(-5)

	token, err := svc.GenerateToken(1, "tenant")
	assert.NoError(t, err)

	claims, err := svc.ParseToken(token)
	assert.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(24*time.Hour), claims.ExpiresAt.Time, time.Minute)
}
