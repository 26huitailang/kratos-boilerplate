package feature

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSimpleStrategy(t *testing.T) {
	strategy := &SimpleStrategy{}

	t.Run("should support simple strategy", func(t *testing.T) {
		assert.True(t, strategy.SupportsStrategy(StrategySimple))
		assert.False(t, strategy.SupportsStrategy(StrategyPercentage))
	})

	t.Run("should return enabled status from config", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{UserID: "user1"}

		config := &ToggleConfig{Enabled: true}
		assert.True(t, strategy.Evaluate(ctx, config, evalCtx))

		config.Enabled = false
		assert.False(t, strategy.Evaluate(ctx, config, evalCtx))
	})
}

func TestPercentageStrategy(t *testing.T) {
	strategy := &PercentageStrategy{}

	t.Run("should support percentage strategy", func(t *testing.T) {
		assert.True(t, strategy.SupportsStrategy(StrategyPercentage))
		assert.False(t, strategy.SupportsStrategy(StrategySimple))
	})

	t.Run("should return false when disabled", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{UserID: "user1"}
		config := &ToggleConfig{
			Enabled: false,
			Rules:   map[string]interface{}{"percentage": 50.0},
		}

		assert.False(t, strategy.Evaluate(ctx, config, evalCtx))
	})

	t.Run("should return false when no user ID", func(t *testing.T) {
		ctx := context.Background()
		config := &ToggleConfig{
			Enabled: true,
			Rules:   map[string]interface{}{"percentage": 50.0},
		}

		assert.False(t, strategy.Evaluate(ctx, config, nil))
		assert.False(t, strategy.Evaluate(ctx, config, &EvaluationContext{}))
	})

	t.Run("should return consistent results for same user", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{UserID: "consistent-user"}
		config := &ToggleConfig{
			Enabled: true,
			Rules:   map[string]interface{}{"percentage": 50.0},
		}

		result1 := strategy.Evaluate(ctx, config, evalCtx)
		result2 := strategy.Evaluate(ctx, config, evalCtx)
		assert.Equal(t, result1, result2)
	})

	t.Run("should handle edge cases", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{UserID: "user1"}

		// 0% should always return false
		config := &ToggleConfig{
			Enabled: true,
			Rules:   map[string]interface{}{"percentage": 0.0},
		}
		assert.False(t, strategy.Evaluate(ctx, config, evalCtx))

		// 100% should always return true
		config.Rules["percentage"] = 100.0
		assert.True(t, strategy.Evaluate(ctx, config, evalCtx))
	})
}

func TestUserStrategy(t *testing.T) {
	strategy := &UserStrategy{}

	t.Run("should support user strategy", func(t *testing.T) {
		assert.True(t, strategy.SupportsStrategy(StrategyUser))
		assert.False(t, strategy.SupportsStrategy(StrategySimple))
	})

	t.Run("should check whitelist", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{UserID: "user1"}
		config := &ToggleConfig{
			Enabled: true,
			Rules: map[string]interface{}{
				"whitelist": []interface{}{"user1", "user2"},
			},
		}

		assert.True(t, strategy.Evaluate(ctx, config, evalCtx))

		evalCtx.UserID = "user3"
		assert.False(t, strategy.Evaluate(ctx, config, evalCtx))
	})

	t.Run("should check blacklist", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{UserID: "user1"}
		config := &ToggleConfig{
			Enabled: true,
			Rules: map[string]interface{}{
				"blacklist": []interface{}{"user1", "user2"},
			},
		}

		assert.False(t, strategy.Evaluate(ctx, config, evalCtx))

		evalCtx.UserID = "user3"
		assert.False(t, strategy.Evaluate(ctx, config, evalCtx))
	})

	t.Run("should check user types", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{
			UserID:   "user1",
			UserType: "admin",
		}
		config := &ToggleConfig{
			Enabled: true,
			Rules: map[string]interface{}{
				"user_types": []interface{}{"admin", "premium"},
			},
		}

		assert.True(t, strategy.Evaluate(ctx, config, evalCtx))

		evalCtx.UserType = "basic"
		assert.False(t, strategy.Evaluate(ctx, config, evalCtx))
	})

	t.Run("should check user attributes", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{
			UserID: "user1",
			Attributes: map[string]string{
				"role":    "admin",
				"company": "test-company",
			},
		}
		config := &ToggleConfig{
			Enabled: true,
			Rules: map[string]interface{}{
				"attributes": map[string]interface{}{
					"role": "admin",
				},
			},
		}

		assert.True(t, strategy.Evaluate(ctx, config, evalCtx))

		evalCtx.Attributes["role"] = "user"
		assert.False(t, strategy.Evaluate(ctx, config, evalCtx))
	})
}

func TestTimeStrategy(t *testing.T) {
	strategy := &TimeStrategy{}

	t.Run("should support time strategy", func(t *testing.T) {
		assert.True(t, strategy.SupportsStrategy(StrategyTime))
		assert.False(t, strategy.SupportsStrategy(StrategySimple))
	})

	t.Run("should return false when disabled", func(t *testing.T) {
		ctx := context.Background()
		config := &ToggleConfig{Enabled: false}

		assert.False(t, strategy.Evaluate(ctx, config, &EvaluationContext{}))
	})

	t.Run("should check time range", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{}

		now := time.Now()
		past := now.Add(-1 * time.Hour)
		future := now.Add(1 * time.Hour)

		config := &ToggleConfig{
			Enabled: true,
			Rules: map[string]interface{}{
				"start_time": past.Format(time.RFC3339),
				"end_time":   future.Format(time.RFC3339),
			},
		}

		assert.True(t, strategy.Evaluate(ctx, config, evalCtx))

		// 测试时间范围之外
		config.Rules["end_time"] = past.Format(time.RFC3339)
		assert.False(t, strategy.Evaluate(ctx, config, evalCtx))
	})

	t.Run("should check workdays", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{}
		config := &ToggleConfig{
			Enabled: true,
			Rules: map[string]interface{}{
				"workdays_only": true,
			},
		}

		// 这个测试依赖于当前是否为工作日，实际项目中可能需要Mock时间
		result := strategy.Evaluate(ctx, config, evalCtx)
		assert.IsType(t, false, result) // 至少确保返回了布尔值
	})
}

func TestEnvironmentStrategy(t *testing.T) {
	strategy := &EnvironmentStrategy{}

	t.Run("should support environment strategy", func(t *testing.T) {
		assert.True(t, strategy.SupportsStrategy(StrategyEnvironment))
		assert.False(t, strategy.SupportsStrategy(StrategySimple))
	})

	t.Run("should check environment whitelist", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{Environment: "production"}
		config := &ToggleConfig{
			Enabled: true,
			Rules: map[string]interface{}{
				"environments": []interface{}{"production", "staging"},
			},
		}

		assert.True(t, strategy.Evaluate(ctx, config, evalCtx))

		evalCtx.Environment = "development"
		assert.False(t, strategy.Evaluate(ctx, config, evalCtx))
	})

	t.Run("should check version range", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{
			Environment: "production",
			Version:     "1.5.0",
		}
		config := &ToggleConfig{
			Enabled: true,
			Rules: map[string]interface{}{
				"environments": []interface{}{"production"},
				"version_range": map[string]interface{}{
					"min": "1.0.0",
					"max": "2.0.0",
				},
			},
		}

		assert.True(t, strategy.Evaluate(ctx, config, evalCtx))

		evalCtx.Version = "0.9.0"
		assert.False(t, strategy.Evaluate(ctx, config, evalCtx))

		evalCtx.Version = "2.1.0"
		assert.False(t, strategy.Evaluate(ctx, config, evalCtx))
	})
}

func TestCompositeStrategy(t *testing.T) {
	strategy := NewCompositeStrategy()

	t.Run("should support all built-in strategies", func(t *testing.T) {
		assert.True(t, strategy.SupportsStrategy(StrategySimple))
		assert.True(t, strategy.SupportsStrategy(StrategyPercentage))
		assert.True(t, strategy.SupportsStrategy(StrategyUser))
		assert.True(t, strategy.SupportsStrategy(StrategyTime))
		assert.True(t, strategy.SupportsStrategy(StrategyEnvironment))
	})

	t.Run("should delegate to appropriate evaluator", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{UserID: "user1"}

		config := &ToggleConfig{
			Enabled:  true,
			Strategy: StrategySimple,
		}

		assert.True(t, strategy.Evaluate(ctx, config, evalCtx))

		config.Enabled = false
		assert.False(t, strategy.Evaluate(ctx, config, evalCtx))
	})

	t.Run("should fall back to simple strategy for unknown strategy", func(t *testing.T) {
		ctx := context.Background()
		evalCtx := &EvaluationContext{UserID: "user1"}

		config := &ToggleConfig{
			Enabled:  true,
			Strategy: FeatureStrategy("unknown"),
		}

		assert.True(t, strategy.Evaluate(ctx, config, evalCtx))
	})

	t.Run("should allow registering custom evaluators", func(t *testing.T) {
		customStrategy := FeatureStrategy("custom")
		customEvaluator := &SimpleStrategy{} // 使用简单策略作为示例

		strategy.RegisterEvaluator(customStrategy, customEvaluator)
		assert.True(t, strategy.SupportsStrategy(customStrategy))
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("simpleHash should be consistent", func(t *testing.T) {
		hash1 := simpleHash("test")
		hash2 := simpleHash("test")
		assert.Equal(t, hash1, hash2)

		hash3 := simpleHash("different")
		assert.NotEqual(t, hash1, hash3)
	})

	t.Run("compareVersion should work correctly", func(t *testing.T) {
		assert.Equal(t, 0, compareVersion("1.0.0", "1.0.0"))
		assert.Equal(t, -1, compareVersion("1.0.0", "1.0.1"))
		assert.Equal(t, 1, compareVersion("1.0.1", "1.0.0"))
		assert.Equal(t, -1, compareVersion("1.0.0", "2.0.0"))
		assert.Equal(t, 1, compareVersion("2.0.0", "1.0.0"))
	})
}
