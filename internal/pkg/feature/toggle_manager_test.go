package feature

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockFeatureRepository 模拟功能开关存储
type MockFeatureRepository struct {
	mock.Mock
}

func (m *MockFeatureRepository) LoadToggles(ctx context.Context) (map[FeatureFlag]*ToggleConfig, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[FeatureFlag]*ToggleConfig), args.Error(1)
}

func (m *MockFeatureRepository) SaveToggle(ctx context.Context, flag FeatureFlag, config *ToggleConfig) error {
	args := m.Called(ctx, flag, config)
	return args.Error(0)
}

func (m *MockFeatureRepository) DeleteToggle(ctx context.Context, flag FeatureFlag) error {
	args := m.Called(ctx, flag)
	return args.Error(0)
}

func (m *MockFeatureRepository) Watch(ctx context.Context, callback func(map[FeatureFlag]*ToggleConfig)) error {
	args := m.Called(ctx, callback)
	return args.Error(0)
}

// MockStrategyEvaluator 模拟策略评估器
type MockStrategyEvaluator struct {
	mock.Mock
}

func (m *MockStrategyEvaluator) Evaluate(ctx context.Context, config *ToggleConfig, evalCtx *EvaluationContext) bool {
	args := m.Called(ctx, config, evalCtx)
	return args.Bool(0)
}

func (m *MockStrategyEvaluator) SupportsStrategy(strategy FeatureStrategy) bool {
	args := m.Called(strategy)
	return args.Bool(0)
}

func TestToggleManager_Initialize(t *testing.T) {
	mockRepo := &MockFeatureRepository{}
	mockEvaluator := &MockStrategyEvaluator{}
	logger := log.NewStdLogger(os.Stdout)

	manager := NewToggleManager(mockRepo, mockEvaluator, logger)

	t.Run("should initialize successfully", func(t *testing.T) {
		toggles := map[FeatureFlag]*ToggleConfig{
			FeatureCaptcha: {
				Enabled:  true,
				Strategy: StrategySimple,
			},
		}

		mockRepo.On("LoadToggles", mock.Anything).Return(toggles, nil)
		mockRepo.On("Watch", mock.Anything, mock.Anything).Return(nil)

		err := manager.Initialize(context.Background())
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("should handle initialization error", func(t *testing.T) {
		mockRepo.ExpectedCalls = nil // Reset mock
		mockRepo.On("LoadToggles", mock.Anything).Return(nil, assert.AnError)

		err := manager.Initialize(context.Background())
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestToggleManager_IsEnabled(t *testing.T) {
	mockRepo := &MockFeatureRepository{}
	mockEvaluator := &MockStrategyEvaluator{}
	logger := log.NewStdLogger(os.Stdout)

	manager := NewToggleManager(mockRepo, mockEvaluator, logger)

	// 初始化管理器
	toggles := map[FeatureFlag]*ToggleConfig{
		FeatureCaptcha: {
			Enabled:  true,
			Strategy: StrategySimple,
		},
	}
	mockRepo.On("LoadToggles", mock.Anything).Return(toggles, nil)
	mockRepo.On("Watch", mock.Anything, mock.Anything).Return(nil)
	manager.Initialize(context.Background())

	t.Run("should return true for existing enabled feature", func(t *testing.T) {
		mockEvaluator.On("Evaluate", mock.Anything, mock.Anything, mock.Anything).Return(true)

		result := manager.IsEnabled(context.Background(), FeatureCaptcha)
		assert.True(t, result)
		mockEvaluator.AssertExpectations(t)
	})

	t.Run("should return true for non-existing feature (default behavior)", func(t *testing.T) {
		result := manager.IsEnabled(context.Background(), FeatureFlag("non-existing"))
		assert.True(t, result)
	})
}

func TestToggleManager_IsEnabledWithContext(t *testing.T) {
	mockRepo := &MockFeatureRepository{}
	mockEvaluator := &MockStrategyEvaluator{}
	logger := log.NewStdLogger(os.Stdout)

	manager := NewToggleManager(mockRepo, mockEvaluator, logger)

	// 初始化管理器
	toggles := map[FeatureFlag]*ToggleConfig{
		FeatureCaptcha: {
			Enabled:  true,
			Strategy: StrategyUser,
		},
	}
	mockRepo.On("LoadToggles", mock.Anything).Return(toggles, nil)
	mockRepo.On("Watch", mock.Anything, mock.Anything).Return(nil)
	manager.Initialize(context.Background())

	t.Run("should evaluate with provided context", func(t *testing.T) {
		evalCtx := &EvaluationContext{
			UserID:   "user123",
			UserType: "premium",
		}

		mockEvaluator.On("Evaluate", mock.Anything, mock.Anything, evalCtx).Return(true)

		result := manager.IsEnabledWithContext(context.Background(), FeatureCaptcha, evalCtx)
		assert.True(t, result)
		mockEvaluator.AssertExpectations(t)
	})

	t.Run("should use default context when none provided", func(t *testing.T) {
		mockEvaluator.ExpectedCalls = nil // Reset mock
		mockEvaluator.On("Evaluate", mock.Anything, mock.Anything, mock.Anything).Return(false)

		result := manager.IsEnabledWithContext(context.Background(), FeatureCaptcha, nil)
		assert.False(t, result)
		mockEvaluator.AssertExpectations(t)
	})
}

func TestToggleManager_UpdateToggle(t *testing.T) {
	mockRepo := &MockFeatureRepository{}
	mockEvaluator := &MockStrategyEvaluator{}
	logger := log.NewStdLogger(os.Stdout)

	manager := NewToggleManager(mockRepo, mockEvaluator, logger)

	// 初始化管理器
	toggles := map[FeatureFlag]*ToggleConfig{
		FeatureCaptcha: {
			Enabled:  true,
			Strategy: StrategySimple,
		},
	}
	mockRepo.On("LoadToggles", mock.Anything).Return(toggles, nil)
	mockRepo.On("Watch", mock.Anything, mock.Anything).Return(nil)
	manager.Initialize(context.Background())

	t.Run("should update existing toggle", func(t *testing.T) {
		newConfig := &ToggleConfig{
			Enabled:     false,
			Strategy:    StrategyPercentage,
			Rules:       map[string]interface{}{"percentage": 50.0},
			Description: "Updated config",
		}

		mockRepo.On("SaveToggle", mock.Anything, FeatureCaptcha, mock.Anything).Return(nil)

		err := manager.UpdateToggle(FeatureCaptcha, newConfig)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)

		// 验证配置已更新
		updatedConfig, err := manager.GetToggleConfig(FeatureCaptcha)
		require.NoError(t, err)
		assert.False(t, updatedConfig.Enabled)
		assert.Equal(t, StrategyPercentage, updatedConfig.Strategy)
	})

	t.Run("should handle save error", func(t *testing.T) {
		mockRepo.ExpectedCalls = nil // Reset mock
		newConfig := &ToggleConfig{
			Enabled:  false,
			Strategy: StrategySimple,
		}

		mockRepo.On("SaveToggle", mock.Anything, FeatureCaptcha, mock.Anything).Return(assert.AnError)

		err := manager.UpdateToggle(FeatureCaptcha, newConfig)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestToggleManager_EnableDisableFeature(t *testing.T) {
	mockRepo := &MockFeatureRepository{}
	mockEvaluator := &MockStrategyEvaluator{}
	logger := log.NewStdLogger(os.Stdout)

	manager := NewToggleManager(mockRepo, mockEvaluator, logger)

	// 初始化管理器
	toggles := map[FeatureFlag]*ToggleConfig{}
	mockRepo.On("LoadToggles", mock.Anything).Return(toggles, nil)
	mockRepo.On("Watch", mock.Anything, mock.Anything).Return(nil)
	manager.Initialize(context.Background())

	t.Run("should enable new feature", func(t *testing.T) {
		mockRepo.On("SaveToggle", mock.Anything, FeatureCaptcha, mock.Anything).Return(nil)

		err := manager.EnableFeature(FeatureCaptcha)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)

		config, err := manager.GetToggleConfig(FeatureCaptcha)
		require.NoError(t, err)
		assert.True(t, config.Enabled)
	})

	t.Run("should disable existing feature", func(t *testing.T) {
		mockRepo.ExpectedCalls = nil // Reset mock
		mockRepo.On("SaveToggle", mock.Anything, FeatureCaptcha, mock.Anything).Return(nil)

		err := manager.DisableFeature(FeatureCaptcha)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)

		config, err := manager.GetToggleConfig(FeatureCaptcha)
		require.NoError(t, err)
		assert.False(t, config.Enabled)
	})
}

func TestToggleManager_DeleteToggle(t *testing.T) {
	mockRepo := &MockFeatureRepository{}
	mockEvaluator := &MockStrategyEvaluator{}
	logger := log.NewStdLogger(os.Stdout)

	manager := NewToggleManager(mockRepo, mockEvaluator, logger)

	// 初始化管理器
	toggles := map[FeatureFlag]*ToggleConfig{
		FeatureCaptcha: {
			Enabled:  true,
			Strategy: StrategySimple,
		},
	}
	mockRepo.On("LoadToggles", mock.Anything).Return(toggles, nil)
	mockRepo.On("Watch", mock.Anything, mock.Anything).Return(nil)
	manager.Initialize(context.Background())

	t.Run("should delete existing toggle", func(t *testing.T) {
		mockRepo.On("DeleteToggle", mock.Anything, FeatureCaptcha).Return(nil)

		err := manager.DeleteToggle(FeatureCaptcha)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)

		// 验证配置已删除
		_, err = manager.GetToggleConfig(FeatureCaptcha)
		assert.Error(t, err)
	})

	t.Run("should handle delete error", func(t *testing.T) {
		// 重新添加配置用于测试
		manager.UpdateToggle(FeatureCaptcha, &ToggleConfig{Enabled: true})

		mockRepo.ExpectedCalls = nil // Reset mock
		mockRepo.On("DeleteToggle", mock.Anything, FeatureCaptcha).Return(assert.AnError)

		err := manager.DeleteToggle(FeatureCaptcha)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)

		// 验证配置仍然存在（回滚）
		_, err = manager.GetToggleConfig(FeatureCaptcha)
		assert.NoError(t, err)
	})
}

func TestToggleManager_Subscribe(t *testing.T) {
	mockRepo := &MockFeatureRepository{}
	mockEvaluator := &MockStrategyEvaluator{}
	logger := log.NewStdLogger(os.Stdout)

	manager := NewToggleManager(mockRepo, mockEvaluator, logger)

	t.Run("should subscribe to changes", func(t *testing.T) {
		called := false
		callback := func(flag FeatureFlag, oldConfig, newConfig *ToggleConfig) {
			called = true
		}

		err := manager.Subscribe(callback)
		assert.NoError(t, err)

		// 模拟配置变更
		mockRepo.On("SaveToggle", mock.Anything, FeatureCaptcha, mock.Anything).Return(nil)
		manager.UpdateToggle(FeatureCaptcha, &ToggleConfig{Enabled: true})

		// 等待异步回调
		time.Sleep(10 * time.Millisecond)
		assert.True(t, called)
	})

	t.Run("should reject nil callback", func(t *testing.T) {
		err := manager.Subscribe(nil)
		assert.Error(t, err)
	})
}

func TestToggleManager_GetStats(t *testing.T) {
	mockRepo := &MockFeatureRepository{}
	mockEvaluator := &MockStrategyEvaluator{}
	logger := log.NewStdLogger(os.Stdout)

	manager := NewToggleManager(mockRepo, mockEvaluator, logger)

	// 初始化管理器
	toggles := map[FeatureFlag]*ToggleConfig{
		FeatureCaptcha: {
			Enabled:  true,
			Strategy: StrategySimple,
		},
		FeatureTOTP: {
			Enabled:  false,
			Strategy: StrategyUser,
		},
		FeatureRateLimit: {
			Enabled:  true,
			Strategy: StrategySimple,
		},
	}
	mockRepo.On("LoadToggles", mock.Anything).Return(toggles, nil)
	mockRepo.On("Watch", mock.Anything, mock.Anything).Return(nil)
	manager.Initialize(context.Background())

	t.Run("should return correct stats", func(t *testing.T) {
		stats := manager.GetStats()

		assert.Equal(t, 3, stats["total_toggles"])
		assert.Equal(t, 2, stats["enabled_count"])
		assert.Equal(t, 1, stats["disabled_count"])

		strategies := stats["strategies"].(map[FeatureStrategy]int)
		assert.Equal(t, 2, strategies[StrategySimple])
		assert.Equal(t, 1, strategies[StrategyUser])
	})
}
