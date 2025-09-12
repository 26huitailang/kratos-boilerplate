package feature

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SimpleStrategy 简单开关策略评估器
type SimpleStrategy struct{}

func (s *SimpleStrategy) Evaluate(ctx context.Context, config *ToggleConfig, evalCtx *EvaluationContext) bool {
	return config.Enabled
}

func (s *SimpleStrategy) SupportsStrategy(strategy FeatureStrategy) bool {
	return strategy == StrategySimple
}

// PercentageStrategy 百分比策略评估器
type PercentageStrategy struct{}

func (p *PercentageStrategy) Evaluate(ctx context.Context, config *ToggleConfig, evalCtx *EvaluationContext) bool {
	if !config.Enabled {
		return false
	}

	percentage, ok := config.Rules["percentage"].(float64)
	if !ok {
		return false
	}

	if percentage <= 0 {
		return false
	}
	if percentage >= 100 {
		return true
	}

	// 使用用户ID进行一致性哈希
	if evalCtx == nil || evalCtx.UserID == "" {
		return false
	}

	hash := simpleHash(evalCtx.UserID)
	return float64(hash%100) < percentage
}

func (p *PercentageStrategy) SupportsStrategy(strategy FeatureStrategy) bool {
	return strategy == StrategyPercentage
}

// UserStrategy 用户属性策略评估器
type UserStrategy struct{}

func (u *UserStrategy) Evaluate(ctx context.Context, config *ToggleConfig, evalCtx *EvaluationContext) bool {
	if !config.Enabled || evalCtx == nil {
		return false
	}

	// 检查用户白名单
	if whitelist, ok := config.Rules["whitelist"].([]interface{}); ok {
		for _, user := range whitelist {
			if userStr, ok := user.(string); ok && userStr == evalCtx.UserID {
				return true
			}
		}
	}

	// 检查用户黑名单
	if blacklist, ok := config.Rules["blacklist"].([]interface{}); ok {
		for _, user := range blacklist {
			if userStr, ok := user.(string); ok && userStr == evalCtx.UserID {
				return false
			}
		}
	}

	// 检查用户类型
	if userTypes, ok := config.Rules["user_types"].([]interface{}); ok {
		for _, userType := range userTypes {
			if typeStr, ok := userType.(string); ok && typeStr == evalCtx.UserType {
				return true
			}
		}
	}

	// 检查用户属性
	if attributes, ok := config.Rules["attributes"].(map[string]interface{}); ok {
		for key, expectedValue := range attributes {
			if actualValue, exists := evalCtx.Attributes[key]; exists {
				if actualValue == fmt.Sprintf("%v", expectedValue) {
					return true
				}
			}
		}
	}

	return false
}

func (u *UserStrategy) SupportsStrategy(strategy FeatureStrategy) bool {
	return strategy == StrategyUser
}

// TimeStrategy 时间策略评估器
type TimeStrategy struct{}

func (t *TimeStrategy) Evaluate(ctx context.Context, config *ToggleConfig, evalCtx *EvaluationContext) bool {
	if !config.Enabled {
		return false
	}

	now := time.Now()

	// 检查开始时间
	if startTimeStr, ok := config.Rules["start_time"].(string); ok {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			if now.Before(startTime) {
				return false
			}
		}
	}

	// 检查结束时间
	if endTimeStr, ok := config.Rules["end_time"].(string); ok {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			if now.After(endTime) {
				return false
			}
		}
	}

	// 检查工作日
	if workdaysOnly, ok := config.Rules["workdays_only"].(bool); ok && workdaysOnly {
		weekday := now.Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			return false
		}
	}

	// 检查工作时间
	if workHours, ok := config.Rules["work_hours"].(map[string]interface{}); ok {
		startHour, _ := workHours["start"].(float64)
		endHour, _ := workHours["end"].(float64)
		currentHour := float64(now.Hour())

		if currentHour < startHour || currentHour >= endHour {
			return false
		}
	}

	return true
}

func (t *TimeStrategy) SupportsStrategy(strategy FeatureStrategy) bool {
	return strategy == StrategyTime
}

// EnvironmentStrategy 环境策略评估器
type EnvironmentStrategy struct{}

func (e *EnvironmentStrategy) Evaluate(ctx context.Context, config *ToggleConfig, evalCtx *EvaluationContext) bool {
	if !config.Enabled || evalCtx == nil {
		return false
	}

	// 检查环境白名单
	if environments, ok := config.Rules["environments"].([]interface{}); ok {
		for _, env := range environments {
			if envStr, ok := env.(string); ok && envStr == evalCtx.Environment {
				return true
			}
		}
		return false
	}

	// 检查版本范围
	if versionRange, ok := config.Rules["version_range"].(map[string]interface{}); ok {
		if evalCtx.Version != "" {
			minVersion, _ := versionRange["min"].(string)
			maxVersion, _ := versionRange["max"].(string)

			if minVersion != "" && compareVersion(evalCtx.Version, minVersion) < 0 {
				return false
			}
			if maxVersion != "" && compareVersion(evalCtx.Version, maxVersion) > 0 {
				return false
			}
		}
	}

	return true
}

func (e *EnvironmentStrategy) SupportsStrategy(strategy FeatureStrategy) bool {
	return strategy == StrategyEnvironment
}

// CompositeStrategy 复合策略评估器
type CompositeStrategy struct {
	evaluators map[FeatureStrategy]StrategyEvaluator
}

func NewCompositeStrategy() *CompositeStrategy {
	return &CompositeStrategy{
		evaluators: map[FeatureStrategy]StrategyEvaluator{
			StrategySimple:      &SimpleStrategy{},
			StrategyPercentage:  &PercentageStrategy{},
			StrategyUser:        &UserStrategy{},
			StrategyTime:        &TimeStrategy{},
			StrategyEnvironment: &EnvironmentStrategy{},
		},
	}
}

func (c *CompositeStrategy) Evaluate(ctx context.Context, config *ToggleConfig, evalCtx *EvaluationContext) bool {
	evaluator, exists := c.evaluators[config.Strategy]
	if !exists {
		// 默认使用简单策略
		evaluator = c.evaluators[StrategySimple]
	}

	return evaluator.Evaluate(ctx, config, evalCtx)
}

func (c *CompositeStrategy) SupportsStrategy(strategy FeatureStrategy) bool {
	_, exists := c.evaluators[strategy]
	return exists
}

func (c *CompositeStrategy) RegisterEvaluator(strategy FeatureStrategy, evaluator StrategyEvaluator) {
	c.evaluators[strategy] = evaluator
}

// 辅助函数
func simpleHash(s string) int {
	hash := 0
	for _, c := range s {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

func compareVersion(version1, version2 string) int {
	v1Parts := strings.Split(version1, ".")
	v2Parts := strings.Split(version2, ".")

	maxLen := len(v1Parts)
	if len(v2Parts) > maxLen {
		maxLen = len(v2Parts)
	}

	for i := 0; i < maxLen; i++ {
		var v1Part, v2Part int

		if i < len(v1Parts) {
			v1Part, _ = strconv.Atoi(v1Parts[i])
		}
		if i < len(v2Parts) {
			v2Part, _ = strconv.Atoi(v2Parts[i])
		}

		if v1Part < v2Part {
			return -1
		} else if v1Part > v2Part {
			return 1
		}
	}

	return 0
}
