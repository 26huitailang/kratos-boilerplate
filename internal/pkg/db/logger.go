package db

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm/logger"
)

// GormLogger GORM日志器实现
type GormLogger struct {
	logger                    log.Logger
	logLevel                  logger.LogLevel
	slowThreshold             time.Duration
	colorful                  bool
	ignoreRecordNotFoundError bool
	logSQL                    bool
}

// NewGormLogger 创建GORM日志器
func NewGormLogger(logger log.Logger, config LogConfig) logger.Interface {
	return &GormLogger{
		logger:                    logger,
		logLevel:                  config.Level,
		slowThreshold:             config.SlowThreshold,
		colorful:                  config.Colorful,
		ignoreRecordNotFoundError: config.IgnoreRecordNotFoundError,
		logSQL:                    config.LogSQL,
	}
}

// LogMode 设置日志级别
func (gl *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *gl
	newLogger.logLevel = level
	return &newLogger
}

// Info 信息日志
func (gl *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if gl.logLevel >= logger.Info {
		gl.logger.Log(log.LevelInfo, "msg", fmt.Sprintf(msg, data...))
	}
}

// Warn 警告日志
func (gl *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if gl.logLevel >= logger.Warn {
		gl.logger.Log(log.LevelWarn, "msg", fmt.Sprintf(msg, data...))
	}
}

// Error 错误日志
func (gl *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if gl.logLevel >= logger.Error {
		gl.logger.Log(log.LevelError, "msg", fmt.Sprintf(msg, data...))
	}
}

// Trace 追踪日志
func (gl *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if gl.logLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	
	logData := []interface{}{
		"duration", elapsed.String(),
		"rows", rows,
	}
	
	if gl.logSQL {
		logData = append(logData, "sql", sql)
	}

	switch {
	case err != nil && gl.logLevel >= logger.Error && (!gl.ignoreRecordNotFoundError || !isRecordNotFoundError(err)):
		logData = append(logData, "error", err.Error())
		gl.logger.Log(log.LevelError, logData...)
	case elapsed > gl.slowThreshold && gl.slowThreshold != 0 && gl.logLevel >= logger.Warn:
		logData = append(logData, "slow_query", true)
		gl.logger.Log(log.LevelWarn, logData...)
	case gl.logLevel == logger.Info:
		gl.logger.Log(log.LevelInfo, logData...)
	}
}

// isRecordNotFoundError 检查是否为记录未找到错误
func isRecordNotFoundError(err error) bool {
	return err.Error() == "record not found"
}