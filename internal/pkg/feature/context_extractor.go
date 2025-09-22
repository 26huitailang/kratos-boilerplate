package feature

import (
	"context"
	"strings"
)

// EvaluationContextExtractor 评估上下文提取器接口
type EvaluationContextExtractor interface {
	// Extract 从上下文中提取评估信息
	Extract(ctx context.Context) *EvaluationContext
}

// ConfigurableExtractor 可配置的上下文提取器
type ConfigurableExtractor struct {
	// 上下文键名配置
	UserIDKey     string
	UserTypeKey   string
	EnvironmentKey string
	VersionKey    string
	TenantIDKey   string
	RegionKey     string
	
	// 默认值配置
	DefaultValues map[string]string
	
	// 属性提取配置
	AttributeKeys map[string]string // 自定义属性键映射
}

// ExtractorConfig 提取器配置
type ExtractorConfig struct {
	// 上下文键名
	Keys map[string]string `yaml:"keys" json:"keys"`
	
	// 默认值
	Defaults map[string]string `yaml:"defaults" json:"defaults"`
	
	// 属性映射
	Attributes map[string]string `yaml:"attributes" json:"attributes"`
}

// NewConfigurableExtractor 创建可配置的上下文提取器
func NewConfigurableExtractor(config *ExtractorConfig) *ConfigurableExtractor {
	extractor := &ConfigurableExtractor{
		UserIDKey:     "user_id",
		UserTypeKey:   "user_type", 
		EnvironmentKey: "environment",
		VersionKey:    "version",
		TenantIDKey:   "tenant_id",
		RegionKey:     "region",
		DefaultValues: map[string]string{
			"environment": "production",
			"version":     "1.0.0",
		},
		AttributeKeys: make(map[string]string),
	}
	
	if config != nil {
		// 设置自定义键名
		if keys := config.Keys; keys != nil {
			if userID, ok := keys["user_id"]; ok {
				extractor.UserIDKey = userID
			}
			if userType, ok := keys["user_type"]; ok {
				extractor.UserTypeKey = userType
			}
			if environment, ok := keys["environment"]; ok {
				extractor.EnvironmentKey = environment
			}
			if version, ok := keys["version"]; ok {
				extractor.VersionKey = version
			}
			if tenantID, ok := keys["tenant_id"]; ok {
				extractor.TenantIDKey = tenantID
			}
			if region, ok := keys["region"]; ok {
				extractor.RegionKey = region
			}
		}
		
		// 设置默认值
		if defaults := config.Defaults; defaults != nil {
			for k, v := range defaults {
				extractor.DefaultValues[k] = v
			}
		}
		
		// 设置属性映射
		if attributes := config.Attributes; attributes != nil {
			extractor.AttributeKeys = attributes
		}
	}
	
	return extractor
}

// Extract 从上下文中提取评估信息
func (ce *ConfigurableExtractor) Extract(ctx context.Context) *EvaluationContext {
	evalCtx := &EvaluationContext{
		Environment: ce.getDefault("environment"),
		Version:     ce.getDefault("version"),
		Attributes:  make(map[string]string),
	}
	
	// 提取用户ID
	if userID := ce.extractStringValue(ctx, ce.UserIDKey); userID != "" {
		evalCtx.UserID = userID
	}
	
	// 提取用户类型
	if userType := ce.extractStringValue(ctx, ce.UserTypeKey); userType != "" {
		evalCtx.UserType = userType
	}
	
	// 提取环境信息
	if environment := ce.extractStringValue(ctx, ce.EnvironmentKey); environment != "" {
		evalCtx.Environment = environment
	}
	
	// 提取版本信息
	if version := ce.extractStringValue(ctx, ce.VersionKey); version != "" {
		evalCtx.Version = version
	}
	
	// 提取租户ID
	if tenantID := ce.extractStringValue(ctx, ce.TenantIDKey); tenantID != "" {
		evalCtx.Attributes["tenant_id"] = tenantID
	}
	
	// 提取区域信息
	if region := ce.extractStringValue(ctx, ce.RegionKey); region != "" {
		evalCtx.Attributes["region"] = region
	}
	
	// 提取自定义属性
	for attrName, contextKey := range ce.AttributeKeys {
		if value := ce.extractStringValue(ctx, contextKey); value != "" {
			evalCtx.Attributes[attrName] = value
		}
	}
	
	return evalCtx
}

// extractStringValue 从上下文中提取字符串值
func (ce *ConfigurableExtractor) extractStringValue(ctx context.Context, key string) string {
	if value := ctx.Value(key); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// getDefault 获取默认值
func (ce *ConfigurableExtractor) getDefault(key string) string {
	if value, ok := ce.DefaultValues[key]; ok {
		return value
	}
	return ""
}

// SetDefault 设置默认值
func (ce *ConfigurableExtractor) SetDefault(key, value string) {
	if ce.DefaultValues == nil {
		ce.DefaultValues = make(map[string]string)
	}
	ce.DefaultValues[key] = value
}

// SetAttributeKey 设置属性键映射
func (ce *ConfigurableExtractor) SetAttributeKey(attrName, contextKey string) {
	if ce.AttributeKeys == nil {
		ce.AttributeKeys = make(map[string]string)
	}
	ce.AttributeKeys[attrName] = contextKey
}

// SimpleExtractor 简单的上下文提取器
type SimpleExtractor struct {
	defaultContext *EvaluationContext
}

// NewSimpleExtractor 创建简单的上下文提取器
func NewSimpleExtractor() *SimpleExtractor {
	return &SimpleExtractor{
		defaultContext: &EvaluationContext{
			Environment: "production",
			Version:     "1.0.0",
			Attributes:  make(map[string]string),
		},
	}
}

// Extract 简单提取逻辑
func (se *SimpleExtractor) Extract(ctx context.Context) *EvaluationContext {
	evalCtx := &EvaluationContext{
		Environment: se.defaultContext.Environment,
		Version:     se.defaultContext.Version,
		Attributes:  make(map[string]string),
	}
	
	// 基本提取逻辑
	if userID, ok := ctx.Value("user_id").(string); ok {
		evalCtx.UserID = userID
	}
	
	if userType, ok := ctx.Value("user_type").(string); ok {
		evalCtx.UserType = userType
	}
	
	if environment, ok := ctx.Value("environment").(string); ok {
		evalCtx.Environment = environment
	}
	
	return evalCtx
}

// SetDefaultContext 设置默认上下文
func (se *SimpleExtractor) SetDefaultContext(ctx *EvaluationContext) {
	se.defaultContext = ctx
}

// HTTPHeaderExtractor HTTP请求头提取器
type HTTPHeaderExtractor struct {
	*ConfigurableExtractor
	headerMappings map[string]string // HTTP头到上下文键的映射
}

// NewHTTPHeaderExtractor 创建HTTP头提取器
func NewHTTPHeaderExtractor(config *ExtractorConfig) *HTTPHeaderExtractor {
	return &HTTPHeaderExtractor{
		ConfigurableExtractor: NewConfigurableExtractor(config),
		headerMappings: map[string]string{
			"X-User-ID":     "user_id",
			"X-User-Type":   "user_type",
			"X-Environment": "environment",
			"X-Version":     "version",
			"X-Tenant-ID":   "tenant_id",
			"X-Region":      "region",
		},
	}
}

// SetHeaderMapping 设置HTTP头映射
func (he *HTTPHeaderExtractor) SetHeaderMapping(header, contextKey string) {
	if he.headerMappings == nil {
		he.headerMappings = make(map[string]string)
	}
	he.headerMappings[header] = contextKey
}

// Extract 从HTTP头和上下文中提取信息
func (he *HTTPHeaderExtractor) Extract(ctx context.Context) *EvaluationContext {
	evalCtx := he.ConfigurableExtractor.Extract(ctx)
	
	// 从HTTP头中提取信息（如果存在）
	if headers := he.extractHeaders(ctx); headers != nil {
		for header, contextKey := range he.headerMappings {
			if value := headers[header]; value != "" {
				switch contextKey {
				case he.UserIDKey:
					if evalCtx.UserID == "" {
						evalCtx.UserID = value
					}
				case he.UserTypeKey:
					if evalCtx.UserType == "" {
						evalCtx.UserType = value
					}
				case he.EnvironmentKey:
					if evalCtx.Environment == he.getDefault("environment") {
						evalCtx.Environment = value
					}
				case he.VersionKey:
					if evalCtx.Version == he.getDefault("version") {
						evalCtx.Version = value
					}
				default:
					// 作为自定义属性存储
					evalCtx.Attributes[contextKey] = value
				}
			}
		}
	}
	
	return evalCtx
}

// extractHeaders 从上下文中提取HTTP头信息
func (he *HTTPHeaderExtractor) extractHeaders(ctx context.Context) map[string]string {
	// 这里需要根据具体的HTTP框架实现
	// 示例：从上下文中获取HTTP头
	if headers, ok := ctx.Value("http_headers").(map[string]string); ok {
		result := make(map[string]string)
		for k, v := range headers {
			// 标准化头名称
			result[strings.Title(k)] = v
		}
		return result
	}
	return nil
}

// ChainExtractor 链式提取器
type ChainExtractor struct {
	extractors []EvaluationContextExtractor
}

// NewChainExtractor 创建链式提取器
func NewChainExtractor(extractors ...EvaluationContextExtractor) *ChainExtractor {
	return &ChainExtractor{
		extractors: extractors,
	}
}

// AddExtractor 添加提取器
func (ce *ChainExtractor) AddExtractor(extractor EvaluationContextExtractor) {
	ce.extractors = append(ce.extractors, extractor)
}

// Extract 按顺序执行提取器，后面的提取器可以覆盖前面的结果
func (ce *ChainExtractor) Extract(ctx context.Context) *EvaluationContext {
	var result *EvaluationContext
	
	for _, extractor := range ce.extractors {
		extracted := extractor.Extract(ctx)
		if result == nil {
			result = extracted
		} else {
			// 合并结果，后面的覆盖前面的
			ce.mergeContext(result, extracted)
		}
	}
	
	if result == nil {
		return &EvaluationContext{
			Environment: "production",
			Version:     "1.0.0",
			Attributes:  make(map[string]string),
		}
	}
	
	return result
}

// mergeContext 合并上下文
func (ce *ChainExtractor) mergeContext(target, source *EvaluationContext) {
	if source.UserID != "" {
		target.UserID = source.UserID
	}
	if source.UserType != "" {
		target.UserType = source.UserType
	}
	if source.Environment != "" {
		target.Environment = source.Environment
	}
	if source.Version != "" {
		target.Version = source.Version
	}
	
	// 合并属性
	if source.Attributes != nil {
		if target.Attributes == nil {
			target.Attributes = make(map[string]string)
		}
		for k, v := range source.Attributes {
			target.Attributes[k] = v
		}
	}
}

// ExtractEvaluationContext 默认上下文提取函数（向后兼容）
func ExtractEvaluationContext(ctx context.Context) *EvaluationContext {
	extractor := NewSimpleExtractor()
	return extractor.Extract(ctx)
}