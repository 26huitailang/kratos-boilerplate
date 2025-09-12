package feature

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-kratos/kratos/v2/log"
	"gopkg.in/yaml.v2"
)

// FileRepository 基于文件的功能开关存储实现
type FileRepository struct {
	mu           sync.RWMutex
	configPath   string
	format       string // "yaml" or "json"
	logger       *log.Helper
	watcher      *fsnotify.Watcher
	watchHandler func(map[FeatureFlag]*ToggleConfig)
}

// NewFileRepository 创建新的文件存储实现
func NewFileRepository(configPath, format string, logger log.Logger) *FileRepository {
	return &FileRepository{
		configPath: configPath,
		format:     format,
		logger:     log.NewHelper(logger),
	}
}

// LoadToggles 从文件加载所有功能开关配置
func (fr *FileRepository) LoadToggles(ctx context.Context) (map[FeatureFlag]*ToggleConfig, error) {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	toggles := make(map[FeatureFlag]*ToggleConfig)

	// 检查配置文件是否存在
	if _, err := os.Stat(fr.configPath); os.IsNotExist(err) {
		fr.logger.Infof("Config file %s does not exist, creating with default toggles", fr.configPath)
		return fr.createDefaultToggles(ctx)
	}

	data, err := os.ReadFile(fr.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	switch fr.format {
	case "json":
		err = json.Unmarshal(data, &toggles)
	case "yaml", "yml":
		// Parse YAML with inline capabilities support
		var rawToggles map[string]interface{}
		err = yaml.Unmarshal(data, &rawToggles)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
		}

		// Process each toggle configuration
		for flagName, rawConfig := range rawToggles {
			configMap, ok := rawConfig.(map[interface{}]interface{})
			if !ok {
				continue
			}

			// Convert to ToggleConfig
			config := &ToggleConfig{
				Rules: make(map[string]interface{}),
			}

			// Parse standard fields
			if enabled, ok := configMap["enabled"].(bool); ok {
				config.Enabled = enabled
			}
			if strategy, ok := configMap["strategy"].(string); ok {
				config.Strategy = FeatureStrategy(strategy)
			}
			if desc, ok := configMap["description"].(string); ok {
				config.Description = desc
			}
			if rules, ok := configMap["rules"].(map[interface{}]interface{}); ok {
				for k, v := range rules {
					if keyStr, ok := k.(string); ok {
						config.Rules[keyStr] = v
					}
				}
			}
			if tags, ok := configMap["tags"].([]interface{}); ok {
				for _, tag := range tags {
					if tagStr, ok := tag.(string); ok {
						config.Tags = append(config.Tags, tagStr)
					}
				}
			}

			// Parse time fields
			if createdAt, ok := configMap["created_at"].(string); ok {
				if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
					config.CreatedAt = t
				}
			}
			if updatedAt, ok := configMap["updated_at"].(string); ok {
				if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
					config.UpdatedAt = t
				}
			}

			// Parse capabilities fields
			capabilities := &FeatureCapabilities{}
			hasCapabilities := false

			if permissions, ok := configMap["permissions"].([]interface{}); ok {
				for _, perm := range permissions {
					if permStr, ok := perm.(string); ok {
						capabilities.Permissions = append(capabilities.Permissions, permStr)
						hasCapabilities = true
					}
				}
			}
			if auditLogs, ok := configMap["audit_logs"].([]interface{}); ok {
				for _, log := range auditLogs {
					if logStr, ok := log.(string); ok {
						capabilities.AuditLogs = append(capabilities.AuditLogs, logStr)
						hasCapabilities = true
					}
				}
			}
			if apis, ok := configMap["apis"].([]interface{}); ok {
				for _, api := range apis {
					if apiStr, ok := api.(string); ok {
						capabilities.APIs = append(capabilities.APIs, apiStr)
						hasCapabilities = true
					}
				}
			}
			if routes, ok := configMap["routes"].([]interface{}); ok {
				for _, route := range routes {
					if routeStr, ok := route.(string); ok {
						capabilities.Routes = append(capabilities.Routes, routeStr)
						hasCapabilities = true
					}
				}
			}
			if configs, ok := configMap["configs"].([]interface{}); ok {
				for _, cfg := range configs {
					if cfgStr, ok := cfg.(string); ok {
						capabilities.Configs = append(capabilities.Configs, cfgStr)
						hasCapabilities = true
					}
				}
			}

			if hasCapabilities {
				config.Capabilities = capabilities
			}

			toggles[FeatureFlag(flagName)] = config
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", fr.format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	fr.logger.Infof("Loaded %d feature toggles from %s", len(toggles), fr.configPath)
	return toggles, nil
}

// SaveToggle 保存单个功能开关配置
func (fr *FileRepository) SaveToggle(ctx context.Context, flag FeatureFlag, config *ToggleConfig) error {
	// 加载当前所有配置
	toggles, err := fr.LoadToggles(ctx)
	if err != nil {
		toggles = make(map[FeatureFlag]*ToggleConfig)
	}

	// 更新配置
	toggles[flag] = config

	// 保存所有配置
	return fr.saveAllToggles(toggles)
}

// DeleteToggle 删除功能开关配置
func (fr *FileRepository) DeleteToggle(ctx context.Context, flag FeatureFlag) error {
	// 加载当前所有配置
	toggles, err := fr.LoadToggles(ctx)
	if err != nil {
		return err
	}

	// 删除配置
	delete(toggles, flag)

	// 保存所有配置
	return fr.saveAllToggles(toggles)
}

// Watch 监听配置文件变更
func (fr *FileRepository) Watch(ctx context.Context, callback func(map[FeatureFlag]*ToggleConfig)) error {
	if fr.watcher != nil {
		fr.watcher.Close()
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	fr.watcher = watcher
	fr.watchHandler = callback

	// 添加配置文件到监听列表
	configDir := filepath.Dir(fr.configPath)
	if err := watcher.Add(configDir); err != nil {
		return fmt.Errorf("failed to watch config directory: %w", err)
	}

	go fr.watchLoop(ctx)

	fr.logger.Infof("Started watching config file: %s", fr.configPath)
	return nil
}

// watchLoop 文件监听循环
func (fr *FileRepository) watchLoop(ctx context.Context) {
	defer fr.watcher.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-fr.watcher.Events:
			if !ok {
				return
			}

			// 只处理配置文件的写入事件
			if event.Name == fr.configPath && (event.Op&fsnotify.Write == fsnotify.Write) {
				fr.logger.Infof("Config file changed: %s", event.Name)

				// 延迟一点时间确保文件写入完成
				time.Sleep(100 * time.Millisecond)

				toggles, err := fr.LoadToggles(ctx)
				if err != nil {
					fr.logger.Errorf("Failed to reload toggles after file change: %v", err)
					continue
				}

				if fr.watchHandler != nil {
					fr.watchHandler(toggles)
				}
			}
		case err, ok := <-fr.watcher.Errors:
			if !ok {
				return
			}
			fr.logger.Errorf("File watcher error: %v", err)
		}
	}
}

// saveAllToggles 保存所有功能开关配置到文件
func (fr *FileRepository) saveAllToggles(toggles map[FeatureFlag]*ToggleConfig) error {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	// 确保目录存在
	configDir := filepath.Dir(fr.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	var data []byte
	var err error

	switch fr.format {
	case "json":
		data, err = json.MarshalIndent(toggles, "", "  ")
	case "yaml", "yml":
		data, err = yaml.Marshal(toggles)
	default:
		return fmt.Errorf("unsupported format: %s", fr.format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal toggles: %w", err)
	}

	// 使用临时文件确保原子写入
	tmpPath := fr.configPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, fr.configPath); err != nil {
		os.Remove(tmpPath) // 清理临时文件
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	fr.logger.Infof("Saved %d feature toggles to %s", len(toggles), fr.configPath)
	return nil
}

// createDefaultToggles 创建默认的功能开关配置
func (fr *FileRepository) createDefaultToggles(ctx context.Context) (map[FeatureFlag]*ToggleConfig, error) {
	now := time.Now()
	defaultToggles := map[FeatureFlag]*ToggleConfig{
		// 用户相关功能集合
		FeatureUserManagement: {
			Enabled:  true,
			Strategy: StrategySimple,
			Capabilities: &FeatureCapabilities{
				Permissions: []string{"user.create", "user.update", "user.delete", "user.list"},
				AuditLogs:   []string{"user_created", "user_updated", "user_deleted"},
				APIs:        []string{"POST /api/v1/users", "PUT /api/v1/users/{id}", "DELETE /api/v1/users/{id}", "GET /api/v1/users"},
				Routes:      []string{"/admin/users", "/admin/users/create"},
				Configs:     []string{"user.max_count", "user.password_policy"},
			},
			Rules:       make(map[string]interface{}),
			Description: "用户管理功能集合",
			Tags:        []string{"user", "admin"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeatureUserProfile: {
			Enabled:  true,
			Strategy: StrategySimple,
			Capabilities: &FeatureCapabilities{
				Permissions: []string{"profile.view", "profile.edit"},
				AuditLogs:   []string{"profile_updated"},
				APIs:        []string{"GET /api/v1/profile", "PUT /api/v1/profile"},
				Routes:      []string{"/profile", "/profile/edit"},
				Configs:     []string{"profile.avatar_size"},
			},
			Rules:       make(map[string]interface{}),
			Description: "用户资料功能集合",
			Tags:        []string{"user", "profile"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeatureUserSecurity: {
			Enabled:  true,
			Strategy: StrategySimple,
			Capabilities: &FeatureCapabilities{
				Permissions: []string{"security.password_change", "security.totp_manage"},
				AuditLogs:   []string{"password_changed", "totp_enabled"},
				APIs:        []string{"POST /api/v1/auth/password", "POST /api/v1/auth/totp"},
				Routes:      []string{"/security/password", "/security/totp"},
				Configs:     []string{"security.password_policy", "security.totp_required"},
			},
			Rules:       make(map[string]interface{}),
			Description: "用户安全功能集合",
			Tags:        []string{"user", "security"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},

		// 系统相关功能集合
		FeatureSystemConfig: {
			Enabled:  true,
			Strategy: StrategyEnvironment,
			Rules: map[string]interface{}{
				"environments": []string{"development", "staging", "production"},
			},
			Capabilities: &FeatureCapabilities{
				Permissions: []string{"system.config.view", "system.config.edit"},
				AuditLogs:   []string{"system_config_changed"},
				APIs:        []string{"GET /api/v1/system/config", "PUT /api/v1/system/config"},
				Routes:      []string{"/admin/system/config"},
				Configs:     []string{"system.debug_mode", "system.log_level"},
			},
			Description: "系统配置功能集合",
			Tags:        []string{"system", "admin"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeatureSystemMonitoring: {
			Enabled:  true,
			Strategy: StrategySimple,
			Capabilities: &FeatureCapabilities{
				Permissions: []string{"monitoring.view", "monitoring.alerts"},
				AuditLogs:   []string{"alert_configured"},
				APIs:        []string{"GET /api/v1/monitoring/metrics", "GET /api/v1/monitoring/alerts"},
				Routes:      []string{"/admin/monitoring"},
				Configs:     []string{"monitoring.retention_days"},
			},
			Rules:       make(map[string]interface{}),
			Description: "系统监控功能集合",
			Tags:        []string{"system", "monitoring"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},

		// 报表相关功能集合
		FeatureReportBasic: {
			Enabled:  true,
			Strategy: StrategySimple,
			Capabilities: &FeatureCapabilities{
				Permissions: []string{"report.basic.view"},
				AuditLogs:   []string{"basic_report_generated"},
				APIs:        []string{"GET /api/v1/reports/basic"},
				Routes:      []string{"/reports/basic"},
				Configs:     []string{"report.max_records"},
			},
			Rules:       make(map[string]interface{}),
			Description: "基础报表功能集合",
			Tags:        []string{"report", "basic"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeatureReportAdvanced: {
			Enabled:  true,
			Strategy: StrategyUser,
			Rules: map[string]interface{}{
				"user_types": []string{"admin", "premium"},
			},
			Capabilities: &FeatureCapabilities{
				Permissions: []string{"report.advanced.view", "report.advanced.export"},
				AuditLogs:   []string{"advanced_report_generated", "report_exported"},
				APIs:        []string{"GET /api/v1/reports/advanced", "POST /api/v1/reports/export"},
				Routes:      []string{"/reports/advanced"},
				Configs:     []string{"report.export_formats"},
			},
			Description: "高级报表功能集合",
			Tags:        []string{"report", "premium"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},

		// 向后兼容 - 原有功能开关
		FeatureCaptcha: {
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "验证码功能开关",
			Tags:        []string{"auth", "security"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeatureTOTP: {
			Enabled:     false,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "TOTP双因子认证开关",
			Tags:        []string{"auth", "security", "2fa"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeaturePasswordPolicy: {
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "密码策略检查开关",
			Tags:        []string{"auth", "security"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeatureAccountLockout: {
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "账户锁定功能开关",
			Tags:        []string{"auth", "security"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},

		// API相关功能
		FeatureRateLimit: {
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "API限流功能开关",
			Tags:        []string{"api", "performance"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeatureAPIVersioning: {
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "API版本控制开关",
			Tags:        []string{"api", "compatibility"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeatureSwaggerUI: {
			Enabled:  true,
			Strategy: StrategyEnvironment,
			Rules: map[string]interface{}{
				"environments": []string{"development", "staging"},
			},
			Description: "Swagger UI开关",
			Tags:        []string{"api", "documentation"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},

		// 数据相关功能
		FeatureDataEncryption: {
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "数据加密功能开关",
			Tags:        []string{"data", "security"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeatureAuditLog: {
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "审计日志功能开关",
			Tags:        []string{"data", "audit", "compliance"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeatureSensitiveData: {
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "敏感数据处理开关",
			Tags:        []string{"data", "security", "privacy"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},

		// 插件相关功能
		FeaturePluginSystem: {
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "插件系统开关",
			Tags:        []string{"plugin", "extensibility"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeaturePluginSandbox: {
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "插件沙盒功能开关",
			Tags:        []string{"plugin", "security"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},

		// 运维相关功能
		FeatureMetrics: {
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "指标收集功能开关",
			Tags:        []string{"monitoring", "observability"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeatureTracing: {
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "链路追踪功能开关",
			Tags:        []string{"monitoring", "observability"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		FeatureHealthCheck: {
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: "健康检查功能开关",
			Tags:        []string{"monitoring", "health"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	// 保存默认配置到文件
	if err := fr.saveAllToggles(defaultToggles); err != nil {
		return nil, fmt.Errorf("failed to save default toggles: %w", err)
	}

	return defaultToggles, nil
}
