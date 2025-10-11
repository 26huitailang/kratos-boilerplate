package performance

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// ConcurrencyController 并发控制器
type ConcurrencyController struct {
	config     *ConcurrencyConfig
	logger     log.Logger
	workerPool *WorkerPool
	semaphore  chan struct{}
	stats      *ConcurrencyStats
}

// ConcurrencyConfig 并发控制配置
type ConcurrencyConfig struct {
	// Worker池配置
	WorkerPool struct {
		Enabled     bool          `yaml:"enabled" json:"enabled"`
		MinWorkers  int           `yaml:"min_workers" json:"min_workers"`
		MaxWorkers  int           `yaml:"max_workers" json:"max_workers"`
		QueueSize   int           `yaml:"queue_size" json:"queue_size"`
		IdleTimeout time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	} `yaml:"worker_pool" json:"worker_pool"`

	// 信号量配置
	Semaphore struct {
		Enabled bool `yaml:"enabled" json:"enabled"`
		Size    int  `yaml:"size" json:"size"`
	} `yaml:"semaphore" json:"semaphore"`

	// 背压控制配置
	Backpressure struct {
		Enabled       bool          `yaml:"enabled" json:"enabled"`
		MaxQueueSize  int           `yaml:"max_queue_size" json:"max_queue_size"`
		MaxWaitTime   time.Duration `yaml:"max_wait_time" json:"max_wait_time"`
		RejectOnQueue bool          `yaml:"reject_on_queue" json:"reject_on_queue"`
	} `yaml:"backpressure" json:"backpressure"`

	// 监控配置
	Monitoring struct {
		Enabled       bool          `yaml:"enabled" json:"enabled"`
		CheckInterval time.Duration `yaml:"check_interval" json:"check_interval"`
	} `yaml:"monitoring" json:"monitoring"`
}

// DefaultConcurrencyConfig 默认并发控制配置
func DefaultConcurrencyConfig() *ConcurrencyConfig {
	config := &ConcurrencyConfig{}

	// Worker池默认配置
	config.WorkerPool.Enabled = true
	config.WorkerPool.MinWorkers = 10
	config.WorkerPool.MaxWorkers = 100
	config.WorkerPool.QueueSize = 1000
	config.WorkerPool.IdleTimeout = 5 * time.Minute

	// 信号量默认配置
	config.Semaphore.Enabled = true
	config.Semaphore.Size = 50

	// 背压控制默认配置
	config.Backpressure.Enabled = true
	config.Backpressure.MaxQueueSize = 1000
	config.Backpressure.MaxWaitTime = 30 * time.Second
	config.Backpressure.RejectOnQueue = true

	// 监控默认配置
	config.Monitoring.Enabled = true
	config.Monitoring.CheckInterval = 30 * time.Second

	return config
}

// ConcurrencyStats 并发统计信息
type ConcurrencyStats struct {
	// Worker池统计
	ActiveWorkers  int64 `json:"active_workers"`
	IdleWorkers    int64 `json:"idle_workers"`
	TotalWorkers   int64 `json:"total_workers"`
	QueueSize      int64 `json:"queue_size"`
	ProcessedTasks int64 `json:"processed_tasks"`

	// 信号量统计
	SemaphoreUsage int64 `json:"semaphore_usage"`
	SemaphoreSize  int64 `json:"semaphore_size"`

	// 背压统计
	RejectedRequests   int64 `json:"rejected_requests"`
	BackpressureEvents int64 `json:"backpressure_events"`

	// 错误统计
	Errors int64 `json:"errors"`

	mu sync.RWMutex
}

// NewConcurrencyController 创建并发控制器
func NewConcurrencyController(config *ConcurrencyConfig, logger log.Logger) *ConcurrencyController {
	if config == nil {
		config = DefaultConcurrencyConfig()
	}

	controller := &ConcurrencyController{
		config: config,
		logger: logger,
		stats:  &ConcurrencyStats{},
	}

	// 初始化Worker池
	if config.WorkerPool.Enabled {
		controller.workerPool = NewWorkerPool(
			config.WorkerPool.MinWorkers,
			config.WorkerPool.MaxWorkers,
			config.WorkerPool.QueueSize,
			config.WorkerPool.IdleTimeout,
			logger,
		)
		controller.workerPool.Start()
	}

	// 初始化信号量
	if config.Semaphore.Enabled && config.Semaphore.Size > 0 {
		controller.semaphore = make(chan struct{}, config.Semaphore.Size)
		controller.stats.SemaphoreSize = int64(config.Semaphore.Size)
	}

	return controller
}

// ExecuteWithWorkerPool 使用Worker池执行任务
func (cc *ConcurrencyController) ExecuteWithWorkerPool(ctx context.Context, task func() error) error {
	if cc.workerPool == nil {
		return fmt.Errorf("worker pool is not enabled")
	}

	// 检查背压
	if cc.config.Backpressure.Enabled {
		if err := cc.checkBackpressure(); err != nil {
			atomic.AddInt64(&cc.stats.RejectedRequests, 1)
			return err
		}
	}

	// 提交任务到Worker池
	err := cc.workerPool.Submit(ctx, task)
	if err != nil {
		atomic.AddInt64(&cc.stats.Errors, 1)
		return err
	}

	atomic.AddInt64(&cc.stats.ProcessedTasks, 1)
	return nil
}

// ExecuteWithSemaphore 使用信号量执行任务
func (cc *ConcurrencyController) ExecuteWithSemaphore(ctx context.Context, task func() error) error {
	if cc.semaphore == nil {
		return fmt.Errorf("semaphore is not enabled")
	}

	// 检查背压
	if cc.config.Backpressure.Enabled {
		if err := cc.checkBackpressure(); err != nil {
			atomic.AddInt64(&cc.stats.RejectedRequests, 1)
			return err
		}
	}

	// 获取信号量
	select {
	case cc.semaphore <- struct{}{}:
		// 更新信号量使用统计
		atomic.AddInt64(&cc.stats.SemaphoreUsage, 1)
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(cc.config.Backpressure.MaxWaitTime):
		atomic.AddInt64(&cc.stats.BackpressureEvents, 1)
		return fmt.Errorf("timeout waiting for semaphore")
	}

	// 执行任务
	defer func() {
		// 释放信号量
		<-cc.semaphore
		atomic.AddInt64(&cc.stats.SemaphoreUsage, -1)
	}()

	err := task()
	if err != nil {
		atomic.AddInt64(&cc.stats.Errors, 1)
		return err
	}

	atomic.AddInt64(&cc.stats.ProcessedTasks, 1)
	return nil
}

// ExecuteWithBackpressure 使用背压控制执行任务
func (cc *ConcurrencyController) ExecuteWithBackpressure(ctx context.Context, task func() error) error {
	// 检查背压
	if cc.config.Backpressure.Enabled {
		if err := cc.checkBackpressure(); err != nil {
			atomic.AddInt64(&cc.stats.RejectedRequests, 1)
			return err
		}
	}

	// 执行任务
	err := task()
	if err != nil {
		atomic.AddInt64(&cc.stats.Errors, 1)
		return err
	}

	atomic.AddInt64(&cc.stats.ProcessedTasks, 1)
	return nil
}

// checkBackpressure 检查背压
func (cc *ConcurrencyController) checkBackpressure() error {
	// 检查Worker池队列大小
	if cc.workerPool != nil {
		queueSize := cc.workerPool.GetQueueSize()
		if queueSize >= int64(cc.config.Backpressure.MaxQueueSize) {
			if cc.config.Backpressure.RejectOnQueue {
				atomic.AddInt64(&cc.stats.BackpressureEvents, 1)
				return fmt.Errorf("service temporarily unavailable due to high load")
			}
		}
		cc.stats.QueueSize = queueSize
	}

	// 检查信号量使用情况
	if cc.semaphore != nil {
		semUsage := atomic.LoadInt64(&cc.stats.SemaphoreUsage)
		semSize := atomic.LoadInt64(&cc.stats.SemaphoreSize)
		if semUsage >= semSize {
			if cc.config.Backpressure.RejectOnQueue {
				atomic.AddInt64(&cc.stats.BackpressureEvents, 1)
				return fmt.Errorf("service temporarily unavailable due to high load")
			}
		}
	}

	return nil
}

// GetStats 获取并发统计信息
func (cc *ConcurrencyController) GetStats() *ConcurrencyStats {
	cc.stats.mu.RLock()
	defer cc.stats.mu.RUnlock()

	// 更新Worker池统计信息
	if cc.workerPool != nil {
		poolStats := cc.workerPool.GetStats()
		cc.stats.ActiveWorkers = poolStats.ActiveWorkers
		cc.stats.IdleWorkers = poolStats.IdleWorkers
		cc.stats.TotalWorkers = poolStats.TotalWorkers
		cc.stats.QueueSize = poolStats.QueueSize
	}

	// 返回统计信息的副本
	stats := &ConcurrencyStats{
		ActiveWorkers:      cc.stats.ActiveWorkers,
		IdleWorkers:        cc.stats.IdleWorkers,
		TotalWorkers:       cc.stats.TotalWorkers,
		QueueSize:          cc.stats.QueueSize,
		ProcessedTasks:     cc.stats.ProcessedTasks,
		SemaphoreUsage:     cc.stats.SemaphoreUsage,
		SemaphoreSize:      cc.stats.SemaphoreSize,
		RejectedRequests:   cc.stats.RejectedRequests,
		BackpressureEvents: cc.stats.BackpressureEvents,
		Errors:             cc.stats.Errors,
	}

	return stats
}

// Close 关闭并发控制器
func (cc *ConcurrencyController) Close() error {
	// 关闭Worker池
	if cc.workerPool != nil {
		cc.workerPool.Stop()
	}

	// 重置统计信息
	cc.stats.mu.Lock()
	cc.stats.ActiveWorkers = 0
	cc.stats.IdleWorkers = 0
	cc.stats.TotalWorkers = 0
	cc.stats.QueueSize = 0
	cc.stats.ProcessedTasks = 0
	cc.stats.SemaphoreUsage = 0
	cc.stats.RejectedRequests = 0
	cc.stats.BackpressureEvents = 0
	cc.stats.Errors = 0
	cc.stats.mu.Unlock()

	return nil
}

// WorkerPool Worker池
type WorkerPool struct {
	minWorkers  int
	maxWorkers  int
	queueSize   int
	idleTimeout time.Duration
	logger      log.Logger
	taskQueue   chan *Task
	workers     []*Worker
	workerMutex sync.RWMutex
	stats       *WorkerPoolStats
	ctx         context.Context
	cancel      context.CancelFunc
}

// Task 任务
type Task struct {
	Ctx  context.Context
	Func func() error
}

// Worker 工作者
type Worker struct {
	id       int
	pool     *WorkerPool
	taskChan chan *Task
	stopChan chan struct{}
	active   int32
}

// WorkerPoolStats Worker池统计信息
type WorkerPoolStats struct {
	ActiveWorkers int64 `json:"active_workers"`
	IdleWorkers   int64 `json:"idle_workers"`
	TotalWorkers  int64 `json:"total_workers"`
	QueueSize     int64 `json:"queue_size"`
}

// NewWorkerPool 创建Worker池
func NewWorkerPool(minWorkers, maxWorkers, queueSize int, idleTimeout time.Duration, logger log.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		minWorkers:  minWorkers,
		maxWorkers:  maxWorkers,
		queueSize:   queueSize,
		idleTimeout: idleTimeout,
		logger:      logger,
		taskQueue:   make(chan *Task, queueSize),
		workers:     make([]*Worker, 0),
		stats:       &WorkerPoolStats{},
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 启动Worker池
func (wp *WorkerPool) Start() {
	// 创建最小数量的Worker
	for i := 0; i < wp.minWorkers; i++ {
		wp.createWorker()
	}

	// 启动监控协程
	go wp.monitor()
}

// Stop 停止Worker池
func (wp *WorkerPool) Stop() {
	wp.cancel()

	// 等待所有Worker停止
	wp.workerMutex.RLock()
	workers := make([]*Worker, len(wp.workers))
	copy(workers, wp.workers)
	wp.workerMutex.RUnlock()

	for _, worker := range workers {
		worker.Stop()
	}

	// 关闭任务队列
	close(wp.taskQueue)
}

// Submit 提交任务
func (wp *WorkerPool) Submit(ctx context.Context, taskFunc func() error) error {
	task := &Task{
		Ctx:  ctx,
		Func: taskFunc,
	}

	select {
	case wp.taskQueue <- task:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("task queue is full")
	}
}

// createWorker 创建Worker
func (wp *WorkerPool) createWorker() *Worker {
	wp.workerMutex.Lock()
	defer wp.workerMutex.Unlock()

	workerID := len(wp.workers)
	worker := &Worker{
		id:       workerID,
		pool:     wp,
		taskChan: make(chan *Task, 10),
		stopChan: make(chan struct{}),
	}

	wp.workers = append(wp.workers, worker)
	atomic.AddInt64(&wp.stats.TotalWorkers, 1)

	// 启动Worker
	go worker.Run()

	log.NewHelper(wp.logger).Debugf("Created worker %d", workerID)
	return worker
}

// removeWorker 移除Worker
func (wp *WorkerPool) removeWorker(worker *Worker) {
	wp.workerMutex.Lock()
	defer wp.workerMutex.Unlock()

	for i, w := range wp.workers {
		if w.id == worker.id {
			// 从切片中移除
			wp.workers = append(wp.workers[:i], wp.workers[i+1:]...)
			atomic.AddInt64(&wp.stats.TotalWorkers, -1)
			log.NewHelper(wp.logger).Debugf("Removed worker %d", worker.id)
			break
		}
	}
}

// monitor 监控Worker池
func (wp *WorkerPool) monitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wp.ctx.Done():
			return
		case <-ticker.C:
			wp.adjustWorkers()
		}
	}
}

// adjustWorkers 调整Worker数量
func (wp *WorkerPool) adjustWorkers() {
	queueLen := int64(len(wp.taskQueue))
	activeWorkers := atomic.LoadInt64(&wp.stats.ActiveWorkers)
	totalWorkers := atomic.LoadInt64(&wp.stats.TotalWorkers)

	// 如果队列中有任务且活跃Worker不足，增加Worker
	if queueLen > 0 && activeWorkers < int64(wp.maxWorkers) {
		// 计算需要增加的Worker数量
		neededWorkers := queueLen - activeWorkers
		if neededWorkers > int64(wp.maxWorkers)-totalWorkers {
			neededWorkers = int64(wp.maxWorkers) - totalWorkers
		}

		if neededWorkers > 0 {
			for i := int64(0); i < neededWorkers; i++ {
				wp.createWorker()
			}
		}
	}

	// 如果队列为空且Worker过多，减少Worker
	if queueLen == 0 && totalWorkers > int64(wp.minWorkers) {
		// 检查是否有空闲时间过长的Worker
		wp.workerMutex.RLock()
		workers := make([]*Worker, len(wp.workers))
		copy(workers, wp.workers)
		wp.workerMutex.RUnlock()

		for _, worker := range workers {
			// 如果Worker空闲且总数超过最小值，可以移除
			if atomic.LoadInt32(&worker.active) == 0 && totalWorkers > int64(wp.minWorkers) {
				worker.Stop()
				wp.removeWorker(worker)
				totalWorkers = atomic.LoadInt64(&wp.stats.TotalWorkers)
			}
		}
	}

	// 更新统计信息
	atomic.StoreInt64(&wp.stats.QueueSize, queueLen)
}

// GetStats 获取Worker池统计信息
func (wp *WorkerPool) GetStats() *WorkerPoolStats {
	return &WorkerPoolStats{
		ActiveWorkers: atomic.LoadInt64(&wp.stats.ActiveWorkers),
		IdleWorkers:   atomic.LoadInt64(&wp.stats.IdleWorkers),
		TotalWorkers:  atomic.LoadInt64(&wp.stats.TotalWorkers),
		QueueSize:     atomic.LoadInt64(&wp.stats.QueueSize),
	}
}

// GetQueueSize 获取队列大小
func (wp *WorkerPool) GetQueueSize() int64 {
	return int64(len(wp.taskQueue))
}

// Run 运行Worker
func (w *Worker) Run() {
	idleTimer := time.NewTimer(w.pool.idleTimeout)
	defer idleTimer.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case task := <-w.taskChan:
			// 标记为活跃
			atomic.StoreInt32(&w.active, 1)
			atomic.AddInt64(&w.pool.stats.ActiveWorkers, 1)
			atomic.AddInt64(&w.pool.stats.IdleWorkers, -1)

			// 执行任务
			if task.Func != nil {
				err := task.Func()
				if err != nil {
					log.NewHelper(w.pool.logger).Errorf("Worker %d task failed: %v", w.id, err)
				}
			}

			// 标记为空闲
			atomic.StoreInt32(&w.active, 0)
			atomic.AddInt64(&w.pool.stats.ActiveWorkers, -1)
			atomic.AddInt64(&w.pool.stats.IdleWorkers, 1)

			// 重置空闲计时器
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(w.pool.idleTimeout)
		case <-idleTimer.C:
			// 空闲超时，检查是否需要停止
			if atomic.LoadInt64(&w.pool.stats.TotalWorkers) > int64(w.pool.minWorkers) {
				w.Stop()
				w.pool.removeWorker(w)
				return
			}
			idleTimer.Reset(w.pool.idleTimeout)
		}
	}
}

// Stop 停止Worker
func (w *Worker) Stop() {
	close(w.stopChan)
}
