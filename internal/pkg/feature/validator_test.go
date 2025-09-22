package feature

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationResult_AddIssue(t *testing.T) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationIssue, 0),
		Warnings: make([]ValidationIssue, 0),
		Infos:    make([]ValidationIssue, 0),
	}

	// Add error
	errorIssue := ValidationIssue{
		Level:   ValidationLevelError,
		Code:    "TEST_ERROR",
		Message: "test error",
	}
	result.addIssue(errorIssue)

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, errorIssue, result.Errors[0])

	// Add warning
	warningIssue := ValidationIssue{
		Level:   ValidationLevelWarning,
		Code:    "TEST_WARNING",
		Message: "test warning",
	}
	result.addIssue(warningIssue)

	assert.Len(t, result.Warnings, 1)
	assert.Equal(t, warningIssue, result.Warnings[0])

	// Add info
	infoIssue := ValidationIssue{
		Level:   ValidationLevelInfo,
		Code:    "TEST_INFO",
		Message: "test info",
	}
	result.addIssue(infoIssue)

	assert.Len(t, result.Infos, 1)
	assert.Equal(t, infoIssue, result.Infos[0])
}

func TestValidationResult_MergeResult(t *testing.T) {
	result1 := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationIssue{{Level: ValidationLevelError, Code: "ERROR1"}},
		Warnings: []ValidationIssue{{Level: ValidationLevelWarning, Code: "WARNING1"}},
		Infos:    []ValidationIssue{{Level: ValidationLevelInfo, Code: "INFO1"}},
	}

	result2 := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationIssue{{Level: ValidationLevelError, Code: "ERROR2"}},
		Warnings: []ValidationIssue{{Level: ValidationLevelWarning, Code: "WARNING2"}},
		Infos:    []ValidationIssue{{Level: ValidationLevelInfo, Code: "INFO2"}},
	}

	result1.mergeResult(result2)

	assert.False(t, result1.Valid) // Should be false due to errors
	assert.Len(t, result1.Errors, 2)
	assert.Len(t, result1.Warnings, 2)
	assert.Len(t, result1.Infos, 2)
}

func TestDefaultConfigValidator_ValidateFeatureName(t *testing.T) {
	validator := NewDefaultConfigValidator(nil)

	tests := []struct {
		name        string
		feature     FeatureFlag
		expectError bool
	}{
		{
			name:        "valid hierarchical feature",
			feature:     "user.management",
			expectError: false,
		},
		{
			name:        "valid single feature",
			feature:     "captcha",
			expectError: false,
		},
		{
			name:        "empty feature name",
			feature:     "",
			expectError: true,
		},
		{
			name:        "feature name too long",
			feature:     FeatureFlag(string(make([]byte, 101))), // 101 characters
			expectError: true,
		},
		{
			name:        "invalid pattern - uppercase",
			feature:     "User.Management",
			expectError: true,
		},
		{
			name:        "invalid pattern - special chars",
			feature:     "user.management!",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateFeatureName(tt.feature)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultConfigValidator_ValidateToggle(t *testing.T) {
	validator := NewDefaultConfigValidator(&ValidationSettings{
		RequiredFields:    []string{"enabled", "strategy", "description"},
		AllowedStrategies: []string{"simple", "percentage"},
	})

	tests := []struct {
		name          string
		feature       FeatureFlag
		config        *ToggleConfig
		expectValid   bool
		expectErrors  int
		expectWarnings int
	}{
		{
			name:    "valid config",
			feature: "user.management",
			config: &ToggleConfig{
				Enabled:     true,
				Strategy:    StrategySimple,
				Description: "User management feature",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			expectValid:   true,
			expectErrors:  0,
			expectWarnings: 0,
		},
		{
			name:    "missing required fields",
			feature: "user.management",
			config: &ToggleConfig{
				Enabled:  true,
				Strategy: StrategySimple,
				// Missing description
			},
			expectValid:   false,
			expectErrors:  0,
			expectWarnings: 1, // Description is warning level
		},
		{
			name:    "unsupported strategy",
			feature: "user.management",
			config: &ToggleConfig{
				Enabled:     true,
				Strategy:    StrategyUser, // Not in allowed list
				Description: "Test",
			},
			expectValid:  false,
			expectErrors: 1,
		},
		{
			name:    "invalid timestamps",
			feature: "user.management",
			config: &ToggleConfig{
				Enabled:     true,
				Strategy:    StrategySimple,
				Description: "Test",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now().Add(-time.Hour), // Before created
			},
			expectValid:  false,
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateToggle(tt.feature, tt.config)

			assert.Equal(t, tt.expectValid, result.Valid)
			assert.Len(t, result.Errors, tt.expectErrors)
			assert.Len(t, result.Warnings, tt.expectWarnings)
		})
	}
}

func TestDefaultConfigValidator_ValidateSchema(t *testing.T) {
	validator := NewDefaultConfigValidator(nil)

	tests := []struct {
		name        string
		config      *FeatureConfiguration
		expectValid bool
		expectErrors int
	}{
		{
			name: "valid schema",
			config: &FeatureConfiguration{
				APIVersion: "v1",
				Kind:       "FeatureConfiguration",
				Metadata: ConfigurationMetadata{
					Name: "test-config",
				},
			},
			expectValid:  true,
			expectErrors: 0,
		},
		{
			name: "missing API version",
			config: &FeatureConfiguration{
				Kind: "FeatureConfiguration",
				Metadata: ConfigurationMetadata{
					Name: "test-config",
				},
			},
			expectValid:  false,
			expectErrors: 1,
		},
		{
			name: "invalid kind",
			config: &FeatureConfiguration{
				APIVersion: "v1",
				Kind:       "InvalidKind",
				Metadata: ConfigurationMetadata{
					Name: "test-config",
				},
			},
			expectValid:  false,
			expectErrors: 1,
		},
		{
			name: "unsupported API version",
			config: &FeatureConfiguration{
				APIVersion: "v2",
				Kind:       "FeatureConfiguration",
				Metadata: ConfigurationMetadata{
					Name: "test-config",
				},
			},
			expectValid:   true, // Warning, not error
			expectErrors:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateSchema(tt.config)

			assert.Equal(t, tt.expectValid, result.Valid)
			assert.Len(t, result.Errors, tt.expectErrors)
		})
	}
}

func TestDefaultConfigValidator_ValidateBusinessRules(t *testing.T) {
	validator := NewDefaultConfigValidator(&ValidationSettings{
		MaxFeatures: 2,
	})

	tests := []struct {
		name         string
		config       *FeatureConfiguration
		expectValid  bool
		expectErrors int
	}{
		{
			name: "valid business rules",
			config: &FeatureConfiguration{
				Spec: ConfigurationSpec{
					Features: map[FeatureFlag]*ToggleConfig{
						"feature1": {Enabled: true, Strategy: StrategySimple},
					},
				},
			},
			expectValid:  true,
			expectErrors: 0,
		},
		{
			name: "no features defined",
			config: &FeatureConfiguration{
				Spec: ConfigurationSpec{
					Features: nil,
				},
			},
			expectValid:  false,
			expectErrors: 1,
		},
		{
			name: "too many features",
			config: &FeatureConfiguration{
				Spec: ConfigurationSpec{
					Features: map[FeatureFlag]*ToggleConfig{
						"feature1": {Enabled: true, Strategy: StrategySimple},
						"feature2": {Enabled: true, Strategy: StrategySimple},
						"feature3": {Enabled: true, Strategy: StrategySimple}, // Exceeds limit
					},
				},
			},
			expectValid:  false,
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateBusinessRules(tt.config)

			assert.Equal(t, tt.expectValid, result.Valid)
			assert.Len(t, result.Errors, tt.expectErrors)
		})
	}
}

func TestDefaultConfigValidator_ValidateConfig(t *testing.T) {
	validator := NewDefaultConfigValidator(nil)

	config := &FeatureConfiguration{
		APIVersion: "v1",
		Kind:       "FeatureConfiguration",
		Metadata: ConfigurationMetadata{
			Name: "test-config",
		},
		Spec: ConfigurationSpec{
			Features: map[FeatureFlag]*ToggleConfig{
				"user.management": {
					Enabled:     true,
					Strategy:    StrategySimple,
					Description: "User management",
				},
			},
		},
	}

	result := validator.ValidateConfig(config)

	assert.True(t, result.Valid)
	assert.NotZero(t, result.ValidatedAt)
}

func TestDefaultConfigValidator_AddCustomValidator(t *testing.T) {
	validator := NewDefaultConfigValidator(nil)

	// Create a custom validator
	customValidator := &testCustomValidator{
		name: "test-validator",
		issues: []ValidationIssue{
			{
				Level:   ValidationLevelWarning,
				Code:    "CUSTOM_WARNING",
				Message: "Custom validation warning",
			},
		},
	}

	validator.AddCustomValidator(customValidator)

	config := &FeatureConfiguration{
		APIVersion: "v1",
		Kind:       "FeatureConfiguration",
		Metadata:   ConfigurationMetadata{Name: "test"},
		Spec: ConfigurationSpec{
			Features: map[FeatureFlag]*ToggleConfig{
				"test": {Enabled: true, Strategy: StrategySimple},
			},
		},
	}

	result := validator.ValidateConfig(config)

	// Should include custom validation issues
	assert.Len(t, result.Warnings, 1)
	assert.Equal(t, "CUSTOM_WARNING", result.Warnings[0].Code)
}

func TestDefaultConfigValidator_RemoveCustomValidator(t *testing.T) {
	validator := NewDefaultConfigValidator(nil)

	customValidator := &testCustomValidator{name: "test-validator"}
	validator.AddCustomValidator(customValidator)

	// Verify it was added
	assert.Len(t, validator.customValidators, 1)

	// Remove it
	validator.RemoveCustomValidator("test-validator")

	// Verify it was removed
	assert.Len(t, validator.customValidators, 0)
}

func TestValidationResult_GenerateSummary(t *testing.T) {
	result := &ValidationResult{
		Valid: true,
		Errors: []ValidationIssue{
			{Feature: "feature1"},
			{Feature: "feature2"},
		},
		Warnings: []ValidationIssue{
			{Feature: "feature1"},
		},
		Infos: []ValidationIssue{
			{Feature: "feature3"},
		},
	}

	config := &FeatureConfiguration{
		Spec: ConfigurationSpec{
			Features: map[FeatureFlag]*ToggleConfig{
				"feature1": {},
				"feature2": {},
				"feature3": {},
			},
		},
	}

	result.generateSummary(config)

	assert.Equal(t, 3, result.Summary.TotalFeatures)
	assert.Equal(t, 2, result.Summary.ErrorCount)
	assert.Equal(t, 1, result.Summary.WarningCount)
	assert.Equal(t, 1, result.Summary.InfoCount)
}

// Test helper: custom validator implementation
type testCustomValidator struct {
	name   string
	issues []ValidationIssue
}

func (tcv *testCustomValidator) Validate(config *FeatureConfiguration) []ValidationIssue {
	return tcv.issues
}

func (tcv *testCustomValidator) GetName() string {
	return tcv.name
}