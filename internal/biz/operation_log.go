package biz

import (
	"context"
	"time"
)

// OperationLog 操作日志六要素
type OperationLog struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"` // 用户标识
	Username  string    `json:"username"`
	Operation string    `json:"operation"` // 操作类型
	Target    string    `json:"target"`    // 操作对象
	Content   string    `json:"content"`   // 操作内容
	Result    string    `json:"result"`    // 操作结果
	CreatedAt time.Time `json:"created_at"` // 操作时间
}

// OperationLogRepo 操作日志仓储接口
type OperationLogRepo interface {
	CreateLog(ctx context.Context, log *OperationLog) error
	ListLogs(ctx context.Context, userID int64, startTime, endTime time.Time) ([]*OperationLog, error)
}