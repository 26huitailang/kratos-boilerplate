package performance

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

func TestConcurrencyController_ExecuteWithWorkerPool(t *testing.T) {
	// 创建并发控制器
	logger := log.NewStdLogger(&testWriter{t: t})
	config := DefaultConcurrencyConfig()
	config.WorkerPool.MinWorkers = 2
	config.WorkerPool.MaxWorkers = 5
	config.WorkerPool.QueueSize = 10

	controller := NewConcurrencyController(config, logger)
	defer controller.Close()

	// 执行单个任务进行测试
	err := controller.ExecuteWithWorkerPool(context.Background(), func() error {
		// 模拟任务执行时间
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("ExecuteWithWorkerPool failed: %v", err)
	}

	// 等待任务完成
	time.Sleep(100 * time.Millisecond)

	// 验证统计信息
	stats := controller.GetStats()
	if stats.ProcessedTasks < 1 {
		t.Errorf("Expected at least 1 processed task, got %d", stats.ProcessedTasks)
	}
}

func TestConcurrencyController_ExecuteWithSemaphore(t *testing.T) {
	// 创建并发控制器
	logger := log.NewStdLogger(&testWriter{t: t})
	config := DefaultConcurrencyConfig()
	config.Semaphore.Size = 3
	config.Backpressure.MaxQueueSize = 10

	controller := NewConcurrencyController(config, logger)
	defer controller.Close()

	// 执行单个任务进行测试
	err := controller.ExecuteWithSemaphore(context.Background(), func() error {
		// 模拟任务执行时间
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("ExecuteWithSemaphore failed: %v", err)
	}

	// 等待任务完成
	time.Sleep(100 * time.Millisecond)

	// 验证统计信息
	stats := controller.GetStats()
	if stats.ProcessedTasks < 1 {
		t.Errorf("Expected at least 1 processed task, got %d", stats.ProcessedTasks)
	}
}

func TestConcurrencyController_ExecuteWithBackpressure(t *testing.T) {
	// 创建并发控制器
	logger := log.NewStdLogger(&testWriter{t: t})
	config := DefaultConcurrencyConfig()
	config.Backpressure.Enabled = true
	config.Backpressure.MaxQueueSize = 2
	config.Backpressure.RejectOnQueue = true

	controller := NewConcurrencyController(config, logger)
	defer controller.Close()

	// 执行单个任务进行测试
	err := controller.ExecuteWithBackpressure(context.Background(), func() error {
		// 模拟任务执行时间
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("ExecuteWithBackpressure failed: %v", err)
	}

	// 等待任务完成
	time.Sleep(100 * time.Millisecond)

	// 验证统计信息
	stats := controller.GetStats()
	if stats.ProcessedTasks < 1 {
		t.Errorf("Expected at least 1 processed task, got %d", stats.ProcessedTasks)
	}
}

func TestConcurrencyController_GetStats(t *testing.T) {
	// 创建并发控制器
	logger := log.NewStdLogger(&testWriter{t: t})
	config := DefaultConcurrencyConfig()
	controller := NewConcurrencyController(config, logger)
	defer controller.Close()

	// 验证初始统计信息
	stats := controller.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	// 检查各个字段是否存在且为0
	if stats.ActiveWorkers != 0 {
		t.Errorf("Expected 0 active workers, got %d", stats.ActiveWorkers)
	}
	if stats.ProcessedTasks != 0 {
		t.Errorf("Expected 0 processed tasks, got %d", stats.ProcessedTasks)
	}
	if stats.RejectedRequests != 0 {
		t.Errorf("Expected 0 rejected requests, got %d", stats.RejectedRequests)
	}
}
