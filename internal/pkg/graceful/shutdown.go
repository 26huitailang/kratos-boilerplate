package graceful

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// ShutdownFunc defines a function to be called during shutdown
type ShutdownFunc func(ctx context.Context) error

// GracefulShutdown manages graceful shutdown of the application
type GracefulShutdown struct {
	logger    *log.Helper
	timeout   time.Duration
	callbacks []ShutdownFunc
	mu        sync.RWMutex
}

// NewGracefulShutdown creates a new graceful shutdown manager
func NewGracefulShutdown(timeout time.Duration, logger log.Logger) *GracefulShutdown {
	return &GracefulShutdown{
		logger:    log.NewHelper(log.With(logger, "component", "graceful_shutdown")),
		timeout:   timeout,
		callbacks: make([]ShutdownFunc, 0),
	}
}

// AddCallback adds a shutdown callback function
func (gs *GracefulShutdown) AddCallback(fn ShutdownFunc) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.callbacks = append(gs.callbacks, fn)
}

// Wait waits for shutdown signals and executes graceful shutdown
func (gs *GracefulShutdown) Wait() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	gs.logger.Info("Received shutdown signal, starting graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), gs.timeout)
	defer cancel()

	gs.executeShutdown(ctx)
}

// Shutdown executes all registered shutdown callbacks
func (gs *GracefulShutdown) executeShutdown(ctx context.Context) {
	gs.mu.RLock()
	callbacks := make([]ShutdownFunc, len(gs.callbacks))
	copy(callbacks, gs.callbacks)
	gs.mu.RUnlock()

	var wg sync.WaitGroup
	errorChan := make(chan error, len(callbacks))

	// Execute all callbacks concurrently
	for i, callback := range callbacks {
		wg.Add(1)
		go func(index int, fn ShutdownFunc) {
			defer wg.Done()

			gs.logger.Infof("Executing shutdown callback %d", index+1)

			if err := fn(ctx); err != nil {
				gs.logger.Errorf("Shutdown callback %d failed: %v", index+1, err)
				errorChan <- err
			} else {
				gs.logger.Infof("Shutdown callback %d completed successfully", index+1)
			}
		}(i, callback)
	}

	// Wait for all callbacks to complete or context to timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		gs.logger.Info("All shutdown callbacks completed")
	case <-ctx.Done():
		gs.logger.Warn("Shutdown timeout reached, some callbacks may not have completed")
	}

	close(errorChan)

	// Log any errors that occurred
	errorCount := 0
	for err := range errorChan {
		if err != nil {
			errorCount++
		}
	}

	if errorCount > 0 {
		gs.logger.Errorf("Graceful shutdown completed with %d errors", errorCount)
	} else {
		gs.logger.Info("Graceful shutdown completed successfully")
	}
}

// DefaultGracefulShutdown creates a graceful shutdown manager with default settings
func DefaultGracefulShutdown(logger log.Logger) *GracefulShutdown {
	return NewGracefulShutdown(30*time.Second, logger)
}
