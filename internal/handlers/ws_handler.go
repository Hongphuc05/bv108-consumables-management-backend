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

const websocketAuthProtocol = "bv108.auth"

type WSHandler struct {
	userRepo  *models.UserRepository
	jwtSecret []byte
	hub       *realtime.Hub
	upgrader  websocket.Upgrader
}

func NewWSHandler(userRepo *models.UserRepository, jwtSecret string, hub *realtime.Hub, frontendURL string) *WSHandler {
	allowedOrigins := map[string]struct{}{
		"http://localhost:5173": {},
		"http://localhost:5174": {},
		"http://localhost:3000": {},
	}
	if origin := strings.TrimSpace(frontendURL); origin != "" {
		allowedOrigins[origin] = struct{}{}
	}

	return &WSHandler{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
		hub:       hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			Subprotocols:    []string{websocketAuthProtocol},
			CheckOrigin: func(r *http.Request) bool {
				origin := strings.TrimSpace(r.Header.Get("Origin"))
				if origin == "" {
					return true
				}
				_, ok := allowedOrigins[origin]
				return ok
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

	if _, err := loadActiveUserByID(h.userRepo, userID); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
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
	tokenString := ""
	subprotocols := websocket.Subprotocols(c.Request)
	if len(subprotocols) >= 2 && strings.EqualFold(strings.TrimSpace(subprotocols[0]), websocketAuthProtocol) {
		tokenString = strings.TrimSpace(subprotocols[1])
	}
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
		if token.Method.Alg() != "HS256" {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return h.jwtSecret, nil
	})
	if err != nil {
		return 0, fmt.Errorf("invalid token: %w", err)
	}
	if !token.Valid {
		return 0, fmt.Errorf("token is not valid")
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
