package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type internalSupplySyncRunner interface {
	RunOnce(ctx context.Context) (int, error)
}

type InternalSupplySyncHandler struct {
	runner internalSupplySyncRunner
}

func NewInternalSupplySyncHandler(runner internalSupplySyncRunner) *InternalSupplySyncHandler {
	return &InternalSupplySyncHandler{runner: runner}
}

func (h *InternalSupplySyncHandler) SyncNow(c *gin.Context) {
	count, err := h.runner.RunOnce(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL_SUPPLY_SYNC_FAILED",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Internal supply sync completed",
		"count":   count,
	})
}
