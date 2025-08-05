package data

import (
	"context"
	"os"
	"testing"
	"time"

	"kratos-boilerplate/internal/biz"
	"kratos-boilerplate/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试数据层初始化 - 各种配置情况
func TestDataInitialization(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)

	// 测试nil配置
	_, _, err := NewData(nil, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is nil")

	// 测试缺少数据库配置
	config1 := &conf.Data{
		Redis: &conf.Data_Redis{
			Addr:    "localhost:6379",
			Network: "tcp",
		},
	}
	_, _, err = NewData(config1, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database config is nil")

	// 测试缺少Redis配置
	config2 := &conf.Data{
		Database: &conf.Data_Database{
			Driver: "postgres",
			Source: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		},
	}
	_, _, err = NewData(config2, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis config is nil")
}

// 测试用户仓储创建
func TestUserRepoCreation(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	kmsManager := &mockKMSManager{}
	repo, err := NewUserRepo(data, logger, kmsManager)
	require.NoError(t, err)
	assert.NotNil(t, repo)

	// 验证接口实现
	var _ biz.UserRepo = repo
}

// 测试验证码仓储创建
func TestCaptchaRepoCreation(t *testing.T) {
	data := &Data{}
	repo := NewCaptchaRepo(data)
	assert.NotNil(t, repo)

	// 验证接口实现
	var _ biz.CaptchaRepo = repo
}

// 测试操作日志仓储创建
func TestOperationLogRepoCreation(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewOperationLogRepo(data, logger)
	assert.NotNil(t, repo)

	// 验证接口实现
	var _ biz.OperationLogRepo = repo
}

// 测试Greeter仓储创建
func TestGreeterRepoCreation(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewGreeterRepo(data, logger)
	assert.NotNil(t, repo)

	// 验证接口实现
	var _ biz.GreeterRepo = repo
}

// 测试用户数据结构
func TestUserDataStructure(t *testing.T) {
	user := &biz.User{
		ID:         1,
		Username:   "testuser",
		Password:   "hashedpassword",
		Email:      "test@example.com",
		Phone:      "13800138000",
		Name:       "测试用户",
		TotpSecret: "secret",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	assert.Equal(t, int64(1), user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "hashedpassword", user.Password)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "13800138000", user.Phone)
	assert.Equal(t, "测试用户", user.Name)
	assert.Equal(t, "secret", user.TotpSecret)
	assert.NotZero(t, user.CreatedAt)
	assert.NotZero(t, user.UpdatedAt)
}

// 测试账户锁定数据结构
func TestAccountLockDataStructure(t *testing.T) {
	lock := &biz.AccountLock{
		Username:       "testuser",
		FailedAttempts: 3,
		LockUntil:      time.Now().Add(30 * time.Minute),
		LastAttempt:    time.Now(),
	}

	assert.Equal(t, "testuser", lock.Username)
	assert.Equal(t, int32(3), lock.FailedAttempts)
	assert.True(t, lock.LockUntil.After(time.Now()))
	assert.NotZero(t, lock.LastAttempt)
}

// 测试刷新令牌信息结构
func TestRefreshTokenInfoStructure(t *testing.T) {
	info := &refreshTokenInfo{
		username: "testuser",
		used:     false,
		expireAt: time.Now().Add(time.Hour),
	}

	assert.Equal(t, "testuser", info.username)
	assert.False(t, info.used)
	assert.True(t, info.expireAt.After(time.Now()))
}

// 测试用户仓储接口方法
func TestUserRepoInterfaceMethods(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	kmsManager := &mockKMSManager{}
		repo, err := NewUserRepo(data, logger, kmsManager)
		require.NoError(t, err)

	userRepo := repo.(*userRepo)

	// 测试所有接口方法都存在
	// 为了避免panic，我们不实际调用这些方法
	// 只验证接口实现
	var _ biz.UserRepo = userRepo
}

// 测试验证码仓储接口方法
func TestCaptchaRepoInterfaceMethods(t *testing.T) {
	data := &Data{}
	repo := NewCaptchaRepo(data)
	captchaRepo := repo.(*captchaRepo)

	ctx := context.Background()

	// 测试所有接口方法都存在
	captcha := &biz.Captcha{
		ID:       "test-id",
		Code:     "123456",
		Type:     "image",
		Target:   "test@example.com",
		ExpireAt: time.Now().Add(5 * time.Minute),
		Used:     false,
	}

	// 这些方法会失败，因为Redis为nil，但我们可以验证方法存在
	_ = captchaRepo.SaveCaptcha(ctx, captcha)
	_, _ = captchaRepo.GetCaptcha(ctx, "test-id")
	_ = captchaRepo.DeleteCaptcha(ctx, "test-id")
	_ = captchaRepo.MarkCaptchaUsed(ctx, "test-id")
}

// 测试操作日志仓储接口方法
func TestOperationLogRepoInterfaceMethods(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)
	repo := NewOperationLogRepo(data, logger)
	operationLogRepo := repo.(*operationLogRepo)

	// 测试所有接口方法都存在
	// 为了避免panic，我们不实际调用这些方法
	// 只验证接口实现
	var _ biz.OperationLogRepo = operationLogRepo
}

// 测试Greeter仓储接口方法
func TestGreeterRepoInterfaceMethods(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)
	repo := NewGreeterRepo(data, logger)
	greeterRepo := repo.(*greeterRepo)

	ctx := context.Background()

	// 测试所有接口方法都存在
	greeter := &biz.Greeter{
		Hello: "test hello",
	}

	// 这些方法会成功，因为它们是简单的返回
	result, err := greeterRepo.Save(ctx, greeter)
	assert.NoError(t, err)
	assert.Equal(t, greeter, result)

	result, err = greeterRepo.Update(ctx, greeter)
	assert.NoError(t, err)
	assert.Equal(t, greeter, result)

	result, err = greeterRepo.FindByID(ctx, 1)
	assert.NoError(t, err)
	assert.Nil(t, result)

	results, err := greeterRepo.ListByHello(ctx, "test")
	assert.NoError(t, err)
	assert.Nil(t, results)

	results, err = greeterRepo.ListAll(ctx)
	assert.NoError(t, err)
	assert.Nil(t, results)
}

// 测试错误处理
func TestErrorHandling(t *testing.T) {
	// 测试各种错误情况
	logger := log.NewStdLogger(os.Stdout)

	// 测试nil配置
	_, _, err := NewData(nil, logger)
	assert.Error(t, err)

	// 测试空配置
	emptyConfig := &conf.Data{}
	_, _, err = NewData(emptyConfig, logger)
	assert.Error(t, err)

	// 测试部分配置
	partialConfig := &conf.Data{
		Database: &conf.Data_Database{
			Driver: "postgres",
			Source: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		},
	}
	_, _, err = NewData(partialConfig, logger)
	assert.Error(t, err)
}

// 测试并发安全性
func TestConcurrency(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	kmsManager := &mockKMSManager{}
		repo, err := NewUserRepo(data, logger, kmsManager)
		require.NoError(t, err)

		userRepo := repo.(*userRepo)
	ctx := context.Background()

	// 并发测试
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			// 测试内存存储操作
			captcha := &biz.Captcha{
				ID:       "test-id",
				Code:     "123456",
				Type:     "image",
				ExpireAt: time.Now().Add(5 * time.Minute),
			}

			_ = userRepo.SaveCaptcha(ctx, captcha)
			_, _ = userRepo.GetCaptcha(ctx, "test-id")

			lock := &biz.AccountLock{
				Username:       "testuser",
				FailedAttempts: 1,
				LastAttempt:    time.Now(),
			}

			_ = userRepo.SaveLock(ctx, lock)
			_, _ = userRepo.GetLock(ctx, "testuser")

			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
}
