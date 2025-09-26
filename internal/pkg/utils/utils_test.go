package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIDGenerator(t *testing.T) {
	gen := NewIDGenerator(1)

	// 测试UUID生成
	uuid := gen.GenerateUUID()
	assert.NotEmpty(t, uuid)
	assert.Len(t, uuid, 36) // UUID标准长度

	// 测试短ID生成
	shortID := gen.GenerateShortID()
	assert.NotEmpty(t, shortID)
	assert.Len(t, shortID, 8)

	// 测试雪花ID生成
	snowflake := gen.GenerateSnowflake()
	assert.Greater(t, snowflake, int64(0))

	// 测试纳秒ID生成
	nanoID := gen.GenerateNanoID()
	assert.NotEmpty(t, nanoID)

	// 测试自定义ID生成
	customID := gen.GenerateCustomID("test_", 16)
	assert.NotEmpty(t, customID)
	assert.Len(t, customID, 16)
	assert.True(t, startsWith(customID, "test_"))
}

func TestSnowflakeGenerator(t *testing.T) {
	gen := NewSnowflakeGenerator(1)

	// 测试ID生成唯一性
	ids := make(map[int64]bool)
	for i := 0; i < 1000; i++ {
		id := gen.Generate()
		assert.False(t, ids[id], "Duplicate ID generated: %d", id)
		ids[id] = true
	}

	// 测试ID解析
	id := gen.Generate()
	timestamp, machineID, sequence := ParseSnowflake(id)
	assert.Greater(t, timestamp, int64(0))
	assert.Equal(t, int64(1), machineID)
	assert.GreaterOrEqual(t, sequence, int64(0))
}

func TestSnowflakeGeneratorPanic(t *testing.T) {
	// 测试无效机器ID
	assert.Panics(t, func() {
		NewSnowflakeGenerator(-1)
	})

	assert.Panics(t, func() {
		NewSnowflakeGenerator(1024) // 超过最大值
	})
}

func TestRequestIDGenerator(t *testing.T) {
	gen := NewRequestIDGenerator("req_")
	id := gen.Generate()

	assert.NotEmpty(t, id)
	assert.True(t, startsWith(id, "req_"))
	assert.Len(t, id, 32)
}

func TestTraceIDGenerator(t *testing.T) {
	gen := NewTraceIDGenerator()

	// 测试TraceID生成
	traceID := gen.GenerateTraceID()
	assert.NotEmpty(t, traceID)
	assert.Len(t, traceID, 32)

	// 测试SpanID生成
	spanID := gen.GenerateSpanID()
	assert.NotEmpty(t, spanID)
	assert.Len(t, spanID, 16)
}

func TestSessionIDGenerator(t *testing.T) {
	gen := NewSessionIDGenerator()
	id := gen.Generate()

	assert.NotEmpty(t, id)
	assert.Len(t, id, 64)
}

func TestOperationIDGenerator(t *testing.T) {
	gen := NewOperationIDGenerator()
	id := gen.Generate("test_operation")

	assert.NotEmpty(t, id)
	assert.Len(t, id, 32)
}

func TestIDManager(t *testing.T) {
	manager := NewIDManager()
	gen := NewIDGenerator(1)

	// 测试注册生成器
	manager.RegisterGenerator("test", gen)

	// 测试获取生成器
	retrievedGen, exists := manager.GetGenerator("test")
	assert.True(t, exists)
	assert.NotNil(t, retrievedGen)

	// 测试生成ID
	id, err := manager.GenerateID("test", "uuid")
	assert.NoError(t, err)
	assert.NotEmpty(t, id)

	// 测试生成短ID
	id, err = manager.GenerateID("test", "short")
	assert.NoError(t, err)
	assert.NotEmpty(t, id)

	// 测试生成自定义ID
	id, err = manager.GenerateID("test", "custom", "prefix_", 20)
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Len(t, id, 20)

	// 测试无效生成器
	_, err = manager.GenerateID("nonexistent", "uuid")
	assert.Error(t, err)

	// 测试无效ID类型
	_, err = manager.GenerateID("test", "invalid")
	assert.Error(t, err)
}

func TestDefaultIDManager(t *testing.T) {
	manager := GetDefaultIDManager()
	assert.NotNil(t, manager)

	// 验证默认生成器已注册
	_, exists := manager.GetGenerator("default")
	assert.True(t, exists)

	_, exists = manager.GetGenerator("user")
	assert.True(t, exists)
}

func TestConvenienceFunctions(t *testing.T) {
	// 测试便捷函数
	uuid := GenerateUUID()
	assert.NotEmpty(t, uuid)
	assert.Len(t, uuid, 36)

	shortID := GenerateShortID()
	assert.NotEmpty(t, shortID)
	assert.Len(t, shortID, 8)

	snowflakeID := GenerateSnowflakeID()
	assert.Greater(t, snowflakeID, int64(0))

	nanoID := GenerateNanoID()
	assert.NotEmpty(t, nanoID)

	customID := GenerateCustomID("test_", 15)
	assert.NotEmpty(t, customID)
	assert.Len(t, customID, 15)

	requestID := GenerateRequestID()
	assert.NotEmpty(t, requestID)
	assert.True(t, startsWith(requestID, "req_"))

	traceID := GenerateTraceID()
	assert.NotEmpty(t, traceID)
	assert.Len(t, traceID, 32)

	spanID := GenerateSpanID()
	assert.NotEmpty(t, spanID)
	assert.Len(t, spanID, 16)

	sessionID := GenerateSessionID()
	assert.NotEmpty(t, sessionID)
	assert.Len(t, sessionID, 64)

	operationID := GenerateOperationID("test")
	assert.NotEmpty(t, operationID)
	assert.Len(t, operationID, 32)
}

func TestIDValidator(t *testing.T) {
	validator := NewIDValidator()

	// 测试UUID验证
	validUUID := GenerateUUID()
	assert.True(t, validator.ValidateUUID(validUUID))
	assert.False(t, validator.ValidateUUID("invalid-uuid"))

	// 测试雪花ID验证
	gen := NewSnowflakeGenerator(1)
	validSnowflake := gen.Generate()
	assert.True(t, validator.ValidateSnowflake(validSnowflake))
	assert.False(t, validator.ValidateSnowflake(-1))
	assert.False(t, validator.ValidateSnowflake(0))

	// 测试自定义ID验证
	assert.True(t, validator.ValidateCustomID("test_12345", "test_", 10))
	assert.False(t, validator.ValidateCustomID("invalid", "test_", 10))
	assert.False(t, validator.ValidateCustomID("test_123", "test_", 10))
}

func TestIDMetrics(t *testing.T) {
	metrics := NewIDMetrics()

	// 初始状态
	assert.Equal(t, int64(0), metrics.GetGenerationCount("uuid"))

	// 增加计数
	metrics.IncrementGeneration("uuid")
	metrics.IncrementGeneration("uuid")
	metrics.IncrementGeneration("shortid")

	assert.Equal(t, int64(2), metrics.GetGenerationCount("uuid"))
	assert.Equal(t, int64(1), metrics.GetGenerationCount("shortid"))

	// 获取所有计数
	allCounts := metrics.GetAllCounts()
	assert.Equal(t, int64(2), allCounts["uuid"])
	assert.Equal(t, int64(1), allCounts["shortid"])

	// 重置计数
	metrics.Reset()
	assert.Equal(t, int64(0), metrics.GetGenerationCount("uuid"))
}

func TestTimeUtils(t *testing.T) {
	timeUtil := NewTimeUtils()
	now := time.Now()

	// 测试ISO8601格式化
	iso8601 := timeUtil.FormatISO8601(now)
	assert.NotEmpty(t, iso8601)

	// 测试ISO8601解析
	parsed, err := timeUtil.ParseISO8601(iso8601)
	assert.NoError(t, err)
	assert.Equal(t, now.Unix(), parsed.Unix())

	// 测试时间戳转换
	timestamp := timeUtil.FormatTimestamp(now)
	fromTimestamp := timeUtil.ParseTimestamp(timestamp)
	assert.Equal(t, now.Unix(), fromTimestamp.Unix())

	// 测试毫秒时间戳转换
	milliTimestamp := timeUtil.FormatMilliTimestamp(now)
	fromMilliTimestamp := timeUtil.ParseMilliTimestamp(milliTimestamp)
	assert.Equal(t, now.UnixMilli(), fromMilliTimestamp.UnixMilli())

	// 测试时区转换
	utcTime, err := timeUtil.ConvertTimezone(now, "UTC")
	assert.NoError(t, err)
	assert.NotNil(t, utcTime)

	// 测试时间计算
	future := timeUtil.AddDuration(now, time.Hour)
	duration := timeUtil.GetDuration(now, future)
	assert.Equal(t, time.Hour, duration)

	// 测试一天的开始和结束
	startOfDay := timeUtil.StartOfDay(now)
	endOfDay := timeUtil.EndOfDay(now)
	assert.Equal(t, 0, startOfDay.Hour())
	assert.Equal(t, 23, endOfDay.Hour())

	// 测试闰年判断
	assert.True(t, timeUtil.IsLeapYear(2020))
	assert.False(t, timeUtil.IsLeapYear(2021))

	// 测试周末判断
	saturday := time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC) // 2023-01-07是周六
	assert.True(t, timeUtil.IsWeekend(saturday))
	assert.False(t, timeUtil.IsWorkday(saturday))

	monday := time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC) // 2023-01-09是周一
	assert.False(t, timeUtil.IsWeekend(monday))
	assert.True(t, timeUtil.IsWorkday(monday))
}

func TestValidationUtils(t *testing.T) {
	validator := NewValidationUtils()

	// 测试邮箱验证
	assert.True(t, validator.IsEmail("test@example.com"))
	assert.False(t, validator.IsEmail("invalid-email"))

	// 测试手机号验证
	assert.True(t, validator.IsPhone("13812345678"))
	assert.False(t, validator.IsPhone("12345678901"))

	// 测试身份证验证
	assert.True(t, validator.IsIDCard("11010519491231002X"))
	assert.False(t, validator.IsIDCard("123456789012345678"))

	// 测试URL验证
	assert.True(t, validator.IsURL("https://example.com"))
	assert.False(t, validator.IsURL("invalid-url"))

	// 测试IP验证
	assert.True(t, validator.IsIP("192.168.1.1"))
	assert.False(t, validator.IsIP("256.256.256.256"))

	// 测试端口验证
	assert.True(t, validator.IsPort("8080"))
	assert.False(t, validator.IsPort("65536"))

	// 测试强密码验证
	assert.True(t, validator.IsStrongPassword("Password123!"))
	assert.False(t, validator.IsStrongPassword("weak"))
}

func TestStringUtils(t *testing.T) {
	stringUtil := NewStringUtils()

	// 测试空字符串判断
	assert.True(t, stringUtil.IsEmpty(""))
	assert.True(t, stringUtil.IsEmpty("   "))
	assert.False(t, stringUtil.IsEmpty("hello"))

	// 测试默认值
	assert.Equal(t, "default", stringUtil.DefaultIfEmpty("", "default"))
	assert.Equal(t, "hello", stringUtil.DefaultIfEmpty("hello", "default"))

	// 测试截断
	assert.Equal(t, "hello...", stringUtil.Truncate("hello world", 5))
	assert.Equal(t, "hi", stringUtil.Truncate("hi", 5))

	// 测试脱敏
	assert.Equal(t, "te**@example.com", stringUtil.MaskEmail("test@example.com"))
	assert.Equal(t, "138****5678", stringUtil.MaskPhone("13812345678"))
	assert.Equal(t, "110105********002X", stringUtil.MaskIDCard("11010519491231002X"))
	assert.Equal(t, "he***lo", stringUtil.MaskString("hello", 2, 2))

	// 测试命名转换
	assert.Equal(t, "helloWorld", stringUtil.ToCamelCase("hello_world"))
	assert.Equal(t, "HelloWorld", stringUtil.ToPascalCase("hello_world"))
	assert.Equal(t, "hello_world", stringUtil.ToSnakeCase("HelloWorld"))
	assert.Equal(t, "hello-world", stringUtil.ToKebabCase("HelloWorld"))

	// 测试反转
	assert.Equal(t, "olleh", stringUtil.Reverse("hello"))

	// 测试包含
	assert.True(t, stringUtil.Contains("Hello World", "WORLD"))
	assert.True(t, stringUtil.ContainsAny("hello", []string{"hi", "hello"}))

	// 测试移除
	assert.Equal(t, "helloworld", stringUtil.RemoveSpaces("hello world"))
	assert.Equal(t, "hello world", stringUtil.RemoveSpecialChars("hello@#$ world!"))
}

func TestNetworkUtils(t *testing.T) {
	networkUtil := NewNetworkUtils()

	// 测试域名提取
	assert.Equal(t, "example.com", networkUtil.ExtractDomain("https://example.com/path"))
	assert.Equal(t, "example.com", networkUtil.ExtractDomain("http://example.com:8080/path"))

	// 测试端口提取
	assert.Equal(t, "8080", networkUtil.ExtractPort("http://example.com:8080/path"))
	assert.Equal(t, "", networkUtil.ExtractPort("http://example.com/path"))

	// 测试私有IP判断
	assert.True(t, networkUtil.IsPrivateIP("192.168.1.1"))
	assert.True(t, networkUtil.IsPrivateIP("10.0.0.1"))
	assert.True(t, networkUtil.IsPrivateIP("172.16.0.1"))
	assert.True(t, networkUtil.IsPrivateIP("127.0.0.1"))
	assert.False(t, networkUtil.IsPrivateIP("8.8.8.8"))

	// 测试公网IP判断
	assert.True(t, networkUtil.IsPublicIP("8.8.8.8"))
	assert.False(t, networkUtil.IsPublicIP("192.168.1.1"))
}

func TestConvenienceUtilFunctions(t *testing.T) {
	// 测试便捷工具函数
	now := time.Now()
	formatted := FormatTime(now)
	assert.NotEmpty(t, formatted)

	parsed, err := ParseTime(formatted)
	assert.NoError(t, err)
	assert.Equal(t, now.Unix(), parsed.Unix())

	assert.True(t, IsValidEmail("test@example.com"))
	assert.False(t, IsValidEmail("invalid"))

	assert.True(t, IsValidPhone("13812345678"))
	assert.False(t, IsValidPhone("invalid"))

	assert.Equal(t, "te**@example.com", MaskSensitiveInfo("email", "test@example.com"))
	assert.Equal(t, "138****5678", MaskSensitiveInfo("phone", "13812345678"))
	assert.Equal(t, "11****02X", MaskSensitiveInfo("idcard", "11010519491231002X"))
	assert.Equal(t, "te**et", MaskSensitiveInfo("unknown", "test"))
}

// BenchmarkIDGeneration 性能基准测试
func BenchmarkIDGeneration(b *testing.B) {
	gen := NewIDGenerator(1)

	b.Run("UUID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			gen.GenerateUUID()
		}
	})

	b.Run("Snowflake", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			gen.GenerateSnowflake()
		}
	})

	b.Run("ShortID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			gen.GenerateShortID()
		}
	})

	b.Run("NanoID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			gen.GenerateNanoID()
		}
	})
}

func BenchmarkStringMasking(b *testing.B) {
	stringUtil := NewStringUtils()

	b.Run("MaskEmail", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			stringUtil.MaskEmail("test@example.com")
		}
	})

	b.Run("MaskPhone", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			stringUtil.MaskPhone("13812345678")
		}
	})

	b.Run("MaskIDCard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			stringUtil.MaskIDCard("11010519491231002X")
		}
	})
}

func BenchmarkValidation(b *testing.B) {
	validator := NewValidationUtils()

	b.Run("ValidateEmail", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validator.IsEmail("test@example.com")
		}
	})

	b.Run("ValidatePhone", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validator.IsPhone("13812345678")
		}
	})

	b.Run("ValidateIDCard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validator.IsIDCard("11010519491231002X")
		}
	})
}

// TestConcurrency 并发测试
func TestConcurrency(t *testing.T) {
	gen := NewSnowflakeGenerator(1)
	idChan := make(chan int64, 1000)

	// 启动多个goroutine并发生成ID
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				id := gen.Generate()
				idChan <- id
			}
		}()
	}

	// 收集所有ID并检查唯一性
	ids := make(map[int64]bool)
	for i := 0; i < 1000; i++ {
		id := <-idChan
		assert.False(t, ids[id], "Duplicate ID found in concurrent generation: %d", id)
		ids[id] = true
	}

	assert.Len(t, ids, 1000, "Should generate 1000 unique IDs")
}
