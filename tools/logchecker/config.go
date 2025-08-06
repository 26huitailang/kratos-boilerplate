package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config 配置结构
type Config struct {
	// 是否跳过测试文件
	SkipTestFiles bool `json:"skip_test_files"`
	
	// 忽略的文件模式列表
	IgnoreFiles []string `json:"ignore_files"`
	
	// 忽略的目录列表
	IgnoreDirs []string `json:"ignore_dirs"`
	
	// 白名单：允许使用格式化日志的文件
	WhitelistFiles []string `json:"whitelist_files"`
	
	// 白名单：允许使用格式化日志的方法
	WhitelistMethods []string `json:"whitelist_methods"`
	
	// 自定义敏感关键词
	SensitiveKeywords []string `json:"sensitive_keywords"`
	
	// 最小消息长度阈值（超过此长度建议使用结构化日志）
	MinMessageLength int `json:"min_message_length"`
	
	// 严重程度配置
	SeverityConfig SeverityConfig `json:"severity_config"`
}

// SeverityConfig 严重程度配置
type SeverityConfig struct {
	ErrorMethods []string `json:"error_methods"`
	WarnMethods  []string `json:"warn_methods"`
	InfoMethods  []string `json:"info_methods"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) *Config {
	// 默认配置
	config := &Config{
		SkipTestFiles: true,
		IgnoreFiles: []string{
			"*.pb.go",
			"*.gen.go",
			"vendor/*",
			".git/*",
		},
		IgnoreDirs: []string{
			"vendor",
			".git",
			"node_modules",
			"third_party",
		},
		WhitelistFiles: []string{},
		WhitelistMethods: []string{
			"Printf", // 某些场景下Printf可能是合理的
		},
		SensitiveKeywords: []string{
			"password", "token", "secret", "key", "auth",
			"email", "phone", "mobile", "card", "id",
			"user", "account", "login", "credential",
			"session", "cookie", "jwt", "oauth",
		},
		MinMessageLength: 50,
		SeverityConfig: SeverityConfig{
			ErrorMethods: []string{"Errorf", "Error"},
			WarnMethods:  []string{"Warnf", "Warn", "Warningf", "Warning"},
			InfoMethods:  []string{"Infof", "Info", "Debugf", "Debug"},
		},
	}
	
	// 如果指定了配置文件，尝试加载
	if configPath != "" {
		if err := config.loadFromFile(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load config file %s: %v\n", configPath, err)
			fmt.Fprintf(os.Stderr, "Using default configuration\n")
		}
	}
	
	return config
}

// loadFromFile 从文件加载配置
func (c *Config) loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	
	return nil
}

// SaveDefaultConfig 保存默认配置到文件
func SaveDefaultConfig(path string) error {
	config := LoadConfig("")
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// IsWhitelistedFile 检查文件是否在白名单中
func (c *Config) IsWhitelistedFile(filename string) bool {
	for _, pattern := range c.WhitelistFiles {
		if matched, _ := filepath.Match(pattern, filepath.Base(filename)); matched {
			return true
		}
		if strings.Contains(filename, pattern) {
			return true
		}
	}
	return false
}

// IsWhitelistedMethod 检查方法是否在白名单中
func (c *Config) IsWhitelistedMethod(methodName string) bool {
	for _, method := range c.WhitelistMethods {
		if methodName == method {
			return true
		}
	}
	return false
}

// GetSeverity 根据方法名获取严重程度
func (c *Config) GetSeverity(methodName string) string {
	for _, method := range c.SeverityConfig.ErrorMethods {
		if methodName == method {
			return "high"
		}
	}
	
	for _, method := range c.SeverityConfig.WarnMethods {
		if methodName == method {
			return "medium"
		}
	}
	
	for _, method := range c.SeverityConfig.InfoMethods {
		if methodName == method {
			return "low"
		}
	}
	
	return "low" // 默认低严重程度
}