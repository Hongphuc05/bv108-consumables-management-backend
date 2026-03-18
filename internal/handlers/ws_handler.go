package handlers

import (
	"fmt"
	"log"
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

	log.Printf("[WS] Token received (length: %d, first 50chars: %s)", len(tokenString), tokenString[:min(50, len(tokenString))])

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Check if the signing method is HS256
		if token.Method.Alg() != "HS256" {
			log.Printf("[WS] ERROR: Wrong signing method: %s", token.Method.Alg())
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return h.jwtSecret, nil
	})
	if err != nil {
		log.Printf("[WS] ERROR: JWT parse failed: %v", err)
		return 0, fmt.Errorf("invalid token: %w", err)
	}
	if !token.Valid {
		log.Printf("[WS] ERROR: Token is invalid (valid=%v)", token.Valid)
		return 0, fmt.Errorf("token is not valid")
	}
	log.Printf("[WS] Token parsed successfully")

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Printf("[WS] ERROR: Claims type assertion failed")
		return 0, fmt.Errorf("invalid token claims")
	}

	subValue, exists := claims["sub"]
	if !exists {
		log.Printf("[WS] ERROR: Missing 'sub' claim in token")
		return 0, fmt.Errorf("missing subject in token")
	}

	userID, err := convertClaimToInt64(subValue)
	if err != nil {
		log.Printf("[WS] ERROR: Failed to convert sub to int64: %v", err)
		return 0, fmt.Errorf("invalid subject in token")
	}

	log.Printf("[WS] User ID extracted: %d", userID)
	return userID, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
