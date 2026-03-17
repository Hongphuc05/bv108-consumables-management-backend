package handlers

import (
	"fmt"
	"strings"

	"bv108-consumables-management-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func getCurrentUserFromAuthorizationHeader(c *gin.Context, userRepo *models.UserRepository, jwtSecret []byte) (*models.UserProfile, error) {
	userID, err := getUserIDFromAuthorizationHeader(c, jwtSecret)
	if err != nil {
		return nil, err
	}

	user, err := userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	profile := user.ToProfile()
	return &profile, nil
}

func getUserIDFromAuthorizationHeader(c *gin.Context, jwtSecret []byte) (int64, error) {
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
		return jwtSecret, nil
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

