package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"kratos-boilerplate/internal/conf"
)

// Validator configuration validator interface
type Validator interface {
	Validate(config interface{}) error
}

// DefaultValidator default configuration validator
type DefaultValidator struct{}

// NewValidator creates default configuration validator
func NewValidator() Validator {
	return &DefaultValidator{}
}

// Validate validates configuration
func (v *DefaultValidator) Validate(config interface{}) error {
	// Here you can implement generic configuration validation logic
	// Currently returns nil, specific validation logic is implemented by individual validators
	return nil
}

// ConfigValidator configuration validator
type ConfigValidator struct {
	config *conf.Bootstrap
}

// NewConfigValidator creates configuration validator
func NewConfigValidator(config *conf.Bootstrap) *ConfigValidator {
	return &ConfigValidator{
		config: config,
	}
}

// Validate validates configuration
func (v *ConfigValidator) Validate() error {
	if err := v.validateServer(); err != nil {
		return fmt.Errorf("server configuration validation failed: %w", err)
	}

	if err := v.validateData(); err != nil {
		return fmt.Errorf("data layer configuration validation failed: %w", err)
	}

	if err := v.validateAuth(); err != nil {
		return fmt.Errorf("authentication configuration validation failed: %w", err)
	}

	if err := v.validateSecurity(); err != nil {
		return fmt.Errorf("security configuration validation failed: %w", err)
	}

	return nil
}

// validateServer validates server configuration
func (v *ConfigValidator) validateServer() error {
	if v.config.Server == nil {
		return fmt.Errorf("server configuration cannot be empty")
	}

	if v.config.Server.Http == nil {
		return fmt.Errorf("HTTP server configuration cannot be empty")
	}

	if v.config.Server.Http.Addr == "" {
		return fmt.Errorf("HTTP server address cannot be empty")
	}

	if v.config.Server.Grpc == nil {
		return fmt.Errorf("gRPC server configuration cannot be empty")
	}

	if v.config.Server.Grpc.Addr == "" {
		return fmt.Errorf("gRPC server address cannot be empty")
	}

	return nil
}

// validateData validates data layer configuration
func (v *ConfigValidator) validateData() error {
	if v.config.Data == nil {
		return fmt.Errorf("data layer configuration cannot be empty")
	}

	// Validate database configuration
	if err := v.validateDatabase(); err != nil {
		return err
	}

	// Validate Redis configuration
	if err := v.validateRedis(); err != nil {
		return err
	}

	return nil
}

// validateDatabase validates database configuration
func (v *ConfigValidator) validateDatabase() error {
	db := v.config.Data.Database
	if db == nil {
		return fmt.Errorf("database configuration cannot be empty")
	}

	if db.Driver == "" {
		return fmt.Errorf("database driver cannot be empty")
	}

	if db.Source == "" {
		return fmt.Errorf("database connection string cannot be empty")
	}

	// Validate database connection string format
	if _, err := url.Parse(db.Source); err != nil {
		return fmt.Errorf("invalid database connection string format: %w", err)
	}

	// Production environment security checks
	if isProduction() {
		if strings.Contains(db.Source, "sslmode=disable") {
			return fmt.Errorf("production environment must not disable SSL connection")
		}

		if strings.Contains(db.Source, "localhost") || strings.Contains(db.Source, "127.0.0.1") {
			return fmt.Errorf("production environment should not use local database connection")
		}
	}

	return nil
}

// validateRedis validates Redis configuration
func (v *ConfigValidator) validateRedis() error {
	redis := v.config.Data.Redis
	if redis == nil {
		return fmt.Errorf("Redis configuration cannot be empty")
	}

	if redis.Addr == "" {
		return fmt.Errorf("Redis address cannot be empty")
	}

	// Production environment security checks
	if isProduction() {
		// Check Redis password via environment variable
		if os.Getenv("REDIS_PASSWORD") == "" {
			return fmt.Errorf("production environment Redis must set REDIS_PASSWORD environment variable")
		}

		if strings.Contains(redis.Addr, "localhost") || strings.Contains(redis.Addr, "127.0.0.1") {
			return fmt.Errorf("production environment should not use local Redis connection")
		}
	}

	return nil
}

// validateAuth validates authentication configuration
func (v *ConfigValidator) validateAuth() error {
	auth := v.config.Auth
	if auth == nil {
		return fmt.Errorf("authentication configuration cannot be empty")
	}

	if auth.JwtSecretKey == "" {
		return fmt.Errorf("JWT secret key cannot be empty")
	}

	// Validate JWT secret key strength
	if err := v.validateJWTSecret(auth.JwtSecretKey); err != nil {
		return err
	}

	if auth.MaxLoginAttempts <= 0 {
		return fmt.Errorf("maximum login attempts must be greater than 0")
	}

	return nil
}

// validateJWTSecret validates JWT secret key strength
func (v *ConfigValidator) validateJWTSecret(secret string) error {
	// Minimum length check
	minLength := 16
	if isProduction() {
		minLength = 32
	}

	if len(secret) < minLength {
		return fmt.Errorf("JWT secret key length must be at least %d characters (current %d characters)", minLength, len(secret))
	}

	// Additional checks for production environment
	if isProduction() {
		// Check if using default or example keys
		weakSecrets := []string{
			"your-super-secret-jwt-key-here",
			"dev-jwt-secret-key-change-in-production",
			"secret",
			"password",
			"123456",
		}

		for _, weak := range weakSecrets {
			if strings.Contains(strings.ToLower(secret), strings.ToLower(weak)) {
				return fmt.Errorf("production environment cannot use default or weak keys")
			}
		}

		// Check key complexity
		if !isComplexSecret(secret) {
			return fmt.Errorf("production environment JWT secret key complexity is insufficient, should contain uppercase, lowercase, numbers and special characters")
		}
	}

	return nil
}

// validateSecurity validates security configuration
func (v *ConfigValidator) validateSecurity() error {
	// Since Security field may not be defined in proto, skip this validation for now
	// Can perform security configuration checks via environment variables
	if isProduction() {
		// Check TLS configuration
		if os.Getenv("TLS_ENABLED") == "true" {
			if os.Getenv("TLS_CERT_FILE") == "" || os.Getenv("TLS_KEY_FILE") == "" {
				return fmt.Errorf("when TLS is enabled, TLS_CERT_FILE and TLS_KEY_FILE environment variables must be set")
			}

			// Check if certificate files exist
			certFile := os.Getenv("TLS_CERT_FILE")
			keyFile := os.Getenv("TLS_KEY_FILE")

			if _, err := os.Stat(certFile); os.IsNotExist(err) {
				return fmt.Errorf("TLS certificate file does not exist: %s", certFile)
			}

			if _, err := os.Stat(keyFile); os.IsNotExist(err) {
				return fmt.Errorf("TLS key file does not exist: %s", keyFile)
			}
		}
	}

	return nil
}

// isProduction checks if running in production environment
func isProduction() bool {
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	return env == "production" || env == "prod"
}

// isComplexSecret checks secret complexity
func isComplexSecret(secret string) bool {
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range secret {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	// Must contain at least 3 types of characters
	count := 0
	if hasUpper {
		count++
	}
	if hasLower {
		count++
	}
	if hasDigit {
		count++
	}
	if hasSpecial {
		count++
	}

	return count >= 3
}
