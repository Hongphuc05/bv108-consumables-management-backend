package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	DBTLS           string
	SMTPHost        string
	SMTPPort        string
	SMTPUsername    string
	SMTPAppPassword string
	SMTPFrom        string
	ServerPort      string
	GinMode         string
	FrontendURL     string
	JWTSecret       string
	JWTExpiresHours int
}

var AppConfig *Config

// LoadConfig loads configuration from environment variables
func LoadConfig() error {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	serverPort := getEnv("PORT", getEnv("SERVER_PORT", "8080"))

	AppConfig = &Config{
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "3306"),
		DBUser:          getEnv("DB_USER", "root"),
		DBPassword:      getEnv("DB_PASSWORD", ""),
		DBName:          getEnv("DB_NAME", "hospital_db"),
		DBTLS:           getEnv("DB_TLS", ""),
		SMTPHost:        getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:        getEnv("SMTP_PORT", "587"),
		SMTPUsername:    getEnv("SMTP_USERNAME", "kaelbridmon1990@gmail.com"),
		SMTPAppPassword: getEnv("SMTP_APP_PASSWORD", ""),
		SMTPFrom:        getEnv("SMTP_FROM", "kaelbridmon1990@gmail.com"),
		ServerPort:      serverPort,
		GinMode:         getEnv("GIN_MODE", "debug"),
		FrontendURL:     getEnv("FRONTEND_URL", "http://localhost:5173"),
		JWTSecret:       getEnv("JWT_SECRET", "key_for_jwt"),
		JWTExpiresHours: getEnvAsInt("JWT_EXPIRES_HOURS", 8),
	}

	return nil
}

// GetDSN returns the MySQL Data Source Name
func (c *Config) GetDSN() string {
	tlsMode := c.DBTLS
	if tlsMode == "" && strings.Contains(c.DBHost, ".mysql.database.azure.com") {
		tlsMode = "true"
	}

	baseDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
	)

	if tlsMode != "" {
		return baseDSN + "&tls=" + tlsMode
	}

	return baseDSN
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return parsedValue
}
