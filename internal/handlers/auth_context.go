package handlers

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"bv108-consumables-management-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const currentUserCacheTTL = 30 * time.Second

type currentUserCacheEntry struct {
	profile   models.UserProfile
	expiresAt time.Time
}

var currentUserCache sync.Map

func getCurrentUserFromAuthorizationHeader(c *gin.Context, userRepo *models.UserRepository, jwtSecret []byte) (*models.UserProfile, error) {
	userID, err := getUserIDFromAuthorizationHeader(c, jwtSecret)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if cached, ok := currentUserCache.Load(userID); ok {
		entry := cached.(currentUserCacheEntry)
		if now.Before(entry.expiresAt) {
			profile := entry.profile
			return &profile, nil
		}
		currentUserCache.Delete(userID)
	}

	user, err := userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	profile := user.ToProfile()
	currentUserCache.Store(userID, currentUserCacheEntry{
		profile:   profile,
		expiresAt: now.Add(currentUserCacheTTL),
	})

	return &profile, nil
}

func invalidateCurrentUserCache(userID int64) {
	currentUserCache.Delete(userID)
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
