package handler

import (
	"github.com/gin-gonic/gin"
	"whatsnext/backend/internal/dto"
	appmiddleware "whatsnext/backend/internal/middleware"
	"whatsnext/backend/internal/response"
	"whatsnext/backend/internal/service"
)

type LearningSpaceHandler struct{ service *service.LearningSpaceService }

func NewLearningSpaceHandler(service *service.LearningSpaceService) *LearningSpaceHandler {
	return &LearningSpaceHandler{service: service}
}
func (h *LearningSpaceHandler) Create(c *gin.Context) {
	var input dto.CreateSpaceInput
	if !bindJSON(c, &input) {
		return
	}
	space, err := h.service.Create(c.Request.Context(), appmiddleware.UserID(c), c.GetHeader("Idempotency-Key"), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Created(c, space)
}
func (h *LearningSpaceHandler) List(c *gin.Context) {
	page, err := h.service.List(c.Request.Context(), appmiddleware.UserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, page)
}
func (h *LearningSpaceHandler) Get(c *gin.Context) {
	detail, err := h.service.Get(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, detail)
}
func (h *LearningSpaceHandler) Update(c *gin.Context) {
	var input dto.UpdateSpaceInput
	if !bindJSON(c, &input) {
		return
	}
	space, err := h.service.Update(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, space)
}
func (h *LearningSpaceHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId")); err != nil {
		writeServiceError(c, err)
		return
	}
	response.NoContent(c)
}
