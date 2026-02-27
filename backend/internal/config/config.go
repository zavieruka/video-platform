package config

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
)

// Config holds all application configuration
type Config struct {
	// GCP Configuration
	GCPProjectID        string
	GCPRegion           string
	FirestoreDatabaseID string
	SourceBucketName    string
	ProcessedBucketName string
	ServiceAccountEmail string

	// Application Configuration
	Port        string
	Environment string
	LogLevel    string

	// Upload Configuration
	MaxUploadSizeMB     int
	AllowedVideoFormats []string
	UploadURLExpiryHrs  int

	// GCP Clients (initialized after validation)
	FirestoreClient *firestore.Client
	StorageClient   *storage.Client
}

// Load reads configuration from environment variables and validates them
func Load() (*Config, error) {
	cfg := &Config{
		GCPProjectID:        getEnv("GCP_PROJECT_ID", ""),
		GCPRegion:           getEnv("GCP_REGION", "us-central1"),
		FirestoreDatabaseID: getEnv("FIRESTORE_DATABASE_ID", "(default)"),
		SourceBucketName:    getEnv("SOURCE_BUCKET_NAME", ""),
		ProcessedBucketName: getEnv("PROCESSED_BUCKET_NAME", ""),
		ServiceAccountEmail: getEnv("SERVICE_ACCOUNT_EMAIL", ""),
		Port:                getEnv("PORT", "8080"),
		Environment:         getEnv("ENVIRONMENT", "dev"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		MaxUploadSizeMB:     getEnvAsInt("MAX_UPLOAD_SIZE_MB", 500),
		AllowedVideoFormats: getEnvAsSlice("ALLOWED_VIDEO_FORMATS", []string{"mp4", "mov", "avi", "mkv"}),
		UploadURLExpiryHrs:  getEnvAsInt("UPLOAD_URL_EXPIRY_HOURS", 1),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.GCPProjectID == "" {
		return fmt.Errorf("GCP_PROJECT_ID is required")
	}

	if c.SourceBucketName == "" {
		return fmt.Errorf("SOURCE_BUCKET_NAME is required")
	}

	if c.ProcessedBucketName == "" {
		return fmt.Errorf("PROCESSED_BUCKET_NAME is required")
	}

	if c.ServiceAccountEmail == "" {
		return fmt.Errorf("SERVICE_ACCOUNT_EMAIL is required")
	}

	if _, err := strconv.Atoi(c.Port); err != nil {
		return fmt.Errorf("PORT must be a valid number: %w", err)
	}

	validEnvs := map[string]bool{"dev": true, "staging": true, "production": true}
	if !validEnvs[c.Environment] {
		return fmt.Errorf("ENVIRONMENT must be one of: dev, staging, production (got: %s)", c.Environment)
	}

	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error (got: %s)", c.LogLevel)
	}

	return nil
}

// InitializeGCPClients creates and initializes GCP service clients
// This should be called after Load() and Validate()
func (c *Config) InitializeGCPClients(ctx context.Context) error {
	var err error

	c.FirestoreClient, err = firestore.NewClient(ctx, c.GCPProjectID)
	if err != nil {
		return fmt.Errorf("failed to create Firestore client: %w", err)
	}

	c.StorageClient, err = storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Storage client: %w", err)
	}

	return nil
}

// Close gracefully closes all GCP client connections
func (c *Config) Close() error {
	var errs []error

	if c.FirestoreClient != nil {
		if err := c.FirestoreClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Firestore client: %w", err))
		}
	}

	if c.StorageClient != nil {
		if err := c.StorageClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Storage client: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing clients: %v", errs)
	}

	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "dev"
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func (c *Config) GetAddress() string {
	return fmt.Sprintf(":%s", c.Port)
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as int with a fallback default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvAsSlice gets an environment variable as a comma-separated slice
func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	parts := make([]string, 0)
	for _, part := range splitAndTrim(valueStr, ",") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return defaultValue
	}
	return parts
}

func splitAndTrim(s string, sep string) []string {
	parts := make([]string, 0)
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		parts = append(parts, trimmed)
	}
	return parts
}

func splitString(s string, sep string) []string {
	if s == "" {
		return []string{}
	}
	result := []string{}
	current := ""
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, current)
			current = ""
			i += len(sep) - 1
		} else {
			current += string(s[i])
		}
	}
	result = append(result, current)
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
