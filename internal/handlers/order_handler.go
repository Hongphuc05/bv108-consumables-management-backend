package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"bv108-consumables-management-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type OrderHandler struct {
	repo      *models.OrderRepository
	userRepo  *models.UserRepository
	jwtSecret []byte
}

type CreateForecastOrdersRequest struct {
	Items []CreateOrderItemRequest `json:"items" binding:"required"`
}

type CreateOrderItemRequest struct {
	NhaThau    string `json:"nhaThau"`
	MaQuanLy   string `json:"maQuanLy"`
	MaVtytCu   string `json:"maVtytCu"`
	TenVtytBv  string `json:"tenVtytBv"`
	MaHieu     string `json:"maHieu"`
	HangSx     string `json:"hangSx"`
	DonViTinh  string `json:"donViTinh"`
	QuyCach    string `json:"quyCach"`
	DotGoiHang int    `json:"dotGoiHang"`
	Email      string `json:"email"`
}

type PlaceOrdersRequest struct {
	OrderIDs []int64 `json:"orderIds" binding:"required"`
}

func NewOrderHandler(repo *models.OrderRepository, userRepo *models.UserRepository, jwtSecret string) *OrderHandler {
	return &OrderHandler{
		repo:      repo,
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

func (h *OrderHandler) GetPendingOrders(c *gin.Context) {
	if _, err := h.getCurrentUser(c); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	orders, err := h.repo.ListPendingOrders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": orders})
}

func (h *OrderHandler) GetOrderHistory(c *gin.Context) {
	if _, err := h.getCurrentUser(c); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	history, err := h.repo.ListOrderHistory()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": history})
}

func (h *OrderHandler) CreateForecastOrders(c *gin.Context) {
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	var req CreateForecastOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Invalid forecast order payload"})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "At least one order item is required"})
		return
	}

	inputs := make([]models.CreatePendingOrderInput, 0, len(req.Items))
	approvalTime := time.Now().Format(time.RFC3339)
	for _, item := range req.Items {
		if item.DotGoiHang < 1 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "dotGoiHang must be greater than 0"})
			return
		}

		inputs = append(inputs, models.CreatePendingOrderInput{
			NhaThau:      sanitizeText(item.NhaThau),
			MaQuanLy:     sanitizeText(item.MaQuanLy),
			MaVtytCu:     sanitizeText(item.MaVtytCu),
			TenVtytBv:    sanitizeText(item.TenVtytBv),
			MaHieu:       sanitizeText(item.MaHieu),
			HangSx:       sanitizeText(item.HangSx),
			DonViTinh:    sanitizeText(item.DonViTinh),
			QuyCach:      sanitizeText(item.QuyCach),
			DotGoiHang:   item.DotGoiHang,
			Email:        sanitizeText(item.Email),
			Source:       models.OrderSourceForecast,
			Approver:     &models.OrderActor{ID: currentUser.ID, Username: currentUser.Username, Email: currentUser.Email},
			CreatedBy:    models.OrderActor{ID: currentUser.ID, Username: currentUser.Username, Email: currentUser.Email},
			ApprovalTime: approvalTime,
		})
	}

	if err := h.repo.AddForecastOrders(inputs); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Forecast orders added to pending list successfully",
		"count":   len(inputs),
	})
}

func (h *OrderHandler) CreateManualOrder(c *gin.Context) {
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	var req CreateOrderItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Invalid manual order payload"})
		return
	}

	if req.DotGoiHang < 1 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "dotGoiHang must be greater than 0"})
		return
	}

	if sanitizeText(req.NhaThau) == "" || sanitizeText(req.MaVtytCu) == "" || sanitizeText(req.TenVtytBv) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "nhaThau, maVtytCu, tenVtytBv are required"})
		return
	}

	input := models.CreatePendingOrderInput{
		NhaThau:    sanitizeText(req.NhaThau),
		MaQuanLy:   sanitizeText(req.MaQuanLy),
		MaVtytCu:   sanitizeText(req.MaVtytCu),
		TenVtytBv:  sanitizeText(req.TenVtytBv),
		MaHieu:     sanitizeText(req.MaHieu),
		HangSx:     sanitizeText(req.HangSx),
		DonViTinh:  sanitizeText(req.DonViTinh),
		QuyCach:    sanitizeText(req.QuyCach),
		DotGoiHang: req.DotGoiHang,
		Email:      sanitizeText(req.Email),
		Source:     models.OrderSourceManual,
		CreatedBy:  models.OrderActor{ID: currentUser.ID, Username: currentUser.Username, Email: currentUser.Email},
	}

	if err := h.repo.AddManualOrder(input); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Manual order added to pending list successfully"})
}

func (h *OrderHandler) PlaceOrders(c *gin.Context) {
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	var req PlaceOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Invalid place order payload"})
		return
	}

	if len(req.OrderIDs) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "At least one order id is required"})
		return
	}

	placedCount, err := h.repo.PlaceOrders(req.OrderIDs, models.OrderActor{
		ID:       currentUser.ID,
		Username: currentUser.Username,
		Email:    currentUser.Email,
	})
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "no pending orders found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Orders placed successfully",
		"placedCount": placedCount,
	})
}

func (h *OrderHandler) getCurrentUser(c *gin.Context) (*models.UserProfile, error) {
	userID, err := h.getUserIDFromAuthorizationHeader(c)
	if err != nil {
		return nil, err
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	profile := user.ToProfile()
	return &profile, nil
}

func (h *OrderHandler) getUserIDFromAuthorizationHeader(c *gin.Context) (int64, error) {
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if authHeader == "" {
		return 0, fmt.Errorf("missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return 0, fmt.Errorf("invalid authorization header format")
	}

	tokenString := strings.TrimSpace(parts[1])
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

func sanitizeText(value string) string {
	return strings.TrimSpace(value)
}
