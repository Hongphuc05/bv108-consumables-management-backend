package handlers

import (
	"context"
	"net/http"

	"bv108-consumables-management-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type internalSupplySyncRunner interface {
	RunOnce(ctx context.Context) (int, error)
}

type InternalSupplySyncHandler struct {
	runner    internalSupplySyncRunner
	userRepo  *models.UserRepository
	jwtSecret []byte
}

func NewInternalSupplySyncHandler(runner internalSupplySyncRunner, userRepo *models.UserRepository, jwtSecret string) *InternalSupplySyncHandler {
	return &InternalSupplySyncHandler{
		runner:    runner,
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

func (h *InternalSupplySyncHandler) SyncNow(c *gin.Context) {
	currentUser, err := getCurrentUserFromAuthorizationHeader(c, h.userRepo, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	if !canRunInternalSupplySyncRole(currentUser.Role) {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "FORBIDDEN", Message: "Only Admin can run internal supply sync"})
		return
	}

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

func canRunInternalSupplySyncRole(role string) bool {
	return normalizeRoleForPermissions(role) == RoleAdmin
}
