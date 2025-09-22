package feature

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeatureError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *FeatureError
		expected string
	}{
		{
			name: "error with feature",
			err: &FeatureError{
				Code:      ErrCodeFeatureNotFound,
				Operation: "check_feature",
				Message:   "feature not found",
				Feature:   FeatureUserManagement,
			},
			expected: "[FEATURE_NOT_FOUND] check_feature (feature: user.management): feature not found",
		},
		{
			name: "error without feature",
			err: &FeatureError{
				Code:      ErrCodeInvalidConfig,
				Operation: "validate_config",
				Message:   "invalid configuration",
			},
			expected: "[INVALID_CONFIG] validate_config: invalid configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestFeatureError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &FeatureError{
		Code:      ErrCodeRepositoryError,
		Operation: "save",
		Message:   "save failed",
		Cause:     cause,
	}

	assert.Equal(t, cause, err.Unwrap())
}

func TestFeatureError_Is(t *testing.T) {
	err1 := &FeatureError{Code: ErrCodeFeatureNotFound}
	err2 := &FeatureError{Code: ErrCodeFeatureNotFound}
	err3 := &FeatureError{Code: ErrCodeInvalidConfig}

	assert.True(t, err1.Is(err2))
	assert.False(t, err1.Is(err3))
	assert.False(t, err1.Is(errors.New("other error")))
}

func TestFeatureError_Methods(t *testing.T) {
	tests := []struct {
		name        string
		err         *FeatureError
		isRetryable bool
		isClient    bool
		isServer    bool
	}{
		{
			name:        "retryable error",
			err:         &FeatureError{Code: ErrCodeNetworkError},
			isRetryable: true,
			isClient:    false,
			isServer:    false,
		},
		{
			name:        "client error",
			err:         &FeatureError{Code: ErrCodeFeatureNotFound},
			isRetryable: false,
			isClient:    true,
			isServer:    false,
		},
		{
			name:        "server error",
			err:         &FeatureError{Code: ErrCodeRepositoryError},
			isRetryable: true,
			isClient:    false,
			isServer:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isRetryable, tt.err.IsRetryable())
			assert.Equal(t, tt.isClient, tt.err.IsClientError())
			assert.Equal(t, tt.isServer, tt.err.IsServerError())
		})
	}
}

func TestFeatureError_WithMethods(t *testing.T) {
	err := NewFeatureError(ErrCodeInvalidConfig, "test", "test message")

	// Test WithFeature
	err = err.WithFeature(FeatureUserManagement)
	assert.Equal(t, FeatureUserManagement, err.Feature)

	// Test WithCause
	cause := errors.New("cause")
	err = err.WithCause(cause)
	assert.Equal(t, cause, err.Cause)

	// Test WithDetail
	err = err.WithDetail("key", "value")
	assert.Equal(t, "value", err.Details["key"])
}

func TestNewFeatureError_Functions(t *testing.T) {
	t.Run("ErrFeatureNotFound", func(t *testing.T) {
		err := ErrFeatureNotFound(FeatureUserManagement)
		assert.Equal(t, ErrCodeFeatureNotFound, err.Code)
		assert.Equal(t, FeatureUserManagement, err.Feature)
		assert.Equal(t, "check_feature", err.Operation)
	})

	t.Run("ErrInvalidConfig", func(t *testing.T) {
		err := ErrInvalidConfig(FeatureUserManagement, "invalid strategy")
		assert.Equal(t, ErrCodeInvalidConfig, err.Code)
		assert.Equal(t, FeatureUserManagement, err.Feature)
		assert.Equal(t, "invalid strategy", err.Message)
	})

	t.Run("ErrStrategyNotSupported", func(t *testing.T) {
		err := ErrStrategyNotSupported("unknown")
		assert.Equal(t, ErrCodeStrategyNotSupported, err.Code)
		assert.Equal(t, "unknown", err.Details["strategy"])
	})
}

func TestErrorCollector(t *testing.T) {
	collector := NewErrorCollector()

	// Initially no errors
	assert.False(t, collector.HasErrors())
	assert.Empty(t, collector.Errors())
	assert.Nil(t, collector.First())
	assert.Empty(t, collector.Error())

	// Add errors
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	collector.Add(err1)
	collector.Add(err2)
	collector.Add(nil) // Should be ignored

	// Check state
	assert.True(t, collector.HasErrors())
	assert.Len(t, collector.Errors(), 2)
	assert.Equal(t, err1, collector.First())
	assert.Contains(t, collector.Error(), "multiple errors (2)")
	assert.Contains(t, collector.Error(), "error 1")
	assert.Contains(t, collector.Error(), "error 2")

	// Test single error
	singleCollector := NewErrorCollector()
	singleCollector.Add(err1)
	assert.Equal(t, "error 1", singleCollector.Error())
}

func TestDefaultErrorHandler(t *testing.T) {
	handler := NewDefaultErrorHandler()

	t.Run("HandleError", func(t *testing.T) {
		// Test with FeatureError
		featureErr := ErrFeatureNotFound(FeatureUserManagement)
		result := handler.HandleError(nil, featureErr)
		assert.Equal(t, featureErr, result)

		// Test with generic error
		genericErr := errors.New("generic error")
		result = handler.HandleError(nil, genericErr)
		assert.IsType(t, &FeatureError{}, result)
		featureResult := result.(*FeatureError)
		assert.Equal(t, ErrCodeDependencyError, featureResult.Code)
		assert.Equal(t, genericErr, featureResult.Cause)
	})

	t.Run("ShouldRetry", func(t *testing.T) {
		// Retryable error
		retryableErr := &FeatureError{Code: ErrCodeNetworkError}
		assert.True(t, handler.ShouldRetry(retryableErr))

		// Non-retryable error
		nonRetryableErr := &FeatureError{Code: ErrCodeFeatureNotFound}
		assert.False(t, handler.ShouldRetry(nonRetryableErr))

		// Generic error
		genericErr := errors.New("generic")
		assert.False(t, handler.ShouldRetry(genericErr))
	})

	t.Run("GetRetryDelay", func(t *testing.T) {
		delay1 := handler.GetRetryDelay(0)
		delay2 := handler.GetRetryDelay(1)
		delay3 := handler.GetRetryDelay(10) // Should be capped

		assert.True(t, delay1 < delay2)
		assert.Equal(t, handler.GetRetryDelay(10), handler.GetRetryDelay(11)) // Should be same (capped)
	})
}