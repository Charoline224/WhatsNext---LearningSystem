package handler

import (
	"github.com/gin-gonic/gin"
	"whatsnext/backend/internal/dto"
	appmiddleware "whatsnext/backend/internal/middleware"
	"whatsnext/backend/internal/response"
	"whatsnext/backend/internal/service"
)

func (h *LearningAssetHandler) CreateNode(c *gin.Context) {
	var input dto.KnowledgeNodeInput
	if !bindJSON(c, &input) {
		return
	}
	out, err := h.service.CreateNode(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Created(c, out)
}
func (h *LearningAssetHandler) UpdateNode(c *gin.Context) {
	var input dto.KnowledgeNodeInput
	if !bindJSON(c, &input) {
		return
	}
	out, err := h.service.UpdateNode(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("nodeId"), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, out)
}
func (h *LearningAssetHandler) UpdateNodePosition(c *gin.Context) {
	var input dto.KnowledgeNodePositionInput
	if !bindJSON(c, &input) {
		return
	}
	if err := h.service.UpdateNodePosition(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("nodeId"), input); err != nil {
		writeServiceError(c, err)
		return
	}
	response.NoContent(c)
}
func (h *LearningAssetHandler) DeleteNode(c *gin.Context) {
	if err := h.service.DeleteNode(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("nodeId")); err != nil {
		writeServiceError(c, err)
		return
	}
	response.NoContent(c)
}
func (h *LearningAssetHandler) CreateEdge(c *gin.Context) {
	var input dto.KnowledgeEdgeInput
	if !bindJSON(c, &input) {
		return
	}
	out, err := h.service.CreateEdge(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Created(c, out)
}
func (h *LearningAssetHandler) DeleteEdge(c *gin.Context) {
	if err := h.service.DeleteEdge(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("edgeId")); err != nil {
		writeServiceError(c, err)
		return
	}
	response.NoContent(c)
}
func (h *LearningAssetHandler) UpdateArticle(c *gin.Context) {
	var input dto.KnowledgeArticleInput
	if !bindJSON(c, &input) {
		return
	}
	out, err := h.service.UpdateArticle(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("articleId"), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, out)
}

type LearningAssetHandler struct{ service *service.LearningAssetService }

func NewLearningAssetHandler(service *service.LearningAssetService) *LearningAssetHandler {
	return &LearningAssetHandler{service: service}
}
func (h *LearningAssetHandler) Generate(c *gin.Context) {
	result, err := h.service.Generate(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Accepted(c, result)
}
func (h *LearningAssetHandler) GeneratePlan(c *gin.Context) {
	result, err := h.service.GeneratePlan(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Accepted(c, result)
}
func (h *LearningAssetHandler) Get(c *gin.Context) {
	result, err := h.service.Get(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}
func (h *LearningAssetHandler) GetJob(c *gin.Context) {
	result, err := h.service.GetJob(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("jobId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}
