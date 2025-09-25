package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Database 数据库接口
type Database interface {
	// GetDB 获取数据库连接
	GetDB() *gorm.DB
	// GetReadDB 获取只读数据库连接
	GetReadDB() *gorm.DB
	// GetWriteDB 获取写数据库连接
	GetWriteDB() *gorm.DB
	// Health 健康检查
	Health(ctx context.Context) error
	// Close 关闭连接
	Close() error
	// Migrate 数据库迁移
	Migrate(models ...interface{}) error
	// Transaction 事务执行
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	// Stats 获取连接池统计信息
	Stats() sql.DBStats
}

// Config 数据库配置
type Config struct {
	// 主数据库配置
	Master MasterConfig `yaml:"master" json:"master"`
	// 从数据库配置
	Slaves []SlaveConfig `yaml:"slaves" json:"slaves"`
	// 连接池配置
	Pool PoolConfig `yaml:"pool" json:"pool"`
	// 日志配置
	Log LogConfig `yaml:"log" json:"log"`
	// 其他配置
	Options OptionsConfig `yaml:"options" json:"options"`
}

// MasterConfig 主数据库配置
type MasterConfig struct {
	Driver string `yaml:"driver" json:"driver" validate:"required,oneof=postgres mysql sqlite sqlserver"`
	DSN    string `yaml:"dsn" json:"dsn" validate:"required"`
}

// SlaveConfig 从数据库配置
type SlaveConfig struct {
	Driver string `yaml:"driver" json:"driver" validate:"required,oneof=postgres mysql sqlite sqlserver"`
	DSN    string `yaml:"dsn" json:"dsn" validate:"required"`
	Weight int    `yaml:"weight" json:"weight"` // 权重
}

// PoolConfig 连接池配置
type PoolConfig struct {
	MaxOpenConns    int           `yaml:"max_open_conns" json:"max_open_conns"`       // 最大开放连接数
	MaxIdleConns    int           `yaml:"max_idle_conns" json:"max_idle_conns"`       // 最大空闲连接数
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"` // 连接最大生命周期
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" json:"conn_max_idle_time"` // 连接最大空闲时间
}

// LogConfig 日志配置
type LogConfig struct {
	Level                     logger.LogLevel `yaml:"level" json:"level"`                           // 日志级别
	SlowThreshold             time.Duration   `yaml:"slow_threshold" json:"slow_threshold"`         // 慢查询阈值
	Colorful                  bool            `yaml:"colorful" json:"colorful"`                     // 是否彩色输出
	IgnoreRecordNotFoundError bool            `yaml:"ignore_not_found" json:"ignore_not_found"`     // 忽略记录未找到错误
	LogSQL                    bool            `yaml:"log_sql" json:"log_sql"`                       // 是否记录SQL
}

// OptionsConfig 其他配置
type OptionsConfig struct {
	TablePrefix   string `yaml:"table_prefix" json:"table_prefix"`     // 表前缀
	SingularTable bool   `yaml:"singular_table" json:"singular_table"` // 使用单数表名
	AutoMigrate   bool   `yaml:"auto_migrate" json:"auto_migrate"`     // 自动迁移
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Master: MasterConfig{
			Driver: "postgres",
			DSN:    "postgresql://postgres:postgres@localhost:5432/test?sslmode=disable",
		},
		Pool: PoolConfig{
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
		},
		Log: LogConfig{
			Level:                     logger.Info,
			SlowThreshold:             200 * time.Millisecond,
			Colorful:                  false,
			IgnoreRecordNotFoundError: true,
			LogSQL:                    true,
		},
		Options: OptionsConfig{
			TablePrefix:   "",
			SingularTable: false,
			AutoMigrate:   false,
		},
	}
}

// database 数据库实现
type database struct {
	masterDB  *gorm.DB
	slavesDB  []*gorm.DB
	config    *Config
	logger    log.Logger
	balancer  *LoadBalancer
}

// NewDatabase 创建数据库实例
func NewDatabase(config *Config, logger log.Logger) (Database, error) {
	if config == nil {
		config = DefaultConfig()
	}
	
	db := &database{
		config: config,
		logger: logger,
	}
	
	// 初始化主数据库
	if err := db.initMaster(); err != nil {
		return nil, fmt.Errorf("failed to initialize master database: %w", err)
	}
	
	// 初始化从数据库
	if err := db.initSlaves(); err != nil {
		return nil, fmt.Errorf("failed to initialize slave databases: %w", err)
	}
	
	// 初始化负载均衡器
	db.initLoadBalancer()
	
	return db, nil
}

// initMaster 初始化主数据库
func (d *database) initMaster() error {
	gormConfig := d.buildGormConfig()
	
	// 根据驱动类型创建数据库连接
	dialector, err := d.createDialector(d.config.Master.Driver, d.config.Master.DSN)
	if err != nil {
		return fmt.Errorf("failed to create master dialector: %w", err)
	}
	
	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to master database: %w", err)
	}
	
	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB from master: %w", err)
	}
	
	d.configureConnectionPool(sqlDB, &d.config.Pool)
	d.masterDB = db
	
	return nil
}

// initSlaves 初始化从数据库
func (d *database) initSlaves() error {
	if len(d.config.Slaves) == 0 {
		return nil
	}
	
	gormConfig := d.buildGormConfig()
	d.slavesDB = make([]*gorm.DB, 0, len(d.config.Slaves))
	
	for i, slaveConfig := range d.config.Slaves {
		dialector, err := d.createDialector(slaveConfig.Driver, slaveConfig.DSN)
		if err != nil {
			return fmt.Errorf("failed to create slave[%d] dialector: %w", i, err)
		}
		
		db, err := gorm.Open(dialector, gormConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to slave[%d] database: %w", i, err)
		}
		
		// 配置连接池
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("failed to get underlying sql.DB from slave[%d]: %w", i, err)
		}
		
		d.configureConnectionPool(sqlDB, &d.config.Pool)
		d.slavesDB = append(d.slavesDB, db)
	}
	
	return nil
}

// initLoadBalancer 初始化负载均衡器
func (d *database) initLoadBalancer() {
	if len(d.slavesDB) == 0 {
		return
	}
	
	var nodes []LoadBalancerNode
	for i, slaveDB := range d.slavesDB {
		weight := 1
		if i < len(d.config.Slaves) {
			weight = d.config.Slaves[i].Weight
		}
		if weight <= 0 {
			weight = 1
		}
		
		nodes = append(nodes, LoadBalancerNode{
			ID:     fmt.Sprintf("slave-%d", i),
			Weight: weight,
			DB:     slaveDB,
		})
	}
	
	d.balancer = NewLoadBalancer(nodes)
}

// createDialector 创建数据库驱动
func (d *database) createDialector(driver, dsn string) (gorm.Dialector, error) {
	switch driver {
	case "postgres":
		return postgres.Open(dsn), nil
	case "mysql":
		return mysql.Open(dsn), nil
	case "sqlite":
		return sqlite.Open(dsn), nil
	case "sqlserver":
		return sqlserver.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}

// buildGormConfig 构建GORM配置
func (d *database) buildGormConfig() *gorm.Config {
	config := &gorm.Config{
		Logger: d.buildLogger(),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   d.config.Options.TablePrefix,
			SingularTable: d.config.Options.SingularTable,
		},
	}
	
	return config
}

// buildLogger 构建GORM日志器
func (d *database) buildLogger() logger.Interface {
	return NewGormLogger(d.logger, d.config.Log)
}

// configureConnectionPool 配置连接池
func (d *database) configureConnectionPool(sqlDB *sql.DB, poolConfig *PoolConfig) {
	if poolConfig.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(poolConfig.MaxOpenConns)
	}
	if poolConfig.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(poolConfig.MaxIdleConns)
	}
	if poolConfig.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
	}
	if poolConfig.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(poolConfig.ConnMaxIdleTime)
	}
}

// GetDB 获取数据库连接（默认使用主库）
func (d *database) GetDB() *gorm.DB {
	return d.masterDB
}

// GetReadDB 获取只读数据库连接
func (d *database) GetReadDB() *gorm.DB {
	if d.balancer != nil {
		if node := d.balancer.Next(); node != nil {
			return node.DB
		}
	}
	// 如果没有从库，使用主库
	return d.masterDB
}

// GetWriteDB 获取写数据库连接
func (d *database) GetWriteDB() *gorm.DB {
	return d.masterDB
}

// Health 健康检查
func (d *database) Health(ctx context.Context) error {
	// 检查主库
	if err := d.healthCheck(ctx, d.masterDB, "master"); err != nil {
		return err
	}
	
	// 检查从库
	for i, slaveDB := range d.slavesDB {
		if err := d.healthCheck(ctx, slaveDB, fmt.Sprintf("slave-%d", i)); err != nil {
			d.logger.Log(log.LevelWarn, "msg", "slave database health check failed", "slave", i, "error", err)
			// 从库健康检查失败不影响整体健康状态
		}
	}
	
	return nil
}

// healthCheck 单个数据库健康检查
func (d *database) healthCheck(ctx context.Context, db *gorm.DB, name string) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("%s database: failed to get sql.DB: %w", name, err)
	}
	
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("%s database ping failed: %w", name, err)
	}
	
	return nil
}

// Close 关闭连接
func (d *database) Close() error {
	var errors []error
	
	// 关闭主库
	if d.masterDB != nil {
		if sqlDB, err := d.masterDB.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				errors = append(errors, fmt.Errorf("failed to close master database: %w", err))
			}
		}
	}
	
	// 关闭从库
	for i, slaveDB := range d.slavesDB {
		if sqlDB, err := slaveDB.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				errors = append(errors, fmt.Errorf("failed to close slave[%d] database: %w", i, err))
			}
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("database close errors: %v", errors)
	}
	
	return nil
}

// Migrate 数据库迁移
func (d *database) Migrate(models ...interface{}) error {
	if len(models) == 0 {
		return nil
	}
	
	for _, model := range models {
		if err := d.masterDB.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate model %T: %w", model, err)
		}
	}
	
	d.logger.Log(log.LevelInfo, "msg", "database migration completed", "models", len(models))
	return nil
}

// Transaction 事务执行
func (d *database) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return d.masterDB.WithContext(ctx).Transaction(fn)
}

// Stats 获取连接池统计信息
func (d *database) Stats() sql.DBStats {
	if sqlDB, err := d.masterDB.DB(); err == nil {
		return sqlDB.Stats()
	}
	return sql.DBStats{}
}

// LoadBalancerNode 负载均衡节点
type LoadBalancerNode struct {
	ID     string   // 节点ID
	Weight int      // 权重
	DB     *gorm.DB // 数据库连接
}

// LoadBalancer 负载均衡器
type LoadBalancer struct {
	nodes   []LoadBalancerNode
	current int
	total   int
}

// NewLoadBalancer 创建负载均衡器
func NewLoadBalancer(nodes []LoadBalancerNode) *LoadBalancer {
	total := 0
	for _, node := range nodes {
		total += node.Weight
	}
	
	return &LoadBalancer{
		nodes:   nodes,
		current: 0,
		total:   total,
	}
}

// Next 获取下一个节点（加权轮询）
func (lb *LoadBalancer) Next() *LoadBalancerNode {
	if len(lb.nodes) == 0 {
		return nil
	}
	
	if len(lb.nodes) == 1 {
		return &lb.nodes[0]
	}
	
	// 简单轮询（可以扩展为加权轮询）
	node := &lb.nodes[lb.current]
	lb.current = (lb.current + 1) % len(lb.nodes)
	
	return node
}

// Repository 仓库基类
type Repository struct {
	db Database
}

// NewRepository 创建仓库实例
func NewRepository(db Database) *Repository {
	return &Repository{db: db}
}

// GetDB 获取数据库连接
func (r *Repository) GetDB() *gorm.DB {
	return r.db.GetDB()
}

// GetReadDB 获取只读数据库连接
func (r *Repository) GetReadDB() *gorm.DB {
	return r.db.GetReadDB()
}

// GetWriteDB 获取写数据库连接
func (r *Repository) GetWriteDB() *gorm.DB {
	return r.db.GetWriteDB()
}

// Transaction 执行事务
func (r *Repository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(ctx, fn)
}

// BaseModel 基础模型
type BaseModel struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// QueryBuilder 查询构建器
type QueryBuilder struct {
	db    *gorm.DB
	query *gorm.DB
}

// NewQueryBuilder 创建查询构建器
func NewQueryBuilder(db *gorm.DB) *QueryBuilder {
	return &QueryBuilder{
		db:    db,
		query: db,
	}
}

// Where 添加where条件
func (qb *QueryBuilder) Where(query interface{}, args ...interface{}) *QueryBuilder {
	qb.query = qb.query.Where(query, args...)
	return qb
}

// Order 添加排序
func (qb *QueryBuilder) Order(value interface{}) *QueryBuilder {
	qb.query = qb.query.Order(value)
	return qb
}

// Limit 添加限制
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.query = qb.query.Limit(limit)
	return qb
}

// Offset 添加偏移
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.query = qb.query.Offset(offset)
	return qb
}

// Preload 预加载关联
func (qb *QueryBuilder) Preload(query string, args ...interface{}) *QueryBuilder {
	qb.query = qb.query.Preload(query, args...)
	return qb
}

// Joins 添加连接
func (qb *QueryBuilder) Joins(query string, args ...interface{}) *QueryBuilder {
	qb.query = qb.query.Joins(query, args...)
	return qb
}

// Group 添加分组
func (qb *QueryBuilder) Group(name string) *QueryBuilder {
	qb.query = qb.query.Group(name)
	return qb
}

// Having 添加having条件
func (qb *QueryBuilder) Having(query interface{}, args ...interface{}) *QueryBuilder {
	qb.query = qb.query.Having(query, args...)
	return qb
}

// Select 选择字段
func (qb *QueryBuilder) Select(query interface{}, args ...interface{}) *QueryBuilder {
	qb.query = qb.query.Select(query, args...)
	return qb
}

// Find 查询多条记录
func (qb *QueryBuilder) Find(dest interface{}) error {
	return qb.query.Find(dest).Error
}

// First 查询第一条记录
func (qb *QueryBuilder) First(dest interface{}) error {
	return qb.query.First(dest).Error
}

// Count 统计记录数
func (qb *QueryBuilder) Count(count *int64) error {
	return qb.query.Count(count).Error
}

// Create 创建记录
func (qb *QueryBuilder) Create(value interface{}) error {
	return qb.db.Create(value).Error
}

// Update 更新记录
func (qb *QueryBuilder) Update(column string, value interface{}) error {
	return qb.query.Update(column, value).Error
}

// Updates 批量更新
func (qb *QueryBuilder) Updates(values interface{}) error {
	return qb.query.Updates(values).Error
}

// Delete 删除记录
func (qb *QueryBuilder) Delete(value interface{}) error {
	return qb.query.Delete(value).Error
}

// GetDB 获取GORM实例
func (qb *QueryBuilder) GetDB() *gorm.DB {
	return qb.query
}

// Reset 重置查询
func (qb *QueryBuilder) Reset() *QueryBuilder {
	qb.query = qb.db
	return qb
}

// Paginate 分页查询
func (qb *QueryBuilder) Paginate(page, pageSize int) *QueryBuilder {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	
	offset := (page - 1) * pageSize
	return qb.Offset(offset).Limit(pageSize)
}

// PageResult 分页结果
type PageResult struct {
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Pages    int         `json:"pages"`
	Data     interface{} `json:"data"`
}

// FindWithPagination 分页查询
func (qb *QueryBuilder) FindWithPagination(dest interface{}, page, pageSize int) (*PageResult, error) {
	var total int64
	
	// 先统计总数
	countQuery := qb.db.Model(dest)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}
	
	// 分页查询
	if err := qb.Paginate(page, pageSize).Find(dest); err != nil {
		return nil, err
	}
	
	pages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		pages++
	}
	
	return &PageResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    pages,
		Data:     dest,
	}, nil
}