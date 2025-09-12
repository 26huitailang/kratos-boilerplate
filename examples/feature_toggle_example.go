package examples
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"kratos-boilerplate/internal/pkg/feature"
	"github.com/go-kratos/kratos/v2/log"
)

// 这是一个展示如何使用功能开关系统的示例

func main() {
	logger := log.NewStdLogger(os.Stdout)
	
	// 1. 创建文件存储
	repository := feature.NewFileRepository("./examples/features_example.yaml", "yaml", logger)
	
	// 2. 创建策略评估器
	evaluator := feature.NewCompositeStrategy()
	
	// 3. 创建功能开关管理器
	toggleManager := feature.NewToggleManager(repository, evaluator, logger)
	
	// 4. 初始化
	if err := toggleManager.Initialize(context.Background()); err != nil {
		fmt.Printf("Failed to initialize toggle manager: %v\n", err)
		return
	}
	
	fmt.Println("=== 功能开关系统示例 ===")
	
	// 基础用法示例
	basicUsageExample(toggleManager)
	
	// 策略示例
	strategyExamples(toggleManager)
	
	// 动态配置示例
	dynamicConfigExample(toggleManager)
	
	// 事件订阅示例
	eventSubscriptionExample(toggleManager)
	
	fmt.Println("\n示例运行完成!")
}

func basicUsageExample(toggleManager feature.FeatureToggle) {
	fmt.Println("\n1. 基础用法示例:")
	
	ctx := context.Background()
	
	// 检查验证码功能是否启用
	if toggleManager.IsEnabled(ctx, feature.FeatureCaptcha) {
		fmt.Println("✓ 验证码功能已启用")
	} else {
		fmt.Println("✗ 验证码功能已禁用")
	}
	
	// 检查TOTP功能是否启用
	if toggleManager.IsEnabled(ctx, feature.FeatureTOTP) {
		fmt.Println("✓ TOTP功能已启用")
	} else {
		fmt.Println("✗ TOTP功能已禁用")
	}
}

func strategyExamples(toggleManager feature.FeatureToggle) {
	fmt.Println("\n2. 策略示例:")
	
	ctx := context.Background()
	
	// 测试不同的用户上下文
	testUsers := []*feature.EvaluationContext{
		{
			UserID:      "user123",
			UserType:    "premium",
			Environment: "production",
			Attributes:  map[string]string{"role": "admin"},
		},
		{
			UserID:      "user456",
			UserType:    "basic",
			Environment: "production",
			Attributes:  map[string]string{"role": "user"},
		},
		{
			UserID:      "user789",
			UserType:    "beta_tester",
			Environment: "staging",
			Attributes:  map[string]string{"role": "user"},
		},
	}
	
	// 测试Beta功能开关（用户策略）
	for _, user := range testUsers {
		enabled := toggleManager.IsEnabledWithContext(ctx, "beta_features", user)
		fmt.Printf("用户 %s (类型: %s, 角色: %s) - Beta功能: %v\n", 
			user.UserID, user.UserType, user.Attributes["role"], enabled)
	}
	
	// 测试百分比发布
	fmt.Println("\n百分比发布测试:")
	testUserIDs := []string{"user1", "user2", "user3", "user4", "user5", "user6", "user7", "user8", "user9", "user10"}
	enabledCount := 0
	
	for _, userID := range testUserIDs {
		evalCtx := &feature.EvaluationContext{UserID: userID}
		if toggleManager.IsEnabledWithContext(ctx, "new_user_dashboard", evalCtx) {
			enabledCount++
			fmt.Printf("✓ %s\n", userID)
		} else {
			fmt.Printf("✗ %s\n", userID)
		}
	}
	
	fmt.Printf("百分比发布结果: %d/%d 用户启用 (%.1f%%)\n", 
		enabledCount, len(testUserIDs), float64(enabledCount)/float64(len(testUserIDs))*100)
}

func dynamicConfigExample(toggleManager feature.FeatureToggle) {
	fmt.Println("\n3. 动态配置示例:")
	
	// 创建新的功能开关
	newConfig := &feature.ToggleConfig{
		Enabled:     true,
		Strategy:    feature.StrategyUser,
		Rules: map[string]interface{}{
			"user_types": []interface{}{"admin", "moderator"},
		},
		Description: "动态创建的功能开关示例",
		Tags:        []string{"example", "dynamic"},
	}
	
	// 更新配置
	if err := toggleManager.UpdateToggle("dynamic_feature", newConfig); err != nil {
		fmt.Printf("更新配置失败: %v\n", err)
		return
	}
	
	fmt.Println("✓ 成功创建动态功能开关")
	
	// 测试新功能
	ctx := context.Background()
	adminUser := &feature.EvaluationContext{
		UserID:   "admin1",
		UserType: "admin",
	}
	
	regularUser := &feature.EvaluationContext{
		UserID:   "user1",
		UserType: "regular",
	}
	
	fmt.Printf("管理员用户访问动态功能: %v\n", 
		toggleManager.IsEnabledWithContext(ctx, "dynamic_feature", adminUser))
	fmt.Printf("普通用户访问动态功能: %v\n", 
		toggleManager.IsEnabledWithContext(ctx, "dynamic_feature", regularUser))
}

func eventSubscriptionExample(toggleManager feature.FeatureToggle) {
	fmt.Println("\n4. 事件订阅示例:")
	
	// 订阅配置变更事件
	changeCount := 0
	callback := func(flag feature.FeatureFlag, oldConfig, newConfig *feature.ToggleConfig) {
		changeCount++
		fmt.Printf("📢 功能开关变更通知 #%d: %s\n", changeCount, flag)
		if oldConfig == nil {
			fmt.Printf("   新增功能: 启用=%v, 策略=%s\n", newConfig.Enabled, newConfig.Strategy)
		} else if newConfig == nil {
			fmt.Printf("   删除功能: %s\n", flag)
		} else {
			fmt.Printf("   更新功能: %v -> %v\n", oldConfig.Enabled, newConfig.Enabled)
		}
	}
	
	if err := toggleManager.Subscribe(callback); err != nil {
		fmt.Printf("订阅事件失败: %v\n", err)
		return
	}
	
	fmt.Println("✓ 已订阅功能开关变更事件")
	
	// 触发一些变更来演示事件
	toggleManager.EnableFeature("example_feature_1")
	time.Sleep(10 * time.Millisecond) // 等待异步回调
	
	toggleManager.DisableFeature("example_feature_1")
	time.Sleep(10 * time.Millisecond)
	
	toggleManager.DeleteToggle("example_feature_1")
	time.Sleep(10 * time.Millisecond)
	
	fmt.Printf("总共收到 %d 个变更通知\n", changeCount)
}

// 业务逻辑示例
func exampleBusinessLogic(toggleManager feature.FeatureToggle, userID string) {
	ctx := context.Background()
	evalCtx := &feature.EvaluationContext{
		UserID:      userID,
		Environment: "production",
	}
	
	// 认证流程
	fmt.Printf("\n用户 %s 的认证流程:\n", userID)
	
	if toggleManager.IsEnabledWithContext(ctx, feature.FeatureCaptcha, evalCtx) {
		fmt.Println("1. ✓ 显示验证码")
	} else {
		fmt.Println("1. ✗ 跳过验证码")
	}
	
	if toggleManager.IsEnabledWithContext(ctx, feature.FeatureTOTP, evalCtx) {
		fmt.Println("2. ✓ 要求TOTP验证")
	} else {
		fmt.Println("2. ✗ 跳过TOTP验证")
	}
	
	if toggleManager.IsEnabledWithContext(ctx, feature.FeatureAccountLockout, evalCtx) {
		fmt.Println("3. ✓ 检查账户锁定状态")
	} else {
		fmt.Println("3. ✗ 跳过账户锁定检查")
	}
	
	fmt.Println("4. ✓ 认证完成")
}