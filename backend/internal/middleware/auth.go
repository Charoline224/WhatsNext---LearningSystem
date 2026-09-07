package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	appauth "whatsnext/backend/internal/auth"
	"whatsnext/backend/internal/response"
)

const UserIDKey = "user_id"

func RequireAuth(tokens *appauth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		scheme, token, ok := strings.Cut(header, " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "请先登录", nil)
			return
		}
		claims, err := tokens.ParseAccessToken(strings.TrimSpace(token))
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "登录状态已失效", nil)
			return
		}
		c.Set(UserIDKey, claims.Subject)
		c.Next()
	}
}
func UserID(c *gin.Context) string { return c.GetString(UserIDKey) }
