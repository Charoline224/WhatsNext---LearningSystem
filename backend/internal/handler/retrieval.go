package handler

import (
	"github.com/gin-gonic/gin"
	"whatsnext/backend/internal/dto"
	appmiddleware "whatsnext/backend/internal/middleware"
	"whatsnext/backend/internal/response"
	"whatsnext/backend/internal/service"
)

type RetrievalHandler struct{ service *service.RetrievalService }

func NewRetrievalHandler(service *service.RetrievalService) *RetrievalHandler {
	return &RetrievalHandler{service: service}
}

func (h *RetrievalHandler) Search(c *gin.Context) {
	var input dto.RetrievalSearchInput
	if !bindJSON(c, &input) {
		return
	}
	result, err := h.service.Search(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}
