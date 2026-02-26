package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"bv108-consumables-management-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	RoleNhanVien   = "nhan_vien"
	RoleTruongKhoa = "truong_khoa"
)

type AuthHandler struct {
	userRepo        *models.UserRepository
	jwtSecret       []byte
	jwtExpiresHours int
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token     string             `json:"token"`
	ExpiresAt string             `json:"expiresAt"`
	User      models.UserProfile `json:"user"`
}

type UpdateProfileRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
}

func NewAuthHandler(userRepo *models.UserRepository, jwtSecret string, jwtExpiresHours int) *AuthHandler {
	return &AuthHandler{
		userRepo:        userRepo,
		jwtSecret:       []byte(jwtSecret),
		jwtExpiresHours: jwtExpiresHours,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "Invalid register payload",
		})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))

	if req.Username == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USERNAME", Message: "Username is required"})
		return
	}

	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_EMAIL", Message: "Email is invalid"})
		return
	}

	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_PASSWORD", Message: "Password must be at least 6 characters"})
		return
	}

	if !isValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_ROLE", Message: "Role must be nhan_vien or truong_khoa"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "HASH_ERROR", Message: "Failed to hash password"})
		return
	}

	createdUser, err := h.userRepo.Create(req.Username, req.Email, string(passwordHash), req.Role)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			c.JSON(http.StatusConflict, ErrorResponse{Error: "DUPLICATE_USER", Message: "Email or username already exists"})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	token, expiresAt, err := h.generateToken(createdUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "TOKEN_ERROR", Message: "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		User:      createdUser.ToProfile(),
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Invalid login payload"})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_EMAIL", Message: "Email is invalid"})
		return
	}

	user, err := h.userRepo.GetByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "INVALID_CREDENTIALS", Message: "Email or password is incorrect"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "ACCOUNT_DISABLED", Message: "User account is disabled"})
		return
	}

	if compareErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); compareErr != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "INVALID_CREDENTIALS", Message: "Email or password is incorrect"})
		return
	}

	token, expiresAt, err := h.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "TOKEN_ERROR", Message: "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		User:      user.ToProfile(),
	})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, err := h.getUserIDFromAuthorizationHeader(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Invalid update profile payload"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Username == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USERNAME", Message: "Username is required"})
		return
	}

	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_EMAIL", Message: "Email is invalid"})
		return
	}

	updatedUser, err := h.userRepo.UpdateProfile(userID, req.Username, req.Email)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			c.JSON(http.StatusConflict, ErrorResponse{Error: "DUPLICATE_USER", Message: "Email or username already exists"})
			return
		}

		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "NOT_FOUND", Message: "User not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user":    updatedUser.ToProfile(),
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, err := h.getUserIDFromAuthorizationHeader(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "NOT_FOUND", Message: "User not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user.ToProfile(),
	})
}

func (h *AuthHandler) generateToken(user *models.User) (string, time.Time, error) {
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(time.Duration(h.jwtExpiresHours) * time.Hour)

	claims := jwt.MapClaims{
		"sub":      user.ID,
		"email":    user.Email,
		"username": user.Username,
		"role":     user.Role,
		"iat":      issuedAt.Unix(),
		"exp":      expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(h.jwtSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("error signing token: %w", err)
	}

	return signedToken, expiresAt, nil
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func isValidRole(role string) bool {
	return role == RoleNhanVien || role == RoleTruongKhoa
}

func (h *AuthHandler) getUserIDFromAuthorizationHeader(c *gin.Context) (int64, error) {
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

func convertClaimToInt64(value interface{}) (int64, error) {
	switch typedValue := value.(type) {
	case float64:
		return int64(typedValue), nil
	case int64:
		return typedValue, nil
	case int:
		return int64(typedValue), nil
	case string:
		parsed, err := strconv.ParseInt(typedValue, 10, 64)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported claim type")
	}
}
