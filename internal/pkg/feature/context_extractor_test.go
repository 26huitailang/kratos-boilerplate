package feature

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigurableExtractor_Extract(t *testing.T) {
	config := &ExtractorConfig{
		Keys: map[string]string{
			"user_id":   "custom_user_id",
			"user_type": "custom_user_type",
		},
		Defaults: map[string]string{
			"environment": "test",
			"version":     "2.0.0",
		},
		Attributes: map[string]string{
			"tenant": "tenant_id",
			"region": "region_code",
		},
	}

	extractor := NewConfigurableExtractor(config)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "custom_user_id", "user123")
	ctx = context.WithValue(ctx, "custom_user_type", "admin")
	ctx = context.WithValue(ctx, "tenant_id", "tenant456")
	ctx = context.WithValue(ctx, "region_code", "us-west")

	result := extractor.Extract(ctx)

	assert.Equal(t, "user123", result.UserID)
	assert.Equal(t, "admin", result.UserType)
	assert.Equal(t, "test", result.Environment)
	assert.Equal(t, "2.0.0", result.Version)
	assert.Equal(t, "tenant456", result.Attributes["tenant"])
	assert.Equal(t, "us-west", result.Attributes["region"])
}

func TestConfigurableExtractor_DefaultConfig(t *testing.T) {
	extractor := NewConfigurableExtractor(nil)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "user_id", "user123")
	ctx = context.WithValue(ctx, "environment", "staging")

	result := extractor.Extract(ctx)

	assert.Equal(t, "user123", result.UserID)
	assert.Equal(t, "staging", result.Environment)
	assert.Equal(t, "1.0.0", result.Version) // Default value
}

func TestConfigurableExtractor_SetMethods(t *testing.T) {
	extractor := NewConfigurableExtractor(nil)

	extractor.SetDefault("test_key", "test_value")
	assert.Equal(t, "test_value", extractor.DefaultValues["test_key"])

	extractor.SetAttributeKey("custom_attr", "custom_key")
	assert.Equal(t, "custom_key", extractor.AttributeKeys["custom_attr"])

	// Test extraction with custom attribute
	ctx := context.WithValue(context.Background(), "custom_key", "custom_value")
	result := extractor.Extract(ctx)
	assert.Equal(t, "custom_value", result.Attributes["custom_attr"])
}

func TestSimpleExtractor_Extract(t *testing.T) {
	extractor := NewSimpleExtractor()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "user_id", "user123")
	ctx = context.WithValue(ctx, "user_type", "admin")
	ctx = context.WithValue(ctx, "environment", "staging")

	result := extractor.Extract(ctx)

	assert.Equal(t, "user123", result.UserID)
	assert.Equal(t, "admin", result.UserType)
	assert.Equal(t, "staging", result.Environment)
	assert.Equal(t, "1.0.0", result.Version) // Default
}

func TestSimpleExtractor_SetDefaultContext(t *testing.T) {
	extractor := NewSimpleExtractor()

	customDefault := &EvaluationContext{
		Environment: "custom",
		Version:     "3.0.0",
		Attributes:  make(map[string]string),
	}

	extractor.SetDefaultContext(customDefault)

	ctx := context.Background()
	result := extractor.Extract(ctx)

	assert.Equal(t, "custom", result.Environment)
	assert.Equal(t, "3.0.0", result.Version)
}

func TestHTTPHeaderExtractor_Extract(t *testing.T) {
	config := &ExtractorConfig{
		Defaults: map[string]string{
			"environment": "test",
		},
	}

	extractor := NewHTTPHeaderExtractor(config)

	// Mock HTTP headers in context
	headers := map[string]string{
		"X-User-ID":     "header_user",
		"X-Environment": "header_env",
		"X-Tenant-ID":   "header_tenant",
	}

	ctx := context.WithValue(context.Background(), "http_headers", headers)

	result := extractor.Extract(ctx)

	assert.Equal(t, "header_user", result.UserID)
	assert.Equal(t, "header_env", result.Environment)
	assert.Equal(t, "header_tenant", result.Attributes["tenant_id"])
}

func TestHTTPHeaderExtractor_SetHeaderMapping(t *testing.T) {
	extractor := NewHTTPHeaderExtractor(nil)

	extractor.SetHeaderMapping("X-Custom-Header", "custom_key")
	assert.Equal(t, "custom_key", extractor.headerMappings["X-Custom-Header"])
}

func TestChainExtractor_Extract(t *testing.T) {
	// Create multiple extractors
	extractor1 := NewSimpleExtractor()
	extractor2 := NewConfigurableExtractor(&ExtractorConfig{
		Defaults: map[string]string{
			"version": "2.0.0",
		},
		Attributes: map[string]string{
			"region": "region_code",
		},
	})

	chain := NewChainExtractor(extractor1, extractor2)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "user_id", "user123")
	ctx = context.WithValue(ctx, "region_code", "us-east")

	result := chain.Extract(ctx)

	// Should have user_id from first extractor and version from second
	assert.Equal(t, "user123", result.UserID)
	assert.Equal(t, "2.0.0", result.Version) // Overridden by second extractor
	assert.Equal(t, "us-east", result.Attributes["region"])
}

func TestChainExtractor_AddExtractor(t *testing.T) {
	chain := NewChainExtractor()
	assert.Empty(t, chain.extractors)

	extractor := NewSimpleExtractor()
	chain.AddExtractor(extractor)
	assert.Len(t, chain.extractors, 1)
}

func TestChainExtractor_EmptyChain(t *testing.T) {
	chain := NewChainExtractor()
	result := chain.Extract(context.Background())

	// Should return default context
	assert.Equal(t, "production", result.Environment)
	assert.Equal(t, "1.0.0", result.Version)
	assert.NotNil(t, result.Attributes)
}

func TestExtractEvaluationContext_BackwardCompatibility(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "user_id", "test_user")

	result := ExtractEvaluationContext(ctx)

	assert.Equal(t, "test_user", result.UserID)
	assert.Equal(t, "production", result.Environment)
	assert.Equal(t, "1.0.0", result.Version)
}

func TestConfigurableExtractor_MissingContext(t *testing.T) {
	extractor := NewConfigurableExtractor(nil)
	
	// Empty context
	ctx := context.Background()
	result := extractor.Extract(ctx)

	// Should use defaults
	assert.Equal(t, "", result.UserID)
	assert.Equal(t, "", result.UserType)
	assert.Equal(t, "production", result.Environment)
	assert.Equal(t, "1.0.0", result.Version)
}

func TestConfigurableExtractor_NonStringValues(t *testing.T) {
	extractor := NewConfigurableExtractor(nil)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "user_id", 123) // Non-string value
	ctx = context.WithValue(ctx, "user_type", "admin")

	result := extractor.Extract(ctx)

	// Non-string values should be ignored
	assert.Equal(t, "", result.UserID)
	assert.Equal(t, "admin", result.UserType)
}