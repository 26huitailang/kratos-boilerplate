package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var (
		dir    = flag.String("dir", ".", "Directory to scan for Go files")
		config = flag.String("config", "", "Path to configuration file")
		output = flag.String("output", "console", "Output format: console, json, html")
		verbose = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	if *verbose {
		fmt.Printf("Scanning directory: %s\n", *dir)
		if *config != "" {
			fmt.Printf("Using config file: %s\n", *config)
		}
	}

	// 创建日志检查器
	checker := NewLogChecker(*config)

	// 扫描目录
	results, err := checker.ScanDirectory(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	// 生成报告
	reporter := NewReporter(*output)
	if err := reporter.GenerateReport(results); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}

	// 如果发现问题，返回非零退出码
	if len(results.Issues) > 0 {
		os.Exit(1)
	}
}