package examples

import (
	"context"
	"fmt"
	"log"
	"os"

	"kratos-boilerplate/internal/pkg/feature"

	kratosLog "github.com/go-kratos/kratos/v2/log"
)

func DemoHierarchicalFeatures() {
	// 创建logger
	logger := kratosLog.NewStdLogger(os.Stdout)

	// 创建文件存储库
	repo := feature.NewFileRepository("./configs/features.yaml", "yaml", logger)

	// 创建策略评估器
	evaluator := feature.NewCompositeStrategy()

	// 创建功能开关管理器
	manager := feature.NewToggleManager(repo, evaluator, logger)

	// 初始化
	if err := manager.Initialize(context.Background()); err != nil {
		log.Fatalf("Failed to initialize feature manager: %v", err)
	}

	// 演示分层功能检查
	fmt.Println("=== 分层功能集合演示 ===")

	// 检查功能域
	fmt.Printf("用户域是否有启用的功能: %v\n", manager.IsDomainEnabled("user"))
	fmt.Printf("系统域是否有启用的功能: %v\n", manager.IsDomainEnabled("system"))
	fmt.Printf("报表域是否有启用的功能: %v\n", manager.IsDomainEnabled("report"))

	// 获取域下的功能列表
	userFeatures := manager.GetDomainFeatures("user")
	fmt.Printf("用户域下的功能: %v\n", userFeatures)

	// 检查具体功能集合
	fmt.Printf("用户管理功能是否启用: %v\n", manager.IsFeatureEnabled("user.management"))
	fmt.Printf("用户资料功能是否启用: %v\n", manager.IsFeatureEnabled("user.profile"))
	fmt.Printf("高级报表功能是否启用: %v\n", manager.IsFeatureEnabled("report.advanced"))

	// 检查功能集合的具体能力
	fmt.Println("\n=== 功能集合能力检查 ===")
	fmt.Printf("用户管理是否包含创建权限: %v\n", manager.HasPermission("user.management", "user.create"))
	fmt.Printf("用户管理是否包含用户创建API: %v\n", manager.HasAPI("user.management", "POST /api/v1/users"))
	fmt.Printf("用户管理是否包含管理页面路由: %v\n", manager.HasRoute("user.management", "/admin/users"))
	fmt.Printf("用户管理是否包含最大用户数配置: %v\n", manager.HasConfig("user.management", "user.max_count"))
	fmt.Printf("用户管理是否包含用户创建日志: %v\n", manager.HasAuditLog("user.management", "user_created"))

	// 批量检查
	fmt.Println("\n=== 批量检查 ===")
	fmt.Printf("是否有任何用户功能启用: %v\n", manager.IsAnyFeatureEnabled("user.management", "user.profile", "user.security"))
	fmt.Printf("是否所有用户功能都启用: %v\n", manager.AreAllFeaturesEnabled("user.management", "user.profile", "user.security"))

	// 获取功能树
	fmt.Println("\n=== 功能树结构 ===")
	tree := manager.GetFeatureTree()
	for domain, features := range tree {
		fmt.Printf("%s: %v\n", domain, features)
	}

	// 获取功能能力
	fmt.Println("\n=== 功能能力详情 ===")
	capabilities := manager.GetFeatureCapabilities("user.management")
	if capabilities != nil {
		fmt.Printf("用户管理功能包含 %d 个权限点\n", len(capabilities.Permissions))
		fmt.Printf("用户管理功能包含 %d 个API\n", len(capabilities.APIs))
		fmt.Printf("用户管理功能包含 %d 个路由\n", len(capabilities.Routes))
		fmt.Printf("用户管理功能包含 %d 个配置点\n", len(capabilities.Configs))
		fmt.Printf("用户管理功能包含 %d 个审计日志\n", len(capabilities.AuditLogs))
	}

	// 导出CSV
	fmt.Println("\n=== CSV导出演示 ===")
	csvData, err := manager.ExportCSV()
	if err != nil {
		log.Printf("Failed to export CSV: %v", err)
	} else {
		fmt.Println("CSV导出成功，前5行内容:")
		lines := splitLines(string(csvData))
		for i, line := range lines {
			if i >= 5 {
				break
			}
			fmt.Println(line)
		}
		fmt.Printf("总共导出 %d 行数据\n", len(lines))
	}

	fmt.Println("\n=== 演示完成 ===")
}

func splitLines(text string) []string {
	var lines []string
	current := ""
	for _, char := range text {
		if char == '\n' {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
