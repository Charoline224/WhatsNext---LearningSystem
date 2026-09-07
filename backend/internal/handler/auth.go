package handler

import (
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"whatsnext/backend/internal/dto"
	appmiddleware "whatsnext/backend/internal/middleware"
	"whatsnext/backend/internal/response"
	"whatsnext/backend/internal/service"
)

const refreshCookieName = "whatsnext_refresh"

type AuthHandler struct {
	service      *service.AuthService
	secureCookie bool
	refreshTTL   time.Duration
}

func NewAuthHandler(service *service.AuthService, secureCookie bool, refreshTTL time.Duration) *AuthHandler {
	return &AuthHandler{service: service, secureCookie: secureCookie, refreshTTL: refreshTTL}
}
func (h *AuthHandler) Register(c *gin.Context) {
	var input dto.RegisterInput
	if !bindJSON(c, &input) {
		return
	}
	result, err := h.service.Register(c.Request.Context(), input, c.Request.UserAgent(), clientIP(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	h.setRefreshCookie(c, result.RefreshToken)
	response.Created(c, result)
}
func (h *AuthHandler) Login(c *gin.Context) {
	var input dto.LoginInput
	if !bindJSON(c, &input) {
		return
	}
	result, err := h.service.Login(c.Request.Context(), input, c.Request.UserAgent(), clientIP(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	h.setRefreshCookie(c, result.RefreshToken)
	response.OK(c, result)
}
func (h *AuthHandler) Refresh(c *gin.Context) {
	plain, _ := c.Cookie(refreshCookieName)
	result, err := h.service.Refresh(c.Request.Context(), plain, c.Request.UserAgent(), clientIP(c))
	if err != nil {
		h.clearRefreshCookie(c)
		writeServiceError(c, err)
		return
	}
	h.setRefreshCookie(c, result.RefreshToken)
	response.OK(c, result)
}
func (h *AuthHandler) Logout(c *gin.Context) {
	plain, _ := c.Cookie(refreshCookieName)
	if err := h.service.Logout(c.Request.Context(), plain); err != nil {
		writeServiceError(c, err)
		return
	}
	h.clearRefreshCookie(c)
	response.NoContent(c)
}
func (h *AuthHandler) Me(c *gin.Context) {
	user, err := h.service.GetUser(c.Request.Context(), appmiddleware.UserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, user)
}
func (h *AuthHandler) setRefreshCookie(c *gin.Context, value string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(refreshCookieName, value, int(h.refreshTTL.Seconds()), "/api/v1/auth", "", h.secureCookie, true)
}
func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(refreshCookieName, "", -1, "/api/v1/auth", "", h.secureCookie, true)
}
func clientIP(c *gin.Context) []byte {
	ip := net.ParseIP(c.ClientIP())
	if ip == nil {
		return nil
	}
	if v4 := ip.To4(); v4 != nil {
		return []byte(v4)
	}
	return []byte(ip.To16())
}
