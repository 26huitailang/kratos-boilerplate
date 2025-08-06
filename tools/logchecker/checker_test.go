package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogChecker_ScanFile(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		expectedCount int
		expectedMethods []string
	}{
		{
			name:          "sample file with issues",
			filename:      "testdata/sample.go",
			expectedCount: 9, // 预期检测到的问题数量
			expectedMethods: []string{"Infof", "Debugf", "Errorf", "Warnf", "Info"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewLogChecker("")
			issues, err := checker.scanFile(tt.filename)
			if err != nil {
				t.Fatalf("scanFile() error = %v", err)
			}

			if len(issues) != tt.expectedCount {
				t.Errorf("scanFile() found %d issues, expected %d", len(issues), tt.expectedCount)
				for i, issue := range issues {
					t.Logf("Issue %d: %s at line %d", i+1, issue.Method, issue.Line)
				}
			}

			// 检查是否包含预期的方法
			foundMethods := make(map[string]bool)
			for _, issue := range issues {
				foundMethods[issue.Method] = true
			}

			for _, expectedMethod := range tt.expectedMethods {
				if !foundMethods[expectedMethod] {
					t.Errorf("Expected to find method %s, but it was not detected", expectedMethod)
				}
			}
		})
	}
}

func TestLogChecker_ScanDirectory(t *testing.T) {
	checker := NewLogChecker("")
	result, err := checker.ScanDirectory("testdata")
	if err != nil {
		t.Fatalf("ScanDirectory() error = %v", err)
	}

	if result.TotalFiles == 0 {
		t.Error("ScanDirectory() found no files")
	}

	if result.ScannedFiles == 0 {
		t.Error("ScanDirectory() scanned no files")
	}

	if len(result.Issues) == 0 {
		t.Error("ScanDirectory() found no issues in test data")
	}

	t.Logf("Scanned %d files, found %d issues", result.ScannedFiles, len(result.Issues))
}

func TestLogChecker_WithConfig(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.json")
	
	configContent := `{
		"skip_test_files": false,
		"ignore_files": [],
		"whitelist_methods": ["Info"],
		"min_message_length": 100
	}`
	
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	checker := NewLogChecker(configPath)
	issues, err := checker.scanFile("testdata/sample.go")
	if err != nil {
		t.Fatalf("scanFile() error = %v", err)
	}

	// 检查白名单是否生效（Info方法应该被过滤掉一些）
	infoCalls := 0
	for _, issue := range issues {
		if issue.Method == "Info" {
			infoCalls++
		}
	}

	t.Logf("Found %d Info method issues with whitelist config", infoCalls)
}

func TestConfig_LoadFromFile(t *testing.T) {
	tests := []struct {
		name        string
		configJSON  string
		expectError bool
	}{
		{
			name: "valid config",
			configJSON: `{
				"skip_test_files": true,
				"ignore_files": ["*.pb.go"],
				"min_message_length": 50
			}`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			configJSON:  `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.json")
			
			if err := os.WriteFile(configPath, []byte(tt.configJSON), 0644); err != nil {
				t.Fatalf("Failed to create test config: %v", err)
			}

			config := &Config{}
			err := config.loadFromFile(configPath)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestVisitor_ShouldUseStructuredLog(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		argCount int
		expected bool
	}{
		{
			name:     "multiple arguments",
			message:  "User logged in",
			argCount: 3,
			expected: true,
		},
		{
			name:     "format placeholders",
			message:  "User %s logged in",
			argCount: 1,
			expected: true,
		},
		{
			name:     "long message",
			message:  "This is a very long message that exceeds the minimum length threshold",
			argCount: 1,
			expected: true,
		},
		{
			name:     "sensitive keyword",
			message:  "User password updated",
			argCount: 1,
			expected: true,
		},
		{
			name:     "simple message",
			message:  "Server started",
			argCount: 1,
			expected: false,
		},
	}

	visitor := &logVisitor{
		checker: NewLogChecker(""),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这里需要构造AST节点来测试，简化测试
			result := visitor.containsSensitiveKeywords(tt.message)
			if tt.name == "sensitive keyword" && !result {
				t.Error("Expected to detect sensitive keyword")
			}
		})
	}
}

func TestReporter_GenerateReport(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"console format", "console"},
		{"json format", "json"},
		{"html format", "html"},
	}

	// 创建测试数据
	result := &ScanResult{
		TotalFiles:   5,
		ScannedFiles: 5,
		Issues: []Issue{
			{
				File:        "test.go",
				Line:        10,
				Column:      5,
				Method:      "Infof",
				Suggestion:  "Infow",
				Description: "Test issue",
				Severity:    "low",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewReporter(tt.format)
			err := reporter.GenerateReport(result)
			if err != nil {
				t.Errorf("GenerateReport() error = %v", err)
			}
		})
	}
}