package response

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Envelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type ErrorEnvelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id"`
}

func OK(c *gin.Context, data any) {
	c.JSON(200, Envelope{Code: "OK", Message: "", Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Code: "OK", Message: "", Data: data})
}
func Accepted(c *gin.Context, data any) {
	c.JSON(http.StatusAccepted, Envelope{Code: "OK", Message: "", Data: data})
}
func NoContent(c *gin.Context) { c.Status(http.StatusNoContent) }

func Error(c *gin.Context, status int, code, message string, details any) {
	c.AbortWithStatusJSON(status, ErrorEnvelope{
		Code: code, Message: message, Details: details, RequestID: c.GetString("request_id"),
	})
}
