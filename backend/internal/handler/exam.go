package handler

import (
	"github.com/gin-gonic/gin"
	"whatsnext/backend/internal/dto"
	appmiddleware "whatsnext/backend/internal/middleware"
	"whatsnext/backend/internal/response"
	"whatsnext/backend/internal/service"
)

type ExamHandler struct{ service *service.ExamService }

func NewExamHandler(service *service.ExamService) *ExamHandler { return &ExamHandler{service: service} }
func (h *ExamHandler) Get(c *gin.Context) {
	out, err := h.service.Get(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, out)
}
func (h *ExamHandler) Feedback(c *gin.Context) {
	var input dto.ExamFeedbackInput
	if !bindJSON(c, &input) {
		return
	}
	out, err := h.service.Feedback(c.Request.Context(), appmiddleware.UserID(c), c.Param("spaceId"), c.Param("questionId"), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, out)
}
