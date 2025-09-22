package feature

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// Middleware 功能开关中间件
func Middleware(toggleManager HierarchicalFeatureToggle, logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 从请求上下文中提取评估上下文
			evalCtx := ExtractEvaluationContext(ctx)

			// 检查请求路径相关的功能开关
			if info, ok := transport.FromServerContext(ctx); ok {
				// 检查分层功能集合
				featureSet := mapPathToFeatureSet(info.Operation())
				if featureSet != "" {
					if !toggleManager.IsFeatureEnabled(featureSet) {
						log.NewHelper(logger).Warnf("Feature set %s is disabled for operation %s", featureSet, info.Operation())
						return nil, errors.Forbidden("FEATURE_DISABLED", fmt.Sprintf("Feature set %s is currently disabled", featureSet))
					}

					// 检查具体API是否在功能集合中
					if !toggleManager.HasAPI(featureSet, info.Operation()) {
						log.NewHelper(logger).Warnf("API %s is not available in feature set %s", info.Operation(), featureSet)
						return nil, errors.Forbidden("API_DISABLED", fmt.Sprintf("API %s is currently disabled", info.Operation()))
					}
				}

				// 向后兼容 - 检查原有的单一功能开关
				feature := mapPathToFeature(info.Operation())
				if feature != "" && !toggleManager.IsEnabledWithContext(ctx, feature, evalCtx) {
					log.NewHelper(logger).Warnf("Feature %s is disabled for operation %s", feature, info.Operation())
					return nil, errors.Forbidden("FEATURE_DISABLED", fmt.Sprintf("Feature %s is currently disabled", feature))
				}
			}

			return handler(ctx, req)
		}
	}
}

// ServiceDecorator 服务装饰器，为服务方法添加功能开关检查
type ServiceDecorator struct {
	toggleManager HierarchicalFeatureToggle
	logger        *log.Helper
}

// NewServiceDecorator 创建新的服务装饰器
func NewServiceDecorator(toggleManager HierarchicalFeatureToggle, logger log.Logger) *ServiceDecorator {
	return &ServiceDecorator{
		toggleManager: toggleManager,
		logger:        log.NewHelper(logger),
	}
}

// WithFeatureCheck 为服务方法添加功能开关检查
func (sd *ServiceDecorator) WithFeatureCheck(featurePath string) func(func(context.Context, interface{}) (interface{}, error)) func(context.Context, interface{}) (interface{}, error) {
	return func(next func(context.Context, interface{}) (interface{}, error)) func(context.Context, interface{}) (interface{}, error) {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			evalCtx := ExtractEvaluationContext(ctx)

			// 检查功能集合是否启用
			if IsHierarchicalFeature(featurePath) {
				if !sd.toggleManager.IsFeatureEnabled(featurePath) {
					sd.logger.Warnf("Feature set %s is disabled", featurePath)
					return nil, errors.Forbidden("FEATURE_DISABLED", fmt.Sprintf("Feature set %s is currently disabled", featurePath))
				}
			} else {
				// 向后兼容 - 使用原有的检查方式
				if !sd.toggleManager.IsEnabledWithContext(ctx, FeatureFlag(featurePath), evalCtx) {
					sd.logger.Warnf("Feature %s is disabled", featurePath)
					return nil, errors.Forbidden("FEATURE_DISABLED", fmt.Sprintf("Feature %s is currently disabled", featurePath))
				}
			}

			return next(ctx, req)
		}
	}
}

// ConditionalDecorator 条件装饰器，根据功能开关选择不同的实现
type ConditionalDecorator struct {
	toggleManager HierarchicalFeatureToggle
	logger        *log.Helper
}

// NewConditionalDecorator 创建新的条件装饰器
func NewConditionalDecorator(toggleManager HierarchicalFeatureToggle, logger log.Logger) *ConditionalDecorator {
	return &ConditionalDecorator{
		toggleManager: toggleManager,
		logger:        log.NewHelper(logger),
	}
}

// WithAlternative 根据功能开关选择不同的实现
func (cd *ConditionalDecorator) WithAlternative(
	featurePath string,
	primaryImpl func(context.Context, interface{}) (interface{}, error),
	fallbackImpl func(context.Context, interface{}) (interface{}, error),
) func(context.Context, interface{}) (interface{}, error) {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		evalCtx := ExtractEvaluationContext(ctx)

		var enabled bool
		if IsHierarchicalFeature(featurePath) {
			enabled = cd.toggleManager.IsFeatureEnabled(featurePath)
		} else {
			enabled = cd.toggleManager.IsEnabledWithContext(ctx, FeatureFlag(featurePath), evalCtx)
		}

		if enabled {
			cd.logger.Debugf("Using primary implementation for feature %s", featurePath)
			return primaryImpl(ctx, req)
		} else {
			cd.logger.Debugf("Using fallback implementation for feature %s", featurePath)
			return fallbackImpl(ctx, req)
		}
	}
}

// GradualRolloutDecorator 渐进式发布装饰器
type GradualRolloutDecorator struct {
	toggleManager HierarchicalFeatureToggle
	logger        *log.Helper
}

// NewGradualRolloutDecorator 创建新的渐进式发布装饰器
func NewGradualRolloutDecorator(toggleManager HierarchicalFeatureToggle, logger log.Logger) *GradualRolloutDecorator {
	return &GradualRolloutDecorator{
		toggleManager: toggleManager,
		logger:        log.NewHelper(logger),
	}
}

// WithRollout 渐进式发布功能
func (grd *GradualRolloutDecorator) WithRollout(
	featurePath string,
	newImpl func(context.Context, interface{}) (interface{}, error),
	oldImpl func(context.Context, interface{}) (interface{}, error),
) func(context.Context, interface{}) (interface{}, error) {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		evalCtx := ExtractEvaluationContext(ctx)

		var enabled bool
		if IsHierarchicalFeature(featurePath) {
			enabled = grd.toggleManager.IsFeatureEnabled(featurePath)
		} else {
			enabled = grd.toggleManager.IsEnabledWithContext(ctx, FeatureFlag(featurePath), evalCtx)
		}

		if enabled {
			grd.logger.Debugf("Using new implementation for feature %s (user: %s)", featurePath, evalCtx.UserID)
			return newImpl(ctx, req)
		} else {
			grd.logger.Debugf("Using old implementation for feature %s (user: %s)", featurePath, evalCtx.UserID)
			return oldImpl(ctx, req)
		}
	}
}

// CircuitBreakerDecorator 熔断器装饰器，根据功能开关启用/禁用功能
type CircuitBreakerDecorator struct {
	toggleManager HierarchicalFeatureToggle
	logger        *log.Helper
}

// NewCircuitBreakerDecorator 创建新的熔断器装饰器
func NewCircuitBreakerDecorator(toggleManager HierarchicalFeatureToggle, logger log.Logger) *CircuitBreakerDecorator {
	return &CircuitBreakerDecorator{
		toggleManager: toggleManager,
		logger:        log.NewHelper(logger),
	}
}

// WithCircuitBreaker 为功能添加熔断器
func (cbd *CircuitBreakerDecorator) WithCircuitBreaker(
	featurePath string,
	impl func(context.Context, interface{}) (interface{}, error),
	fallback func(context.Context, interface{}) (interface{}, error),
) func(context.Context, interface{}) (interface{}, error) {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		evalCtx := ExtractEvaluationContext(ctx)

		var enabled bool
		if IsHierarchicalFeature(featurePath) {
			enabled = cbd.toggleManager.IsFeatureEnabled(featurePath)
		} else {
			enabled = cbd.toggleManager.IsEnabledWithContext(ctx, FeatureFlag(featurePath), evalCtx)
		}

		if !enabled {
			cbd.logger.Infof("Feature %s is circuit broken, using fallback", featurePath)
			if fallback != nil {
				return fallback(ctx, req)
			}
			return nil, errors.ServiceUnavailable("FEATURE_UNAVAILABLE", fmt.Sprintf("Feature %s is currently unavailable", featurePath))
		}

		return impl(ctx, req)
	}
}

// mapPathToFeature 将请求路径映射到功能开关
func mapPathToFeature(operation string) FeatureFlag {
	pathFeatureMap := map[string]FeatureFlag{
		"/api/v1/auth/captcha":        FeatureCaptcha,
		"/api/v1/auth/captcha/verify": FeatureCaptcha,
		"/api/v1/auth/register":       FeatureCaptcha, // 注册通常需要验证码
		"/api/v1/auth/login":          FeatureAccountLockout,
		"/q/":                         FeatureSwaggerUI,
	}

	if feature, exists := pathFeatureMap[operation]; exists {
		return feature
	}

	// 根据路径前缀进行匹配
	for prefix, feature := range map[string]FeatureFlag{
		"/q/":      FeatureSwaggerUI,
		"/api/v1/": FeatureAPIVersioning,
		"/metrics": FeatureMetrics,
		"/health":  FeatureHealthCheck,
	} {
		if len(operation) >= len(prefix) && operation[:len(prefix)] == prefix {
			return feature
		}
	}

	return ""
}

// mapPathToFeatureSet 将请求路径映射到分层功能集合
func mapPathToFeatureSet(operation string) string {
	pathFeatureSetMap := map[string]string{
		// 用户相关API
		"POST /api/v1/users":         "user.management",
		"PUT /api/v1/users/{id}":     "user.management",
		"DELETE /api/v1/users/{id}":  "user.management",
		"GET /api/v1/users":          "user.management",
		"GET /api/v1/profile":        "user.profile",
		"PUT /api/v1/profile":        "user.profile",
		"POST /api/v1/auth/password": "user.security",
		"POST /api/v1/auth/totp":     "user.security",

		// 系统相关API
		"GET /api/v1/system/config":      "system.config",
		"PUT /api/v1/system/config":      "system.config",
		"GET /api/v1/monitoring/metrics": "system.monitoring",
		"GET /api/v1/monitoring/alerts":  "system.monitoring",

		// 报表相关API
		"GET /api/v1/reports/basic":    "report.basic",
		"GET /api/v1/reports/advanced": "report.advanced",
		"POST /api/v1/reports/export":  "report.advanced",
	}

	if featureSet, exists := pathFeatureSetMap[operation]; exists {
		return featureSet
	}

	// 根据路径前缀进行匹配
	for prefix, featureSet := range map[string]string{
		"/api/v1/users":      "user.management",
		"/api/v1/profile":    "user.profile",
		"/api/v1/auth":       "user.security",
		"/api/v1/system":     "system.config",
		"/api/v1/monitoring": "system.monitoring",
		"/api/v1/reports":    "report.basic",
	} {
		if len(operation) >= len(prefix) && operation[:len(prefix)] == prefix {
			return featureSet
		}
	}

	return ""
}
