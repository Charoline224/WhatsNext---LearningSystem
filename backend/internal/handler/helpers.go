package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"whatsnext/backend/internal/response"
	"whatsnext/backend/internal/service"
)

func writeServiceError(c *gin.Context, err error) {
	var validation service.ValidationError
	switch {
	case errors.As(err, &validation):
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数有误", []any{map[string]string{"field": validation.Field, "reason": validation.Reason}})
	case errors.Is(err, service.ErrEmailTaken):
		response.Error(c, http.StatusConflict, "EMAIL_ALREADY_EXISTS", "该邮箱已注册", nil)
	case errors.Is(err, service.ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "邮箱或密码错误", nil)
	case errors.Is(err, service.ErrUnauthenticated):
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "请重新登录", nil)
	case errors.Is(err, service.ErrForbidden):
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权执行此操作", nil)
	case errors.Is(err, service.ErrNotFound):
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "资源不存在", nil)
	case errors.Is(err, service.ErrConflict):
		response.Error(c, http.StatusConflict, "CONFLICT", "当前状态不允许执行此操作", nil)
	case errors.Is(err, service.ErrFileTooLarge):
		response.Error(c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "文件不能超过 100 MB", nil)
	case errors.Is(err, service.ErrUnsupportedFile):
		response.Error(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_FILE", "文件格式或内容无效，仅支持 PDF、PPTX、DOCX、TXT 和 Markdown", nil)
	case errors.Is(err, service.ErrAIProvider):
		response.Error(c, http.StatusBadGateway, "AI_PROVIDER_ERROR", "Embedding 服务调用失败", nil)
	case errors.Is(err, service.ErrAIUnavailable), errors.Is(err, service.ErrRetrievalUnavailable):
		response.Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "RAG 检索服务暂不可用", nil)
	default:
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "服务器内部错误", nil)
	}
}
func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_JSON", "请求体不是有效的 JSON", nil)
		return false
	}
	return true
}
