package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"bv108-consumables-management-backend/config"
	"bv108-consumables-management-backend/internal/database"
	"bv108-consumables-management-backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	roleFilter := flag.String("role", "", "optional role filter")
	emailFilter := flag.String("email", "", "optional email filter")
	flag.Parse()

	if err := config.LoadConfig(); err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := database.InitDB(); err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer database.CloseDB()

	userRepo := models.NewUserRepository(database.DB)
	users, err := userRepo.ListActiveUsers()
	if err != nil {
		log.Fatalf("list users: %v", err)
	}
	if len(users) == 0 {
		log.Fatal("no active users found")
	}

	user, err := pickUser(users, *roleFilter, *emailFilter)
	if err != nil {
		log.Fatal(err)
	}

	token, err := signToken(user)
	if err != nil {
		log.Fatalf("sign token: %v", err)
	}

	fmt.Print(token)
}

func pickUser(users []models.UserProfile, roleFilter, emailFilter string) (*models.UserProfile, error) {
	normalizedRole := strings.ToLower(strings.TrimSpace(roleFilter))
	normalizedEmail := strings.ToLower(strings.TrimSpace(emailFilter))

	for _, user := range users {
		if normalizedEmail != "" && strings.ToLower(strings.TrimSpace(user.Email)) != normalizedEmail {
			continue
		}
		if normalizedRole != "" && strings.ToLower(strings.TrimSpace(user.Role)) != normalizedRole {
			continue
		}
		userCopy := user
		return &userCopy, nil
	}

	if normalizedRole != "" || normalizedEmail != "" {
		return nil, fmt.Errorf("no active user matches role=%q email=%q", roleFilter, emailFilter)
	}

	user := users[0]
	return &user, nil
}

func signToken(user *models.UserProfile) (string, error) {
	issuedAt := time.Now()
	expiresIn := time.Duration(config.AppConfig.JWTExpiresMinutes) * time.Minute
	if expiresIn <= 0 {
		expiresIn = time.Duration(config.AppConfig.JWTExpiresHours) * time.Hour
	}
	if expiresIn <= 0 {
		expiresIn = 8 * time.Hour
	}

	claims := jwt.MapClaims{
		"sub":      user.ID,
		"email":    user.Email,
		"username": user.Username,
		"role":     user.Role,
		"iat":      issuedAt.Unix(),
		"exp":      issuedAt.Add(expiresIn).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}
