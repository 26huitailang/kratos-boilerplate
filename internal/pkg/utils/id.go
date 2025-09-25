package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

// IDGenerator ID生成器接口
type IDGenerator interface {
	// GenerateUUID 生成UUID
	GenerateUUID() string
	// GenerateShortID 生成短ID
	GenerateShortID() string
	// GenerateSnowflake 生成雪花ID
	GenerateSnowflake() int64
	// GenerateNanoID 生成纳秒时间戳ID
	GenerateNanoID() string
	// GenerateCustomID 生成自定义格式ID
	GenerateCustomID(prefix string, length int) string
}

// generator ID生成器实现
type generator struct {
	snowflake *SnowflakeGenerator
	mu        sync.Mutex
}

// NewIDGenerator 创建ID生成器
func NewIDGenerator(machineID int64) IDGenerator {
	return &generator{
		snowflake: NewSnowflakeGenerator(machineID),
	}
}

// GenerateUUID 生成UUID
func (g *generator) GenerateUUID() string {
	return uuid.New().String()
}

// GenerateShortID 生成短ID (8位)
func (g *generator) GenerateShortID() string {
	id := g.snowflake.Generate()
	return fmt.Sprintf("%08x", id&0xFFFFFFFF)
}

// GenerateSnowflake 生成雪花ID
func (g *generator) GenerateSnowflake() int64 {
	return g.snowflake.Generate()
}

// GenerateNanoID 生成纳秒时间戳ID
func (g *generator) GenerateNanoID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// GenerateCustomID 生成自定义格式ID
func (g *generator) GenerateCustomID(prefix string, length int) string {
	if length <= len(prefix) {
		length = len(prefix) + 8
	}
	
	// 生成随机字符串
	randomPart := g.generateRandomString(length - len(prefix))
	return prefix + randomPart
}

// generateRandomString 生成随机字符串
func (g *generator) generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}
	
	return string(result)
}

// SnowflakeGenerator 雪花算法ID生成器
type SnowflakeGenerator struct {
	mutex     sync.Mutex
	timestamp int64
	machineID int64
	sequence  int64
}

const (
	// 时间戳位数
	timestampBits = 41
	// 机器ID位数
	machineIDBits = 10
	// 序列号位数
	sequenceBits = 12
	
	// 最大值
	maxMachineID = -1 ^ (-1 << machineIDBits)
	maxSequence  = -1 ^ (-1 << sequenceBits)
	
	// 位移
	machineIDShift = sequenceBits
	timestampShift = sequenceBits + machineIDBits
	
	// 基准时间戳 (2021-01-01 00:00:00 UTC)
	epoch = 1609459200000
)

// NewSnowflakeGenerator 创建雪花算法生成器
func NewSnowflakeGenerator(machineID int64) *SnowflakeGenerator {
	if machineID < 0 || machineID > maxMachineID {
		panic(fmt.Sprintf("machine ID must be between 0 and %d", maxMachineID))
	}
	
	return &SnowflakeGenerator{
		machineID: machineID,
		sequence:  0,
		timestamp: 0,
	}
}

// Generate 生成ID
func (sf *SnowflakeGenerator) Generate() int64 {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()
	
	now := time.Now().UnixMilli()
	
	if now < sf.timestamp {
		panic("clock moved backwards")
	}
	
	if now == sf.timestamp {
		sf.sequence = (sf.sequence + 1) & maxSequence
		if sf.sequence == 0 {
			// 等待下一毫秒
			for now <= sf.timestamp {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		sf.sequence = 0
	}
	
	sf.timestamp = now
	
	return ((now - epoch) << timestampShift) |
		(sf.machineID << machineIDShift) |
		sf.sequence
}

// ParseSnowflake 解析雪花ID
func ParseSnowflake(id int64) (timestamp, machineID, sequence int64) {
	timestamp = (id >> timestampShift) + epoch
	machineID = (id >> machineIDShift) & maxMachineID
	sequence = id & maxSequence
	return
}

// RequestIDGenerator 请求ID生成器
type RequestIDGenerator struct {
	prefix string
	gen    IDGenerator
}

// NewRequestIDGenerator 创建请求ID生成器
func NewRequestIDGenerator(prefix string) *RequestIDGenerator {
	return &RequestIDGenerator{
		prefix: prefix,
		gen:    NewIDGenerator(1), // 使用固定机器ID
	}
}

// Generate 生成请求ID
func (r *RequestIDGenerator) Generate() string {
	return r.gen.GenerateCustomID(r.prefix, 32)
}

// TraceIDGenerator 链路追踪ID生成器
type TraceIDGenerator struct{}

// NewTraceIDGenerator 创建链路追踪ID生成器
func NewTraceIDGenerator() *TraceIDGenerator {
	return &TraceIDGenerator{}
}

// GenerateTraceID 生成TraceID
func (t *TraceIDGenerator) GenerateTraceID() string {
	// 生成32位十六进制字符串
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		// 如果随机数生成失败，使用时间戳作为后备
		now := time.Now()
		timestamp := now.UnixNano()
		return fmt.Sprintf("%032x", timestamp)
	}
	return hex.EncodeToString(bytes)
}

// GenerateSpanID 生成SpanID
func (t *TraceIDGenerator) GenerateSpanID() string {
	// 生成16位十六进制字符串
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		// 如果随机数生成失败，使用时间戳作为后备
		now := time.Now()
		timestamp := now.UnixNano()
		return fmt.Sprintf("%016x", timestamp&0xFFFFFFFFFFFFFFFF)
	}
	return hex.EncodeToString(bytes)
}

// SessionIDGenerator 会话ID生成器
type SessionIDGenerator struct {
	gen IDGenerator
}

// NewSessionIDGenerator 创建会话ID生成器
func NewSessionIDGenerator() *SessionIDGenerator {
	return &SessionIDGenerator{
		gen: NewIDGenerator(2), // 使用固定机器ID
	}
}

// Generate 生成会话ID
func (s *SessionIDGenerator) Generate() string {
	// 生成64位会话ID
	uuid := s.gen.GenerateUUID()
	hash := sha256.Sum256([]byte(uuid + strconv.FormatInt(time.Now().UnixNano(), 10)))
	return hex.EncodeToString(hash[:])[:64]
}

// OperationIDGenerator 操作ID生成器
type OperationIDGenerator struct {
	gen IDGenerator
}

// NewOperationIDGenerator 创建操作ID生成器
func NewOperationIDGenerator() *OperationIDGenerator {
	return &OperationIDGenerator{
		gen: NewIDGenerator(3), // 使用固定机器ID
	}
}

// Generate 生成操作ID
func (o *OperationIDGenerator) Generate(operation string) string {
	timestamp := time.Now().Unix()
	snowflake := o.gen.GenerateSnowflake()
	
	// 组合操作名称、时间戳和雪花ID
	combined := fmt.Sprintf("%s_%d_%d", operation, timestamp, snowflake)
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])[:32]
}

// IDManager ID管理器
type IDManager struct {
	generators map[string]IDGenerator
	mu         sync.RWMutex
}

// NewIDManager 创建ID管理器
func NewIDManager() *IDManager {
	return &IDManager{
		generators: make(map[string]IDGenerator),
	}
}

// RegisterGenerator 注册ID生成器
func (m *IDManager) RegisterGenerator(name string, generator IDGenerator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generators[name] = generator
}

// GetGenerator 获取ID生成器
func (m *IDManager) GetGenerator(name string) (IDGenerator, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	gen, exists := m.generators[name]
	return gen, exists
}

// GenerateID 生成指定类型的ID
func (m *IDManager) GenerateID(generatorName, idType string, params ...interface{}) (string, error) {
	gen, exists := m.GetGenerator(generatorName)
	if !exists {
		return "", fmt.Errorf("generator %s not found", generatorName)
	}
	
	switch idType {
	case "uuid":
		return gen.GenerateUUID(), nil
	case "short":
		return gen.GenerateShortID(), nil
	case "nano":
		return gen.GenerateNanoID(), nil
	case "custom":
		if len(params) >= 2 {
			if prefix, ok := params[0].(string); ok {
				if length, ok := params[1].(int); ok {
					return gen.GenerateCustomID(prefix, length), nil
				}
			}
		}
		return "", fmt.Errorf("invalid parameters for custom ID")
	case "snowflake":
		return strconv.FormatInt(gen.GenerateSnowflake(), 10), nil
	default:
		return "", fmt.Errorf("unknown ID type: %s", idType)
	}
}

// DefaultIDManager 默认ID管理器实例
var defaultIDManager *IDManager
var initOnce sync.Once

// GetDefaultIDManager 获取默认ID管理器
func GetDefaultIDManager() *IDManager {
	initOnce.Do(func() {
		defaultIDManager = NewIDManager()
		// 注册默认生成器
		defaultIDManager.RegisterGenerator("default", NewIDGenerator(1))
		defaultIDManager.RegisterGenerator("user", NewIDGenerator(10))
		defaultIDManager.RegisterGenerator("order", NewIDGenerator(20))
		defaultIDManager.RegisterGenerator("product", NewIDGenerator(30))
	})
	return defaultIDManager
}

// 便捷函数

// GenerateUUID 生成UUID
func GenerateUUID() string {
	return GetDefaultIDManager().generators["default"].GenerateUUID()
}

// GenerateShortID 生成短ID
func GenerateShortID() string {
	return GetDefaultIDManager().generators["default"].GenerateShortID()
}

// GenerateSnowflakeID 生成雪花ID
func GenerateSnowflakeID() int64 {
	return GetDefaultIDManager().generators["default"].GenerateSnowflake()
}

// GenerateNanoID 生成纳秒ID
func GenerateNanoID() string {
	return GetDefaultIDManager().generators["default"].GenerateNanoID()
}

// GenerateCustomID 生成自定义ID
func GenerateCustomID(prefix string, length int) string {
	return GetDefaultIDManager().generators["default"].GenerateCustomID(prefix, length)
}

// GenerateRequestID 生成请求ID
func GenerateRequestID() string {
	gen := NewRequestIDGenerator("req_")
	return gen.Generate()
}

// GenerateTraceID 生成链路追踪ID
func GenerateTraceID() string {
	gen := NewTraceIDGenerator()
	return gen.GenerateTraceID()
}

// GenerateSpanID 生成Span ID
func GenerateSpanID() string {
	gen := NewTraceIDGenerator()
	return gen.GenerateSpanID()
}

// GenerateSessionID 生成会话ID
func GenerateSessionID() string {
	gen := NewSessionIDGenerator()
	return gen.Generate()
}

// GenerateOperationID 生成操作ID
func GenerateOperationID(operation string) string {
	gen := NewOperationIDGenerator()
	return gen.Generate(operation)
}

// IDValidator ID验证器
type IDValidator struct{}

// NewIDValidator 创建ID验证器
func NewIDValidator() *IDValidator {
	return &IDValidator{}
}

// ValidateUUID 验证UUID格式
func (v *IDValidator) ValidateUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// ValidateSnowflake 验证雪花ID
func (v *IDValidator) ValidateSnowflake(id int64) bool {
	if id <= 0 {
		return false
	}
	
	timestamp, machineID, sequence := ParseSnowflake(id)
	
	// 验证时间戳是否合理（不能是未来时间）
	if timestamp > time.Now().UnixMilli() {
		return false
	}
	
	// 验证机器ID是否在有效范围内
	if machineID < 0 || machineID > maxMachineID {
		return false
	}
	
	// 验证序列号是否在有效范围内
	if sequence < 0 || sequence > maxSequence {
		return false
	}
	
	return true
}

// ValidateCustomID 验证自定义ID格式
func (v *IDValidator) ValidateCustomID(id, expectedPrefix string, expectedLength int) bool {
	if len(id) != expectedLength {
		return false
	}
	
	if expectedPrefix != "" && !startsWith(id, expectedPrefix) {
		return false
	}
	
	return true
}

// startsWith 检查字符串是否以指定前缀开头
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// IDMetrics ID指标收集器
type IDMetrics struct {
	generateCount map[string]int64
	mu           sync.RWMutex
}

// NewIDMetrics 创建ID指标收集器
func NewIDMetrics() *IDMetrics {
	return &IDMetrics{
		generateCount: make(map[string]int64),
	}
}

// IncrementGeneration 增加生成计数
func (m *IDMetrics) IncrementGeneration(idType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generateCount[idType]++
}

// GetGenerationCount 获取生成计数
func (m *IDMetrics) GetGenerationCount(idType string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.generateCount[idType]
}

// GetAllCounts 获取所有计数
func (m *IDMetrics) GetAllCounts() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]int64)
	for k, v := range m.generateCount {
		result[k] = v
	}
	return result
}

// Reset 重置计数
func (m *IDMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generateCount = make(map[string]int64)
}

// 全局指标收集器
var globalMetrics = NewIDMetrics()

// GetGlobalMetrics 获取全局指标收集器
func GetGlobalMetrics() *IDMetrics {
	return globalMetrics
}