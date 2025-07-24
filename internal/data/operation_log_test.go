package data

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"kratos-boilerplate/internal/biz"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试NewOperationLogRepo
func TestNewOperationLogRepo(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewOperationLogRepo(data, logger)
	assert.NotNil(t, repo)
}

// 测试CreateLog
func TestCreateLog(t *testing.T) {
	// 创建mock数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 创建数据层
	data := &Data{db: db}

	// 创建操作日志仓储
	logger := log.NewStdLogger(os.Stdout)
	repo := NewOperationLogRepo(data, logger)
	operationLogRepo := repo.(*operationLogRepo)

	// 测试数据
	logEntry := &biz.OperationLog{
		ID:        1,
		UserID:    123,
		Username:  "testuser",
		Operation: "login",
		Target:    "auth",
		Content:   "user login attempt",
		Result:    "success",
		CreatedAt: time.Now(),
	}

	// 设置mock期望
	mock.ExpectExec("INSERT INTO operation_logs").
		WithArgs(logEntry.UserID, logEntry.Username, logEntry.Operation, logEntry.Target, logEntry.Content, logEntry.Result, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 执行测试
	ctx := context.Background()
	err = operationLogRepo.CreateLog(ctx, logEntry)

	// 验证结果
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// 测试CreateLog - 数据库错误
func TestCreateLog_DatabaseError(t *testing.T) {
	// 创建mock数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 创建数据层
	data := &Data{db: db}

	// 创建操作日志仓储
	logger := log.NewStdLogger(os.Stdout)
	repo := NewOperationLogRepo(data, logger)
	operationLogRepo := repo.(*operationLogRepo)

	// 测试数据
	logEntry := &biz.OperationLog{
		ID:        1,
		UserID:    123,
		Username:  "testuser",
		Operation: "login",
		Target:    "auth",
		Content:   "user login attempt",
		Result:    "success",
		CreatedAt: time.Now(),
	}

	// 设置mock期望 - 返回错误
	mock.ExpectExec("INSERT INTO operation_logs").
		WithArgs(logEntry.UserID, logEntry.Username, logEntry.Operation, logEntry.Target, logEntry.Content, logEntry.Result, sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)

	// 执行测试
	ctx := context.Background()
	err = operationLogRepo.CreateLog(ctx, logEntry)

	// 验证结果
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// 测试ListLogs
func TestListLogs(t *testing.T) {
	// 创建mock数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 创建数据层
	data := &Data{db: db}

	// 创建操作日志仓储
	logger := log.NewStdLogger(os.Stdout)
	repo := NewOperationLogRepo(data, logger)
	operationLogRepo := repo.(*operationLogRepo)

	// 测试数据
	userID := int64(123)
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	// 设置mock期望
	rows := sqlmock.NewRows([]string{"id", "user_id", "username", "operation", "target", "content", "result", "created_at"}).
		AddRow(1, userID, "testuser", "login", "auth", "user login attempt", "success", time.Now()).
		AddRow(2, userID, "testuser", "logout", "auth", "user logout", "success", time.Now())

	mock.ExpectQuery("SELECT id, user_id, username, operation, target, content, result, created_at FROM operation_logs").
		WithArgs(userID, startTime, endTime).
		WillReturnRows(rows)

	// 执行测试
	ctx := context.Background()
	logs, err := operationLogRepo.ListLogs(ctx, userID, startTime, endTime)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, logs)
	assert.Len(t, logs, 2)
	assert.Equal(t, userID, logs[0].UserID)
	assert.Equal(t, "testuser", logs[0].Username)
	assert.Equal(t, "login", logs[0].Operation)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// 测试ListLogs - 无结果
func TestListLogs_NoResults(t *testing.T) {
	// 创建mock数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 创建数据层
	data := &Data{db: db}

	// 创建操作日志仓储
	logger := log.NewStdLogger(os.Stdout)
	repo := NewOperationLogRepo(data, logger)
	operationLogRepo := repo.(*operationLogRepo)

	// 测试数据
	userID := int64(123)
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	// 设置mock期望 - 返回空结果
	rows := sqlmock.NewRows([]string{"id", "user_id", "username", "operation", "target", "content", "result", "created_at"})

	mock.ExpectQuery("SELECT id, user_id, username, operation, target, content, result, created_at FROM operation_logs").
		WithArgs(userID, startTime, endTime).
		WillReturnRows(rows)

	// 执行测试
	ctx := context.Background()
	logs, err := operationLogRepo.ListLogs(ctx, userID, startTime, endTime)

	// 验证结果
	assert.NoError(t, err)
	assert.Len(t, logs, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// 测试ListLogs - 数据库错误
func TestListLogs_DatabaseError(t *testing.T) {
	// 创建mock数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 创建数据层
	data := &Data{db: db}

	// 创建操作日志仓储
	logger := log.NewStdLogger(os.Stdout)
	repo := NewOperationLogRepo(data, logger)
	operationLogRepo := repo.(*operationLogRepo)

	// 测试数据
	userID := int64(123)
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	// 设置mock期望 - 返回错误
	mock.ExpectQuery("SELECT id, user_id, username, operation, target, content, result, created_at FROM operation_logs").
		WithArgs(userID, startTime, endTime).
		WillReturnError(sql.ErrConnDone)

	// 执行测试
	ctx := context.Background()
	logs, err := operationLogRepo.ListLogs(ctx, userID, startTime, endTime)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, logs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// 测试操作日志数据结构
func TestOperationLogStructure(t *testing.T) {
	// 创建操作日志实例
	logEntry := &biz.OperationLog{
		ID:        1,
		UserID:    123,
		Username:  "testuser",
		Operation: "login",
		Target:    "auth",
		Content:   "user login attempt",
		Result:    "success",
		CreatedAt: time.Now(),
	}

	// 验证结构
	assert.Equal(t, int64(1), logEntry.ID)
	assert.Equal(t, int64(123), logEntry.UserID)
	assert.Equal(t, "testuser", logEntry.Username)
	assert.Equal(t, "login", logEntry.Operation)
	assert.Equal(t, "auth", logEntry.Target)
	assert.Equal(t, "user login attempt", logEntry.Content)
	assert.Equal(t, "success", logEntry.Result)
	assert.NotZero(t, logEntry.CreatedAt)
}

// 测试操作日志仓储接口实现
func TestOperationLogRepoInterface(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewOperationLogRepo(data, logger)

	// 验证实现了接口
	var _ biz.OperationLogRepo = repo
	assert.NotNil(t, repo)
}
