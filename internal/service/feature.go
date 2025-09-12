package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	v1 "kratos-boilerplate/api/feature/v1"
	"kratos-boilerplate/internal/pkg/feature"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FeatureToggleService 功能开关服务实现
type FeatureToggleService struct {
	v1.UnimplementedFeatureToggleServer

	toggleManager feature.FeatureToggle
	logger        *log.Helper
}

// NewFeatureToggleService 创建功能开关服务
func NewFeatureToggleService(toggleManager feature.FeatureToggle, logger log.Logger) *FeatureToggleService {
	return &FeatureToggleService{
		toggleManager: toggleManager,
		logger:        log.NewHelper(logger),
	}
}

// ListToggles 获取所有功能开关
func (s *FeatureToggleService) ListToggles(ctx context.Context, req *v1.ListTogglesRequest) (*v1.ListTogglesReply, error) {
	toggles := s.toggleManager.ListToggles()

	var filteredToggles []*v1.FeatureToggleInfo

	for flag, config := range toggles {
		// 根据启用状态过滤
		if req.EnabledOnly != nil && *req.EnabledOnly && !config.Enabled {
			continue
		}

		// 根据标签过滤
		if len(req.Tags) > 0 {
			if !s.hasAnyTag(config.Tags, req.Tags) {
				continue
			}
		}

		toggle := &v1.FeatureToggleInfo{
			Flag:   string(flag),
			Config: s.convertToProtoConfig(config),
		}

		filteredToggles = append(filteredToggles, toggle)
	}

	total := len(filteredToggles)

	// 分页处理
	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	start := int((page - 1) * pageSize)
	end := int(page * pageSize)

	if start >= total {
		filteredToggles = []*v1.FeatureToggleInfo{}
	} else if end > total {
		filteredToggles = filteredToggles[start:]
	} else {
		filteredToggles = filteredToggles[start:end]
	}

	return &v1.ListTogglesReply{
		Toggles:  filteredToggles,
		Total:    int32(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetToggle 获取单个功能开关
func (s *FeatureToggleService) GetToggle(ctx context.Context, req *v1.GetToggleRequest) (*v1.GetToggleReply, error) {
	flag := feature.FeatureFlag(req.Flag)

	config, err := s.toggleManager.GetToggleConfig(flag)
	if err != nil {
		return nil, errors.NotFound("TOGGLE_NOT_FOUND", fmt.Sprintf("Feature toggle %s not found", req.Flag))
	}

	return &v1.GetToggleReply{
		Toggle: &v1.FeatureToggleInfo{
			Flag:   req.Flag,
			Config: s.convertToProtoConfig(config),
		},
	}, nil
}

// UpdateToggle 更新功能开关
func (s *FeatureToggleService) UpdateToggle(ctx context.Context, req *v1.UpdateToggleRequest) (*v1.UpdateToggleReply, error) {
	flag := feature.FeatureFlag(req.Flag)

	config := s.convertFromProtoConfig(req.Config)

	if err := s.toggleManager.UpdateToggle(flag, config); err != nil {
		return nil, errors.InternalServer("UPDATE_FAILED", fmt.Sprintf("Failed to update toggle: %v", err))
	}

	updatedConfig, _ := s.toggleManager.GetToggleConfig(flag)

	return &v1.UpdateToggleReply{
		Toggle: &v1.FeatureToggleInfo{
			Flag:   req.Flag,
			Config: s.convertToProtoConfig(updatedConfig),
		},
	}, nil
}

// EnableToggle 启用功能开关
func (s *FeatureToggleService) EnableToggle(ctx context.Context, req *v1.EnableToggleRequest) (*v1.EnableToggleReply, error) {
	flag := feature.FeatureFlag(req.Flag)

	if err := s.toggleManager.EnableFeature(flag); err != nil {
		return nil, errors.InternalServer("ENABLE_FAILED", fmt.Sprintf("Failed to enable toggle: %v", err))
	}

	config, _ := s.toggleManager.GetToggleConfig(flag)

	return &v1.EnableToggleReply{
		Toggle: &v1.FeatureToggleInfo{
			Flag:   req.Flag,
			Config: s.convertToProtoConfig(config),
		},
	}, nil
}

// DisableToggle 禁用功能开关
func (s *FeatureToggleService) DisableToggle(ctx context.Context, req *v1.DisableToggleRequest) (*v1.DisableToggleReply, error) {
	flag := feature.FeatureFlag(req.Flag)

	if err := s.toggleManager.DisableFeature(flag); err != nil {
		return nil, errors.InternalServer("DISABLE_FAILED", fmt.Sprintf("Failed to disable toggle: %v", err))
	}

	config, _ := s.toggleManager.GetToggleConfig(flag)

	return &v1.DisableToggleReply{
		Toggle: &v1.FeatureToggleInfo{
			Flag:   req.Flag,
			Config: s.convertToProtoConfig(config),
		},
	}, nil
}

// DeleteToggle 删除功能开关
func (s *FeatureToggleService) DeleteToggle(ctx context.Context, req *v1.DeleteToggleRequest) (*v1.DeleteToggleReply, error) {
	flag := feature.FeatureFlag(req.Flag)

	if err := s.toggleManager.DeleteToggle(flag); err != nil {
		return nil, errors.InternalServer("DELETE_FAILED", fmt.Sprintf("Failed to delete toggle: %v", err))
	}

	return &v1.DeleteToggleReply{
		Success: true,
	}, nil
}

// EvaluateToggle 评估功能开关
func (s *FeatureToggleService) EvaluateToggle(ctx context.Context, req *v1.EvaluateToggleRequest) (*v1.EvaluateToggleReply, error) {
	flag := feature.FeatureFlag(req.Flag)

	var evalCtx *feature.EvaluationContext
	if req.Context != nil {
		evalCtx = &feature.EvaluationContext{
			UserID:      req.Context.UserId,
			UserType:    req.Context.UserType,
			Environment: req.Context.Environment,
			Version:     req.Context.Version,
			Attributes:  req.Context.Attributes,
		}
	}

	enabled := s.toggleManager.IsEnabledWithContext(ctx, flag, evalCtx)

	reason := "Feature is enabled"
	if !enabled {
		reason = "Feature is disabled or conditions not met"
	}

	return &v1.EvaluateToggleReply{
		Enabled: enabled,
		Reason:  reason,
	}, nil
}

// GetStats 获取功能开关统计
func (s *FeatureToggleService) GetStats(ctx context.Context, req *v1.GetStatsRequest) (*v1.GetStatsReply, error) {
	if manager, ok := s.toggleManager.(*feature.ToggleManager); ok {
		stats := manager.GetStats()

		strategies := make(map[string]int32)
		if strategyStats, ok := stats["strategies"].(map[feature.FeatureStrategy]int); ok {
			for strategy, count := range strategyStats {
				strategies[string(strategy)] = int32(count)
			}
		}

		return &v1.GetStatsReply{
			TotalToggles:  int32(stats["total_toggles"].(int)),
			EnabledCount:  int32(stats["enabled_count"].(int)),
			DisabledCount: int32(stats["disabled_count"].(int)),
			Strategies:    strategies,
		}, nil
	}

	return nil, errors.InternalServer("STATS_UNAVAILABLE", "Stats not available")
}

// 辅助方法

func (s *FeatureToggleService) convertToProtoConfig(config *feature.ToggleConfig) *v1.ToggleConfig {
	if config == nil {
		return nil
	}

	rules := make(map[string]string)
	for key, value := range config.Rules {
		rules[key] = fmt.Sprintf("%v", value)
	}

	return &v1.ToggleConfig{
		Enabled:     config.Enabled,
		Strategy:    s.convertToProtoStrategy(config.Strategy),
		Rules:       rules,
		Description: config.Description,
		Tags:        config.Tags,
		CreatedAt:   timestamppb.New(config.CreatedAt),
		UpdatedAt:   timestamppb.New(config.UpdatedAt),
	}
}

func (s *FeatureToggleService) convertFromProtoConfig(protoConfig *v1.ToggleConfig) *feature.ToggleConfig {
	if protoConfig == nil {
		return nil
	}

	rules := make(map[string]interface{})
	for key, value := range protoConfig.Rules {
		// 尝试转换为合适的类型
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			rules[key] = floatVal
		} else if boolVal, err := strconv.ParseBool(value); err == nil {
			rules[key] = boolVal
		} else if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			// 简单的数组解析
			arrayStr := strings.Trim(value, "[]")
			if arrayStr != "" {
				items := strings.Split(arrayStr, ",")
				for i, item := range items {
					items[i] = strings.TrimSpace(item)
				}
				rules[key] = items
			} else {
				rules[key] = []string{}
			}
		} else {
			rules[key] = value
		}
	}

	config := &feature.ToggleConfig{
		Enabled:     protoConfig.Enabled,
		Strategy:    s.convertFromProtoStrategy(protoConfig.Strategy),
		Rules:       rules,
		Description: protoConfig.Description,
		Tags:        protoConfig.Tags,
	}

	if protoConfig.CreatedAt != nil {
		config.CreatedAt = protoConfig.CreatedAt.AsTime()
	}
	if protoConfig.UpdatedAt != nil {
		config.UpdatedAt = protoConfig.UpdatedAt.AsTime()
	}

	return config
}

func (s *FeatureToggleService) convertToProtoStrategy(strategy feature.FeatureStrategy) v1.FeatureStrategy {
	switch strategy {
	case feature.StrategyPercentage:
		return v1.FeatureStrategy_PERCENTAGE
	case feature.StrategyUser:
		return v1.FeatureStrategy_USER
	case feature.StrategyTime:
		return v1.FeatureStrategy_TIME
	case feature.StrategyEnvironment:
		return v1.FeatureStrategy_ENVIRONMENT
	default:
		return v1.FeatureStrategy_SIMPLE
	}
}

func (s *FeatureToggleService) convertFromProtoStrategy(protoStrategy v1.FeatureStrategy) feature.FeatureStrategy {
	switch protoStrategy {
	case v1.FeatureStrategy_PERCENTAGE:
		return feature.StrategyPercentage
	case v1.FeatureStrategy_USER:
		return feature.StrategyUser
	case v1.FeatureStrategy_TIME:
		return feature.StrategyTime
	case v1.FeatureStrategy_ENVIRONMENT:
		return feature.StrategyEnvironment
	default:
		return feature.StrategySimple
	}
}

func (s *FeatureToggleService) hasAnyTag(configTags, filterTags []string) bool {
	for _, filterTag := range filterTags {
		for _, configTag := range configTags {
			if configTag == filterTag {
				return true
			}
		}
	}
	return false
}
