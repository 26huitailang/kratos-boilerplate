package data

import (
	"database/sql"
	"fmt"
	"time"
	"kratos-boilerplate/internal/biz"
	"kratos-boilerplate/internal/conf"
	"kratos-boilerplate/internal/pkg/captcha"
	"kratos-boilerplate/internal/pkg/kms"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGreeterRepo, NewUserRepo, NewOperationLogRepo, NewCaptchaRepo, captcha.NewCaptchaService, NewCaptchaConfig, NewKMSRepo, NewKMSManager)

// Data .
type Data struct {
	db    *sql.DB
	redis *redis.Client
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	if c == nil {
		return nil, nil, fmt.Errorf("config is nil")
	}
	if c.Database == nil {
		return nil, nil, fmt.Errorf("database config is nil")
	}
	if c.Redis == nil {
		return nil, nil, fmt.Errorf("redis config is nil")
	}

	db, err := sql.Open(c.Database.Driver, c.Database.Source)
	if err != nil {
		return nil, nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, nil, err
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Addr,
		Network:  c.Redis.Network,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	cleanup := func() {
		if err := db.Close(); err != nil {
			log.NewHelper(logger).Errorf("failed to close database: %v", err)
		}
		if err := redisClient.Close(); err != nil {
			log.NewHelper(logger).Errorf("failed to close redis: %v", err)
		}
		log.NewHelper(logger).Info("closing the data resources")
	}
	return &Data{db: db, redis: redisClient}, cleanup, nil
}

func NewCaptchaConfig(auth *conf.Auth) *captcha.Config {
	return &captcha.Config{
		EnableSMS:   auth.CaptchaEnabled,
		EnableEmail: auth.CaptchaEnabled,
		EnableImage: auth.CaptchaEnabled,
		Expiration:  auth.CaptchaExpiration.AsDuration(),
	}
}

// GetDB 获取数据库连接
func (d *Data) GetDB() *sql.DB {
	return d.db
}



// NewKMSManager 创建KMS管理器
func NewKMSManager(kmsRepo biz.KMSRepo, logger log.Logger) kms.KMSManager {
	// 创建默认配置
	config := &biz.KMSConfig{
		Seed:             "default-seed-value",
		Salt:             "default-salt-value",
		Iterations:       100000,
		KeyLength:        32,
		RotateInterval: 24 * time.Hour, // 24小时
		Algorithm:        "AES-256-GCM",
	}
	return kms.NewKMSManager(kmsRepo, config, logger)
}
