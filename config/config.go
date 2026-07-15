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
	DBHost                          string
	DBPort                          string
	DBUser                          string
	DBPassword                      string
	DBName                          string
	DBTLS                           string
	SMTPHost                        string
	SMTPPort                        string
	SMTPUsername                    string
	SMTPAppPassword                 string
	SMTPFrom                        string
	SMTPTLSPolicy                   string
	DefaultCompanyContactEmail      string
	ServerPort                      string
	GinMode                         string
	FrontendURL                     string
	JWTSecret                       string
	JWTExpiresHours                 int
	JWTExpiresMinutes               int
	InternalSupplyAPIURL            string
	InternalSupplyAPIToken          string
	InternalSupplyAPICookie         string
	InternalSupplyAPIBody           string
	InternalSupplyAPITimeoutSeconds int
	InternalSupplySyncEnabled       bool
	InternalSupplySyncHour          int
	InternalSupplySyncMinute        int
	InternalSupplySyncTimezone      string
	InternalSupplySyncRunOnStartup  bool
	GeminiAPIKey                    string
	GeminiModel                     string
	GeminiAPIBaseURL                string
	GeminiWebSearch                 bool
	GeminiMaxOutputTokens           int
	SupplyMappingTable              string
	VinmesAPIBaseURL                string
	VinmesAPIToken                  string
	VinmesAPITimeoutSeconds         int
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
		DBHost:                          getEnv("DB_HOST", "localhost"),
		DBPort:                          getEnv("DB_PORT", "3306"),
		DBUser:                          getEnv("DB_USER", "root"),
		DBPassword:                      getEnv("DB_PASSWORD", ""),
		DBName:                          getEnv("DB_NAME", "hospital_db"),
		DBTLS:                           getEnv("DB_TLS", ""),
		SMTPHost:                        getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:                        getEnv("SMTP_PORT", "587"),
		SMTPUsername:                    getEnv("SMTP_USERNAME", ""),
		SMTPAppPassword:                 getEnv("SMTP_APP_PASSWORD", ""),
		SMTPFrom:                        getEnv("SMTP_FROM", getEnv("SMTP_USERNAME", "")),
		SMTPTLSPolicy:                   getEnv("SMTP_TLS_POLICY", "mandatory"),
		DefaultCompanyContactEmail:      getEnv("DEFAULT_COMPANY_CONTACT_EMAIL", getEnv("SMTP_FROM", getEnv("SMTP_USERNAME", ""))),
		ServerPort:                      serverPort,
		GinMode:                         getEnv("GIN_MODE", "debug"),
		FrontendURL:                     getEnv("FRONTEND_URL", "http://localhost:5173"),
		JWTSecret:                       getEnv("JWT_SECRET", ""),
		JWTExpiresHours:                 getEnvAsInt("JWT_EXPIRES_HOURS", 8),
		JWTExpiresMinutes:               getEnvAsInt("JWT_EXPIRES_MINUTES", 0),
		InternalSupplyAPIURL:            getEnv("INTERNAL_SUPPLY_API_URL", ""),
		InternalSupplyAPIToken:          getEnv("INTERNAL_SUPPLY_API_TOKEN", ""),
		InternalSupplyAPICookie:         getEnv("INTERNAL_SUPPLY_API_COOKIE", ""),
		InternalSupplyAPIBody:           getEnv("INTERNAL_SUPPLY_API_BODY", "{}"),
		InternalSupplyAPITimeoutSeconds: getEnvAsInt("INTERNAL_SUPPLY_API_TIMEOUT_SECONDS", 120),
		InternalSupplySyncEnabled:       getEnvAsBool("INTERNAL_SUPPLY_SYNC_ENABLED", false),
		InternalSupplySyncHour:          getEnvAsInt("INTERNAL_SUPPLY_SYNC_HOUR", 20),
		InternalSupplySyncMinute:        getEnvAsInt("INTERNAL_SUPPLY_SYNC_MINUTE", 0),
		InternalSupplySyncTimezone:      getEnv("INTERNAL_SUPPLY_SYNC_TIMEZONE", "Asia/Bangkok"),
		InternalSupplySyncRunOnStartup:  getEnvAsBool("INTERNAL_SUPPLY_SYNC_RUN_ON_STARTUP", false),
		GeminiAPIKey:                    getEnv("GEMINI_API_KEY", ""),
		GeminiModel:                     getEnv("GEMINI_MODEL", "gemini-2.5-flash-lite"),
		GeminiAPIBaseURL:                getEnv("GEMINI_API_BASE_URL", "https://generativelanguage.googleapis.com/v1beta"),
		GeminiWebSearch:                 getEnvAsBool("GEMINI_WEB_SEARCH", false),
		GeminiMaxOutputTokens:           getEnvAsInt("GEMINI_MAX_OUTPUT_TOKENS", 4096),
		SupplyMappingTable:              getEnv("SUPPLY_MAPPING_TABLE", "mapping2"),
		VinmesAPIBaseURL:                getEnv("VINMES_API_BASE_URL", "http://108.108.108.251/api/v1/resource"),
		VinmesAPIToken:                  getEnv("VINMES_API_TOKEN", ""),
		VinmesAPITimeoutSeconds:         getEnvAsInt("VINMES_API_TIMEOUT_SECONDS", 60),
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
	if value, exists := os.LookupEnv(key); exists {
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

func getEnvAsBool(key string, defaultValue bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return defaultValue
	}

	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return defaultValue
	}
}
