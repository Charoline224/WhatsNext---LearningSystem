package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	appmiddleware "whatsnext/backend/internal/middleware"
	"whatsnext/backend/internal/response"
	"whatsnext/backend/internal/service"
)

type MaterialHandler struct {
	service  *service.MaterialService
	maxBytes int64
}

func NewMaterialHandler(service *service.MaterialService, maxBytes int64) *MaterialHandler {
	return &MaterialHandler{service: service, maxBytes: maxBytes}
}
func (h *MaterialHandler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBytes+1024*1024)
	header, err := c.FormFile("file")
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			response.Error(c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "文件不能超过 100 MB", nil)
		} else {
			response.Error(c, http.StatusBadRequest, "FILE_REQUIRED", "请选择要上传的学习资料", nil)
		}
		return
	}
	file, err := header.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_FILE", "无法读取上传文件", nil)
		return
	}
	defer file.Close()
	result, err := h.service.Upload(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.PostForm("material_kind"), header.Filename, header.Size, file)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Accepted(c, result)
}
func (h *MaterialHandler) Get(c *gin.Context) {
	result, err := h.service.Get(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("materialId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}
func (h *MaterialHandler) Retry(c *gin.Context) {
	result, err := h.service.Retry(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("materialId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Accepted(c, result)
}
func (h *MaterialHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("materialId")); err != nil {
		writeServiceError(c, err)
		return
	}
	response.NoContent(c)
}
func (h *MaterialHandler) Download(c *gin.Context) {
	result, err := h.service.Download(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("materialId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}
func (h *MaterialHandler) List(c *gin.Context) {
	result, err := h.service.List(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}
func (h *MaterialHandler) GetJob(c *gin.Context) {
	result, err := h.service.GetJob(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("jobId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}
func (h *MaterialHandler) ListChunks(c *gin.Context) {
	result, err := h.service.ListChunks(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("materialId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}
