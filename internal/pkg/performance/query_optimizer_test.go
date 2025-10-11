package performance

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-kratos/kratos/v2/log"
)

func TestQueryOptimizer_ExecuteQuery(t *testing.T) {
	// 创建sqlmock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// 创建查询优化器
	logger := log.NewStdLogger(&testWriter{t: t})
	config := DefaultQueryOptimizerConfig()
	config.SlowQueryThreshold = 100 * time.Millisecond // 设置较低的阈值以便测试
	qo := NewQueryOptimizer(db, config, logger)

	// 设置mock期望
	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "test1").
		AddRow(2, "test2")
	mock.ExpectQuery("SELECT \\* FROM users").
		WillReturnRows(rows)

	// 执行查询
	ctx := context.Background()
	resultRows, err := qo.ExecuteQuery(ctx, "SELECT * FROM users")
	if err != nil {
		t.Fatalf("ExecuteQuery failed: %v", err)
	}
	defer resultRows.Close()

	// 验证结果
	count := 0
	for resultRows.Next() {
		count++
	}
	if count != 2 {
		t.Errorf("Expected 2 rows, got %d", count)
	}

	// 验证mock期望
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}

	// 验证统计信息
	stats := qo.GetStats()
	if stats.TotalQueries != 1 {
		t.Errorf("Expected 1 total query, got %d", stats.TotalQueries)
	}
}

func TestQueryOptimizer_ExecuteQueryRow(t *testing.T) {
	// 创建sqlmock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// 创建查询优化器
	logger := log.NewStdLogger(&testWriter{t: t})
	config := DefaultQueryOptimizerConfig()
	config.SlowQueryThreshold = 100 * time.Millisecond // 设置较低的阈值以便测试
	qo := NewQueryOptimizer(db, config, logger)

	// 设置mock期望
	row := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "test")
	mock.ExpectQuery("SELECT id, name FROM users WHERE id = \\$1").
		WithArgs(1).
		WillReturnRows(row)

	// 执行单行查询
	ctx := context.Background()
	resultRow := qo.ExecuteQueryRow(ctx, "SELECT id, name FROM users WHERE id = $1", 1)

	// 验证结果
	var id int
	var name string
	err = resultRow.Scan(&id, &name)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if id != 1 || name != "test" {
		t.Errorf("Expected id=1 and name=test, got id=%d and name=%s", id, name)
	}

	// 验证mock期望
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}

	// 验证统计信息
	stats := qo.GetStats()
	if stats.TotalQueries != 1 {
		t.Errorf("Expected 1 total query, got %d", stats.TotalQueries)
	}
}

func TestQueryOptimizer_ExecuteExec(t *testing.T) {
	// 创建sqlmock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// 创建查询优化器
	logger := log.NewStdLogger(&testWriter{t: t})
	config := DefaultQueryOptimizerConfig()
	config.SlowQueryThreshold = 100 * time.Millisecond // 设置较低的阈值以便测试
	qo := NewQueryOptimizer(db, config, logger)

	// 设置mock期望
	mock.ExpectExec("UPDATE users SET name = \\$1 WHERE id = \\$2").
		WithArgs("new_name", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 执行更新操作
	ctx := context.Background()
	result, err := qo.ExecuteExec(ctx, "UPDATE users SET name = $1 WHERE id = $2", "new_name", 1)
	if err != nil {
		t.Fatalf("ExecuteExec failed: %v", err)
	}

	// 验证结果
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("RowsAffected failed: %v", err)
	}
	if rowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", rowsAffected)
	}

	// 验证mock期望
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}

	// 验证统计信息
	stats := qo.GetStats()
	if stats.TotalQueries != 1 {
		t.Errorf("Expected 1 total query, got %d", stats.TotalQueries)
	}
}

func TestQueryOptimizer_SlowQueryDetection(t *testing.T) {
	// 创建sqlmock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// 创建查询优化器
	logger := log.NewStdLogger(&testWriter{t: t})
	config := DefaultQueryOptimizerConfig()
	config.SlowQueryThreshold = 10 * time.Millisecond // 设置非常低的阈值以便测试
	qo := NewQueryOptimizer(db, config, logger)

	// 设置mock期望，模拟慢查询
	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "test")
	mock.ExpectQuery("SELECT \\* FROM users").
		WillDelayFor(20 * time.Millisecond). // 延迟20毫秒，超过阈值
		WillReturnRows(rows)

	// 执行查询
	ctx := context.Background()
	_, err = qo.ExecuteQuery(ctx, "SELECT * FROM users")
	if err != nil {
		t.Fatalf("ExecuteQuery failed: %v", err)
	}

	// 验证统计信息
	stats := qo.GetStats()
	if stats.TotalQueries != 1 {
		t.Errorf("Expected 1 total query, got %d", stats.TotalQueries)
	}
	if stats.SlowQueries != 1 {
		t.Errorf("Expected 1 slow query, got %d", stats.SlowQueries)
	}
}

func TestQueryOptimizer_GetStats(t *testing.T) {
	// 创建sqlmock
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// 创建查询优化器
	logger := log.NewStdLogger(&testWriter{t: t})
	config := DefaultQueryOptimizerConfig()
	qo := NewQueryOptimizer(db, config, logger)

	// 验证初始统计信息
	stats := qo.GetStats()
	if stats.TotalQueries != 0 {
		t.Errorf("Expected 0 total queries, got %d", stats.TotalQueries)
	}
	if stats.SlowQueries != 0 {
		t.Errorf("Expected 0 slow queries, got %d", stats.SlowQueries)
	}
	if stats.AverageDuration != 0 {
		t.Errorf("Expected 0 average duration, got %v", stats.AverageDuration)
	}
}

func TestQueryOptimizer_ResetStats(t *testing.T) {
	// 创建sqlmock
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// 创建查询优化器
	logger := log.NewStdLogger(&testWriter{t: t})
	config := DefaultQueryOptimizerConfig()
	qo := NewQueryOptimizer(db, config, logger)

	// 手动增加一些统计信息
	qo.stats.TotalQueries = 10
	qo.stats.SlowQueries = 2
	qo.stats.AverageDuration = 50 * time.Millisecond
	qo.stats.TotalDuration = 500 * time.Millisecond

	// 重置统计信息
	qo.ResetStats()

	// 验证统计信息已被重置
	stats := qo.GetStats()
	if stats.TotalQueries != 0 {
		t.Errorf("Expected 0 total queries after reset, got %d", stats.TotalQueries)
	}
	if stats.SlowQueries != 0 {
		t.Errorf("Expected 0 slow queries after reset, got %d", stats.SlowQueries)
	}
	if stats.AverageDuration != 0 {
		t.Errorf("Expected 0 average duration after reset, got %v", stats.AverageDuration)
	}
	if stats.TotalDuration != 0 {
		t.Errorf("Expected 0 total duration after reset, got %v", stats.TotalDuration)
	}
}

// testWriter 是一个用于测试的日志写入器
type testWriter struct {
	t *testing.T
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	w.t.Logf("LOG: %s", string(p))
	return len(p), nil
}
