package middleware

import (
	"net/http"
	"net/http/httptest"
	"project173/models"
	"project173/pkg/jwt"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter(jwtService *jwt.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", AuthMiddleware(jwtService), func(c *gin.Context) {
		userID, ok := GetUserID(c)
		role, _ := GetUserRole(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"role":    role,
			"ok":      ok,
		})
	})
	r.GET("/landlord-only", AuthMiddleware(jwtService), RoleMiddleware(string(models.RoleLandlord)), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"allowed": true})
	})
	r.GET("/tenant-only", AuthMiddleware(jwtService), RoleMiddleware(string(models.RoleTenant)), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"allowed": true})
	})
	return r
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwtService := jwt.NewService("test-secret")
	token, err := jwtService.GenerateToken(123, string(models.RoleLandlord))
	assert.NoError(t, err)

	r := setupTestRouter(jwtService)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "123")
	assert.Contains(t, w.Body.String(), "landlord")
}

func TestAuthMiddleware_NoHeader(t *testing.T) {
	jwtService := jwt.NewService("test-secret")
	r := setupTestRouter(jwtService)
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "authorization header is required")
}

func TestAuthMiddleware_BadFormat(t *testing.T) {
	jwtService := jwt.NewService("test-secret")
	r := setupTestRouter(jwtService)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Token without-bearer-prefix")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid authorization header format")
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	jwtService := jwt.NewService("test-secret")
	r := setupTestRouter(jwtService)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer fake.token.here")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired token")
}

func TestRoleMiddleware_LandlordAllowed(t *testing.T) {
	jwtService := jwt.NewService("test-secret")
	token, _ := jwtService.GenerateToken(1, string(models.RoleLandlord))

	r := setupTestRouter(jwtService)
	req := httptest.NewRequest("GET", "/landlord-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoleMiddleware_LandlordDeniedForTenant(t *testing.T) {
	jwtService := jwt.NewService("test-secret")
	token, _ := jwtService.GenerateToken(1, string(models.RoleTenant))

	r := setupTestRouter(jwtService)
	req := httptest.NewRequest("GET", "/landlord-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "insufficient permissions")
}

func TestRoleMiddleware_TenantAllowed(t *testing.T) {
	jwtService := jwt.NewService("test-secret")
	token, _ := jwtService.GenerateToken(2, string(models.RoleTenant))

	r := setupTestRouter(jwtService)
	req := httptest.NewRequest("GET", "/tenant-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoleMiddleware_TenantDeniedForLandlord(t *testing.T) {
	jwtService := jwt.NewService("test-secret")
	token, _ := jwtService.GenerateToken(2, string(models.RoleLandlord))

	r := setupTestRouter(jwtService)
	req := httptest.NewRequest("GET", "/tenant-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetUserIDAndRole_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	id, ok := GetUserID(c)
	assert.False(t, ok)
	assert.Equal(t, uint(0), id)

	role, ok := GetUserRole(c)
	assert.False(t, ok)
	assert.Equal(t, "", role)
}
