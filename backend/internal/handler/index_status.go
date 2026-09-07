package handler

import (
	"github.com/gin-gonic/gin"
	appmiddleware "whatsnext/backend/internal/middleware"
	"whatsnext/backend/internal/response"
	"whatsnext/backend/internal/service"
)

type IndexStatusHandler struct{ service *service.IndexStatusService }

func NewIndexStatusHandler(service *service.IndexStatusService) *IndexStatusHandler {
	return &IndexStatusHandler{service: service}
}

func (h *IndexStatusHandler) Get(c *gin.Context) {
	result, err := h.service.Get(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("materialId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}
