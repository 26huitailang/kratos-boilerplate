package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"
	"time"
)

// Reporter 报告生成器
type Reporter struct {
	format string
}

// NewReporter 创建新的报告生成器
func NewReporter(format string) *Reporter {
	return &Reporter{
		format: format,
	}
}

// GenerateReport 生成报告
func (r *Reporter) GenerateReport(result *ScanResult) error {
	switch r.format {
	case "json":
		return r.generateJSONReport(result)
	case "html":
		return r.generateHTMLReport(result)
	case "console":
		return r.generateConsoleReport(result)
	default:
		return fmt.Errorf("unsupported output format: %s", r.format)
	}
}

// generateConsoleReport 生成控制台报告
func (r *Reporter) generateConsoleReport(result *ScanResult) error {
	fmt.Printf("\n=== 日志规范检查报告 ===\n")
	fmt.Printf("扫描时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("总文件数: %d\n", result.TotalFiles)
	fmt.Printf("已扫描文件数: %d\n", result.ScannedFiles)
	fmt.Printf("发现问题数: %d\n\n", len(result.Issues))

	if len(result.Issues) == 0 {
		fmt.Printf("✅ 未发现日志规范问题\n")
		return nil
	}

	// 按严重程度分组
	issuesBySeverity := r.groupIssuesBySeverity(result.Issues)

	// 输出高严重程度问题
	if issues, exists := issuesBySeverity["high"]; exists {
		fmt.Printf("🔴 高严重程度问题 (%d个):\n", len(issues))
		r.printIssues(issues)
		fmt.Println()
	}

	// 输出中等严重程度问题
	if issues, exists := issuesBySeverity["medium"]; exists {
		fmt.Printf("🟡 中等严重程度问题 (%d个):\n", len(issues))
		r.printIssues(issues)
		fmt.Println()
	}

	// 输出低严重程度问题
	if issues, exists := issuesBySeverity["low"]; exists {
		fmt.Printf("🟢 低严重程度问题 (%d个):\n", len(issues))
		r.printIssues(issues)
		fmt.Println()
	}

	// 输出修改建议
	fmt.Printf("\n=== 修改建议 ===\n")
	fmt.Printf("1. 使用结构化日志方法（如 Infow, Debugw, Errorw, Warnw）替代格式化日志方法\n")
	fmt.Printf("2. 将格式化参数转换为键值对形式，便于日志分析和脱敏\n")
	fmt.Printf("3. 对于包含敏感信息的日志，确保使用结构化日志以启用自动脱敏\n")
	fmt.Printf("4. 参考项目中的 SafeLogger 使用指南: docs/structured-logging-guide.md\n")

	return nil
}

// printIssues 打印问题列表
func (r *Reporter) printIssues(issues []Issue) {
	for _, issue := range issues {
		fmt.Printf("  📁 %s:%d:%d\n", issue.File, issue.Line, issue.Column)
		fmt.Printf("     问题: %s\n", issue.Description)
		fmt.Printf("     建议: 使用 %s 替代 %s\n", issue.Suggestion, issue.Method)
		fmt.Println()
	}
}

// generateJSONReport 生成JSON报告
func (r *Reporter) generateJSONReport(result *ScanResult) error {
	report := map[string]interface{}{
		"timestamp":     time.Now().Format(time.RFC3339),
		"total_files":   result.TotalFiles,
		"scanned_files": result.ScannedFiles,
		"total_issues":  len(result.Issues),
		"issues":        result.Issues,
		"summary":       r.generateSummary(result.Issues),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// generateHTMLReport 生成HTML报告
func (r *Reporter) generateHTMLReport(result *ScanResult) error {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>日志规范检查报告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f5f5f5; padding: 20px; border-radius: 5px; }
        .summary { margin: 20px 0; }
        .issue { margin: 10px 0; padding: 10px; border-left: 4px solid #ccc; }
        .high { border-left-color: #ff4444; }
        .medium { border-left-color: #ffaa00; }
        .low { border-left-color: #44ff44; }
        .file-path { font-family: monospace; color: #666; }
        .method { font-family: monospace; background-color: #f0f0f0; padding: 2px 4px; }
        .suggestion { color: #0066cc; font-weight: bold; }
    </style>
</head>
<body>
    <div class="header">
        <h1>日志规范检查报告</h1>
        <p>扫描时间: {{.Timestamp}}</p>
        <p>总文件数: {{.TotalFiles}} | 已扫描: {{.ScannedFiles}} | 发现问题: {{.TotalIssues}}</p>
    </div>
    
    <div class="summary">
        <h2>问题汇总</h2>
        <ul>
            <li>高严重程度: {{.Summary.High}}个</li>
            <li>中等严重程度: {{.Summary.Medium}}个</li>
            <li>低严重程度: {{.Summary.Low}}个</li>
        </ul>
    </div>
    
    <div class="issues">
        <h2>详细问题</h2>
        {{range .Issues}}
        <div class="issue {{.Severity}}">
            <div class="file-path">{{.File}}:{{.Line}}:{{.Column}}</div>
            <div>问题: {{.Description}}</div>
            <div>当前方法: <span class="method">{{.Method}}</span></div>
            <div>建议使用: <span class="suggestion">{{.Suggestion}}</span></div>
        </div>
        {{end}}
    </div>
    
    <div class="recommendations">
        <h2>修改建议</h2>
        <ol>
            <li>使用结构化日志方法（如 Infow, Debugw, Errorw, Warnw）替代格式化日志方法</li>
            <li>将格式化参数转换为键值对形式，便于日志分析和脱敏</li>
            <li>对于包含敏感信息的日志，确保使用结构化日志以启用自动脱敏</li>
            <li>参考项目中的 SafeLogger 使用指南</li>
        </ol>
    </div>
</body>
</html>
`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"Timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"TotalFiles":  result.TotalFiles,
		"ScannedFiles": result.ScannedFiles,
		"TotalIssues": len(result.Issues),
		"Issues":      result.Issues,
		"Summary":     r.generateSummary(result.Issues),
	}

	return t.Execute(os.Stdout, data)
}

// groupIssuesBySeverity 按严重程度分组问题
func (r *Reporter) groupIssuesBySeverity(issues []Issue) map[string][]Issue {
	groups := make(map[string][]Issue)
	for _, issue := range issues {
		groups[issue.Severity] = append(groups[issue.Severity], issue)
	}
	
	// 对每个组内的问题按文件名排序
	for severity := range groups {
		sort.Slice(groups[severity], func(i, j int) bool {
			if groups[severity][i].File == groups[severity][j].File {
				return groups[severity][i].Line < groups[severity][j].Line
			}
			return groups[severity][i].File < groups[severity][j].File
		})
	}
	
	return groups
}

// generateSummary 生成问题汇总
func (r *Reporter) generateSummary(issues []Issue) map[string]int {
	summary := map[string]int{
		"High":   0,
		"Medium": 0,
		"Low":    0,
	}
	
	for _, issue := range issues {
		switch strings.ToLower(issue.Severity) {
		case "high":
			summary["High"]++
		case "medium":
			summary["Medium"]++
		case "low":
			summary["Low"]++
		}
	}
	
	return summary
}