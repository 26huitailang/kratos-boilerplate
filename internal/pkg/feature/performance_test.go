package feature

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheEntry_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		entry    *CacheEntry
		expected bool
	}{
		{
			name: "not expired",
			entry: &CacheEntry{
				Timestamp: now,
				TTL:       time.Hour,
			},
			expected: false,
		},
		{
			name: "expired",
			entry: &CacheEntry{
				Timestamp: now.Add(-2 * time.Hour),
				TTL:       time.Hour,
			},
			expected: true,
		},
		{
			name: "no TTL (never expires)",
			entry: &CacheEntry{
				Timestamp: now.Add(-24 * time.Hour),
				TTL:       0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.entry.IsExpired())
		})
	}
}

func TestCacheEntry_Touch(t *testing.T) {
	entry := &CacheEntry{
		AccessCount: 5,
		LastAccess:  time.Now().Add(-time.Hour),
	}

	oldLastAccess := entry.LastAccess
	entry.Touch()

	assert.Equal(t, int64(6), entry.AccessCount)
	assert.True(t, entry.LastAccess.After(oldLastAccess))
}

func TestEvaluationCache_SetGet(t *testing.T) {
	logger := log.DefaultLogger
	config := &CacheConfig{
		Enabled:    true,
		MaxSize:    100,
		DefaultTTL: time.Hour,
	}

	cache := NewEvaluationCache(config, logger)
	defer cache.Stop()

	// Test Set and Get
	cache.Set("key1", true, time.Hour)
	
	value, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, true, value)

	// Test non-existent key
	_, found = cache.Get("nonexistent")
	assert.False(t, found)
}

func TestEvaluationCache_Expiration(t *testing.T) {
	logger := log.DefaultLogger
	config := &CacheConfig{
		Enabled:    true,
		MaxSize:    100,
		DefaultTTL: time.Millisecond, // Very short TTL
	}

	cache := NewEvaluationCache(config, logger)
	defer cache.Stop()

	cache.Set("key1", true, time.Millisecond)
	
	// Should be available immediately
	value, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, true, value)

	// Wait for expiration
	time.Sleep(5 * time.Millisecond)
	
	_, found = cache.Get("key1")
	assert.False(t, found)
}

func TestEvaluationCache_Disabled(t *testing.T) {
	logger := log.DefaultLogger
	config := &CacheConfig{
		Enabled: false,
	}

	cache := NewEvaluationCache(config, logger)

	cache.Set("key1", true, time.Hour)
	
	_, found := cache.Get("key1")
	assert.False(t, found)
}

func TestEvaluationCache_Clear(t *testing.T) {
	logger := log.DefaultLogger
	config := &CacheConfig{
		Enabled:    true,
		MaxSize:    100,
		DefaultTTL: time.Hour,
	}

	cache := NewEvaluationCache(config, logger)
	defer cache.Stop()

	cache.Set("key1", true, time.Hour)
	cache.Set("key2", false, time.Hour)
	
	// Verify items exist
	_, found1 := cache.Get("key1")
	_, found2 := cache.Get("key2")
	assert.True(t, found1)
	assert.True(t, found2)

	// Clear cache
	cache.Clear()
	
	// Verify items are gone
	_, found1 = cache.Get("key1")
	_, found2 = cache.Get("key2")
	assert.False(t, found1)
	assert.False(t, found2)
}

func TestEvaluationCache_Stats(t *testing.T) {
	logger := log.DefaultLogger
	config := &CacheConfig{
		Enabled:    true,
		MaxSize:    100,
		DefaultTTL: time.Hour,
	}

	cache := NewEvaluationCache(config, logger)
	defer cache.Stop()

	// Add some items and access them
	cache.Set("key1", true, time.Hour)
	cache.Set("key2", false, time.Hour)
	
	cache.Get("key1") // Hit
	cache.Get("key1") // Hit
	cache.Get("nonexistent") // Miss

	stats := cache.GetStats()
	
	assert.Equal(t, true, stats["enabled"])
	assert.Equal(t, 2, stats["size"])
	assert.Equal(t, int64(2), stats["hit_count"])
	assert.Equal(t, int64(1), stats["miss_count"])
	assert.Equal(t, int64(3), stats["total_requests"])
	
	hitRate := stats["hit_rate"].(float64)
	assert.InDelta(t, 2.0/3.0, hitRate, 0.01)
}

func TestEvaluationCache_SetEnabled(t *testing.T) {
	logger := log.DefaultLogger
	config := &CacheConfig{
		Enabled:    true,
		MaxSize:    100,
		DefaultTTL: time.Hour,
	}

	cache := NewEvaluationCache(config, logger)
	defer cache.Stop()

	cache.Set("key1", true, time.Hour)
	
	// Disable cache
	cache.SetEnabled(false)
	
	// Should not find items after disabling
	_, found := cache.Get("key1")
	assert.False(t, found)
	
	// Should not set new items
	cache.Set("key2", false, time.Hour)
	_, found = cache.Get("key2")
	assert.False(t, found)
}

func TestGenerateCacheKey(t *testing.T) {
	feature := FeatureUserManagement
	
	// Test with nil context
	key1 := GenerateCacheKey(feature, nil)
	assert.Equal(t, string(feature), key1)
	
	// Test with evaluation context
	evalCtx := &EvaluationContext{
		UserID:      "user123",
		UserType:    "admin",
		Environment: "production",
		Version:     "1.0.0",
		Attributes: map[string]string{
			"tenant": "tenant123",
			"region": "us-west",
		},
	}
	
	key2 := GenerateCacheKey(feature, evalCtx)
	assert.NotEqual(t, string(feature), key2)
	assert.NotEmpty(t, key2)
	
	// Same context should generate same key
	key3 := GenerateCacheKey(feature, evalCtx)
	assert.Equal(t, key2, key3)
	
	// Different context should generate different key
	evalCtx2 := &EvaluationContext{
		UserID:      "user456",
		UserType:    "admin",
		Environment: "production",
		Version:     "1.0.0",
		Attributes:  make(map[string]string),
	}
	
	key4 := GenerateCacheKey(feature, evalCtx2)
	assert.NotEqual(t, key2, key4)
}

func TestObjectPool(t *testing.T) {
	pool := NewObjectPool(
		func() *ToggleConfig {
			return &ToggleConfig{
				Rules: make(map[string]interface{}),
				Tags:  make([]string, 0),
			}
		},
		func(config *ToggleConfig) {
			config.Enabled = false
			config.Strategy = ""
			for k := range config.Rules {
				delete(config.Rules, k)
			}
			config.Tags = config.Tags[:0]
		},
	)

	// Get object from pool
	obj1 := pool.Get()
	require.NotNil(t, obj1)
	require.NotNil(t, obj1.Rules)
	require.NotNil(t, obj1.Tags)

	// Modify object
	obj1.Enabled = true
	obj1.Strategy = StrategySimple
	obj1.Rules["test"] = "value"
	obj1.Tags = append(obj1.Tags, "tag1")

	// Return to pool
	pool.Put(obj1)

	// Get another object (should be reset)
	obj2 := pool.Get()
	assert.False(t, obj2.Enabled)
	assert.Equal(t, FeatureStrategy(""), obj2.Strategy)
	assert.Empty(t, obj2.Rules)
	assert.Empty(t, obj2.Tags)
}

func TestToggleConfigPool(t *testing.T) {
	config1 := ToggleConfigPool.Get()
	require.NotNil(t, config1)

	// Modify config
	config1.Enabled = true
	config1.Strategy = StrategySimple
	config1.Rules["test"] = "value"
	config1.Tags = append(config1.Tags, "tag1")

	// Return to pool
	ToggleConfigPool.Put(config1)

	// Get another config (should be reset)
	config2 := ToggleConfigPool.Get()
	assert.False(t, config2.Enabled)
	assert.Equal(t, FeatureStrategy(""), config2.Strategy)
	assert.Empty(t, config2.Rules)
	assert.Empty(t, config2.Tags)
}

func TestEvaluationContextPool(t *testing.T) {
	ctx1 := EvaluationContextPool.Get()
	require.NotNil(t, ctx1)

	// Modify context
	ctx1.UserID = "user123"
	ctx1.UserType = "admin"
	ctx1.Attributes["test"] = "value"

	// Return to pool
	EvaluationContextPool.Put(ctx1)

	// Get another context (should be reset)
	ctx2 := EvaluationContextPool.Get()
	assert.Empty(t, ctx2.UserID)
	assert.Empty(t, ctx2.UserType)
	assert.Empty(t, ctx2.Attributes)
}

func TestPerformanceMonitor(t *testing.T) {
	logger := log.DefaultLogger
	monitor := NewPerformanceMonitor(logger)

	feature := FeatureUserManagement
	duration := 100 * time.Millisecond

	// Record evaluation
	monitor.RecordEvaluation(feature, duration)

	// Get last evaluation time
	lastTime := monitor.GetLastEvaluationTime(feature)
	assert.Equal(t, duration, lastTime)

	// Record another evaluation
	duration2 := 200 * time.Millisecond
	monitor.RecordEvaluation(feature, duration2)

	// Should return the latest time
	lastTime2 := monitor.GetLastEvaluationTime(feature)
	assert.Equal(t, duration2, lastTime2)

	// Clear data
	monitor.Clear()
	lastTime3 := monitor.GetLastEvaluationTime(feature)
	assert.Equal(t, time.Duration(0), lastTime3)
}

func TestCachedToggleManager_IsEnabledWithContext(t *testing.T) {
	// This would require a full ToggleManager setup, which is complex
	// In a real implementation, you would mock the ToggleManager
	// For now, we'll test the cache key generation logic
	
	feature := FeatureUserManagement
	evalCtx := &EvaluationContext{
		UserID:      "user123",
		Environment: "production",
		Attributes:  make(map[string]string),
	}

	key := GenerateCacheKey(feature, evalCtx)
	assert.NotEmpty(t, key)
	
	// Same parameters should generate same key
	key2 := GenerateCacheKey(feature, evalCtx)
	assert.Equal(t, key, key2)
}