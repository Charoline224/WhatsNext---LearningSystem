package handler

import (
	"github.com/gin-gonic/gin"
	"whatsnext/backend/internal/dto"
	appmiddleware "whatsnext/backend/internal/middleware"
	"whatsnext/backend/internal/response"
	"whatsnext/backend/internal/service"
)

type ChatHandler struct{ service *service.ChatService }

func NewChatHandler(service *service.ChatService) *ChatHandler { return &ChatHandler{service: service} }
func (h *ChatHandler) Chat(c *gin.Context) {
	var input dto.ChatInput
	if !bindJSON(c, &input) {
		return
	}
	result, err := h.service.Chat(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}
