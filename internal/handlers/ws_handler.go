package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"bv108-consumables-management-backend/internal/models"
	"bv108-consumables-management-backend/internal/realtime"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	userRepo  *models.UserRepository
	jwtSecret []byte
	hub       *realtime.Hub
	upgrader  websocket.Upgrader
}

func NewWSHandler(userRepo *models.UserRepository, jwtSecret string, hub *realtime.Hub) *WSHandler {
	return &WSHandler{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
		hub:       hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *WSHandler) Handle(c *gin.Context) {
	userID, err := h.getUserIDFromRequest(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	if _, err := h.userRepo.GetByID(userID); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: "User not found"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "UPGRADE_FAILED", Message: err.Error()})
		return
	}

	h.hub.Register(userID, conn)
}

func (h *WSHandler) getUserIDFromRequest(c *gin.Context) (int64, error) {
	tokenString := strings.TrimSpace(c.Query("token"))
	if tokenString == "" {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			tokenString = strings.TrimSpace(parts[1])
		}
	}
	if tokenString == "" {
		return 0, fmt.Errorf("missing bearer token")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return h.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid token claims")
	}

	subValue, exists := claims["sub"]
	if !exists {
		return 0, fmt.Errorf("missing subject in token")
	}

	userID, err := convertClaimToInt64(subValue)
	if err != nil {
		return 0, fmt.Errorf("invalid subject in token")
	}

	return userID, nil
}
