package handlers

import (
	"net/http"

	"bv108-consumables-management-backend/internal/models"
	"bv108-consumables-management-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	userRepo    *models.UserRepository
	jwtSecret   []byte
	geminiProxy *services.GeminiProxyService
}

func NewReportHandler(userRepo *models.UserRepository, jwtSecret string, geminiProxy *services.GeminiProxyService) *ReportHandler {
	return &ReportHandler{
		userRepo:    userRepo,
		jwtSecret:   []byte(jwtSecret),
		geminiProxy: geminiProxy,
	}
}

func (h *ReportHandler) requireAuthenticatedUser(c *gin.Context) bool {
	if _, err := getCurrentUserFromAuthorizationHeader(c, h.userRepo, h.jwtSecret); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: err.Error(),
		})
		return false
	}

	return true
}

func (h *ReportHandler) GenerateGeminiCompare(c *gin.Context) {
	if !h.requireAuthenticatedUser(c) {
		return
	}

	if h.geminiProxy == nil || !h.geminiProxy.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "UNAVAILABLE",
			Message: "Gemini backend is not configured",
		})
		return
	}

	var payload services.GeminiProxyRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "Invalid Gemini request payload",
		})
		return
	}

	if len(payload.Contents) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "contents is required",
		})
		return
	}

	resp, status, err := h.geminiProxy.GenerateContent(payload)
	if err != nil {
		c.JSON(status, ErrorResponse{
			Error:   "GEMINI_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
