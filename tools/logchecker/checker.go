package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// LogChecker 日志规范检查器
type LogChecker struct {
	config *Config
	fset   *token.FileSet
}

// Issue 表示检测到的问题
type Issue struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Method      string `json:"method"`
	Suggestion  string `json:"suggestion"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// ScanResult 扫描结果
type ScanResult struct {
	TotalFiles int     `json:"total_files"`
	ScannedFiles int   `json:"scanned_files"`
	Issues     []Issue `json:"issues"`
}

// NewLogChecker 创建新的日志检查器
func NewLogChecker(configPath string) *LogChecker {
	config := LoadConfig(configPath)
	return &LogChecker{
		config: config,
		fset:   token.NewFileSet(),
	}
}

// ScanDirectory 扫描目录中的Go文件
func (c *LogChecker) ScanDirectory(dir string) (*ScanResult, error) {
	result := &ScanResult{
		Issues: make([]Issue, 0),
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过非Go文件
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// 跳过测试文件（可配置）
		if c.config.SkipTestFiles && strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// 检查是否在忽略列表中
		if c.shouldIgnoreFile(path) {
			return nil
		}

		result.TotalFiles++

		// 解析文件
		issues, err := c.scanFile(path)
		if err != nil {
			return err
		}

		result.ScannedFiles++
		result.Issues = append(result.Issues, issues...)

		return nil
	})

	return result, err
}

// scanFile 扫描单个Go文件
func (c *LogChecker) scanFile(filename string) ([]Issue, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// 解析AST
	node, err := parser.ParseFile(c.fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// 创建访问器
	visitor := &logVisitor{
		checker:  c,
		filename: filename,
		issues:   make([]Issue, 0),
	}

	// 遍历AST
	ast.Walk(visitor, node)

	return visitor.issues, nil
}

// shouldIgnoreFile 检查文件是否应该被忽略
func (c *LogChecker) shouldIgnoreFile(path string) bool {
	for _, pattern := range c.config.IgnoreFiles {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}