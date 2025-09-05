package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// Application
	AppName  string
	Env      string
	Port     string
	LogLevel string

	// Database
	DB DatabaseConfig

	// JWT
	JWT JWTConfig

	// AWS
	AWS AWSConfig

	// CORS
	CORS CORSConfig

	// Rate Limiting
	RateLimit RateLimitConfig

	// File Upload
	Upload UploadConfig
}

type DatabaseConfig struct {
	Host         string
	Port         string
	Name         string
	User         string
	Password     string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

type JWTConfig struct {
	Secret      string
	ExpiryHours int
}

type AWSConfig struct {
	Region            string
	AccessKeyID       string
	SecretAccessKey   string
	S3BucketName      string
	S3Region          string
	EndpointURL       string
	PublicEndpointURL string
	S3ForcePathStyle  bool
}

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

type RateLimitConfig struct {
	RequestsPerMinute int
}

type UploadConfig struct {
	MaxFileSizeMB    int
	AllowedFileTypes []string
}

func Load() *Config {
	return &Config{
		AppName:  getEnv("APP_NAME", "hackathon-api"),
		Env:      getEnv("ENV", "development"),
		Port:     getEnv("PORT", "8080"),
		LogLevel: getEnv("LOG_LEVEL", "info"),

		DB: DatabaseConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnv("DB_PORT", "5432"),
			Name:         getEnv("DB_NAME", "hackathon_db"),
			User:         getEnv("DB_USER", "postgres"),
			Password:     getEnv("DB_PASSWORD", "password"),
			SSLMode:      getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 25),
		},

		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", "your-super-secret-jwt-key"),
			ExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 24),
		},

		AWS: AWSConfig{
			Region:            getEnv("AWS_REGION", "ap-northeast-1"),
			AccessKeyID:       getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey:   getEnv("AWS_SECRET_ACCESS_KEY", ""),
			S3BucketName:      getEnv("S3_BUCKET_NAME", "hackathon-uploads"),
			S3Region:          getEnv("S3_REGION", "ap-northeast-1"),
			EndpointURL:       getEnv("AWS_ENDPOINT_URL", ""),
			PublicEndpointURL: getEnv("AWS_PUBLIC_ENDPOINT_URL", ""),
			S3ForcePathStyle:  getEnvBool("AWS_S3_FORCE_PATH_STYLE", true),
		},

		CORS: CORSConfig{
			AllowedOrigins: getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
			AllowedMethods: getEnvSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			AllowedHeaders: getEnvSlice("CORS_ALLOWED_HEADERS", []string{"Origin", "Content-Type", "Accept", "Authorization"}),
		},

		RateLimit: RateLimitConfig{
			RequestsPerMinute: getEnvInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 60),
		},

		Upload: UploadConfig{
			MaxFileSizeMB:    getEnvInt("MAX_FILE_SIZE_MB", 10),
			AllowedFileTypes: getEnvSlice("ALLOWED_FILE_TYPES", []string{"jpg", "jpeg", "png", "gif", "pdf", "doc", "docx", "txt"}),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		lower := strings.ToLower(strings.TrimSpace(value))
		return lower == "1" || lower == "true" || lower == "yes" || lower == "on"
	}
	return defaultValue
}
