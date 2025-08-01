//go:build integration

package database_test

import (
	"context"
	"testing"
	"time"

	"kratos-boilerplate/internal/biz"
	"kratos-boilerplate/internal/conf"
	"kratos-boilerplate/internal/data"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// AuthIntegrationTestSuite 认证集成测试套件
type AuthIntegrationTestSuite struct {
	suite.Suite
	data     *data.Data
	cleanup  func()
	userRepo biz.UserRepo
	ctx      context.Context
	logger   log.Logger
}

// SetupSuite 在测试套件开始前执行
func (suite *AuthIntegrationTestSuite) SetupSuite() {
	suite.logger = log.NewStdLogger(suite.T().Logf)
	suite.ctx = context.Background()

	// 配置测试数据库
	config := &conf.Data{
		Database: &conf.Data_Database{
			Driver: "postgres",
			Source: "postgres://postgres:postgres@localhost:5432/test_db?sslmode=disable",
		},
		Redis: &conf.Data_Redis{
			Addr:    "localhost:6379",
			Network: "tcp",
		},
	}

	var err error
	suite.data, suite.cleanup, err = data.NewData(config, suite.logger)
	require.NoError(suite.T(), err, "Failed to setup test database")

	// 创建用户仓储
	suite.userRepo, err = data.NewUserRepo(suite.data, suite.logger)
	require.NoError(suite.T(), err, "Failed to create user repository")
}

// TearDownSuite 在测试套件结束后执行
func (suite *AuthIntegrationTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

// SetupTest 在每个测试用例前执行
func (suite *AuthIntegrationTestSuite) SetupTest() {
	// 清理测试数据
	suite.cleanupTestData()
}

// TearDownTest 在每个测试用例后执行
func (suite *AuthIntegrationTestSuite) TearDownTest() {
	// 清理测试数据
	suite.cleanupTestData()
}

// cleanupTestData 清理测试数据
func (suite *AuthIntegrationTestSuite) cleanupTestData() {
	// 这里应该清理测试过程中创建的数据
	// 例如删除测试用户、清理缓存等
	// 具体实现取决于你的数据模型
}

// TestCreateUser 测试创建用户
func (suite *AuthIntegrationTestSuite) TestCreateUser() {
	// 准备测试数据
	user := &biz.User{
		Username:  "testuser_" + suite.generateRandomString(8),
		Password:  "$2a$10$hashedpassword",
		Email:     "test@example.com",
		Phone:     "13800138000",
		Name:      "Test User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 执行测试
	err := suite.userRepo.CreateUser(suite.ctx, user)

	// 验证结果
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), user.ID) // 确保ID被设置

	// 验证用户是否真的被创建
	retrievedUser, err := suite.userRepo.GetUser(suite.ctx, user.Username)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Username, retrievedUser.Username)
	assert.Equal(suite.T(), user.Email, retrievedUser.Email)
}

// TestGetUser 测试获取用户
func (suite *AuthIntegrationTestSuite) TestGetUser() {
	// 先创建一个用户
	user := &biz.User{
		Username:  "getuser_" + suite.generateRandomString(8),
		Password:  "$2a$10$hashedpassword",
		Email:     "getuser@example.com",
		Phone:     "13800138001",
		Name:      "Get User Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := suite.userRepo.CreateUser(suite.ctx, user)
	require.NoError(suite.T(), err)

	// 测试获取用户
	retrievedUser, err := suite.userRepo.GetUser(suite.ctx, user.Username)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Username, retrievedUser.Username)
	assert.Equal(suite.T(), user.Email, retrievedUser.Email)
	assert.Equal(suite.T(), user.Phone, retrievedUser.Phone)

	// 测试获取不存在的用户
	_, err = suite.userRepo.GetUser(suite.ctx, "nonexistent_user")
	assert.Equal(suite.T(), biz.ErrUserNotFound, err)
}

// TestGetUserByEmail 测试通过邮箱获取用户
func (suite *AuthIntegrationTestSuite) TestGetUserByEmail() {
	// 先创建一个用户
	user := &biz.User{
		Username:  "emailuser_" + suite.generateRandomString(8),
		Password:  "$2a$10$hashedpassword",
		Email:     "emailuser_" + suite.generateRandomString(8) + "@example.com",
		Phone:     "13800138002",
		Name:      "Email User Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := suite.userRepo.CreateUser(suite.ctx, user)
	require.NoError(suite.T(), err)

	// 测试通过邮箱获取用户
	retrievedUser, err := suite.userRepo.GetUserByEmail(suite.ctx, user.Email)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Username, retrievedUser.Username)
	assert.Equal(suite.T(), user.Email, retrievedUser.Email)

	// 测试获取不存在的邮箱
	_, err = suite.userRepo.GetUserByEmail(suite.ctx, "nonexistent@example.com")
	assert.Equal(suite.T(), biz.ErrUserNotFound, err)
}

// TestAccountLock 测试账户锁定功能
func (suite *AuthIntegrationTestSuite) TestAccountLock() {
	username := "lockuser_" + suite.generateRandomString(8)

	// 测试获取不存在的锁定信息
	_, err := suite.userRepo.GetLock(suite.ctx, username)
	assert.Equal(suite.T(), biz.ErrUserNotFound, err)

	// 创建锁定信息
	lock := &biz.AccountLock{
		Username:       username,
		FailedAttempts: 3,
		LockUntil:      time.Now().Add(30 * time.Minute),
		LastAttempt:    time.Now(),
	}

	err = suite.userRepo.SaveLock(suite.ctx, lock)
	assert.NoError(suite.T(), err)

	// 获取锁定信息
	retrievedLock, err := suite.userRepo.GetLock(suite.ctx, username)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), lock.Username, retrievedLock.Username)
	assert.Equal(suite.T(), lock.FailedAttempts, retrievedLock.FailedAttempts)
	assert.WithinDuration(suite.T(), lock.LockUntil, retrievedLock.LockUntil, time.Second)

	// 删除锁定信息
	err = suite.userRepo.RemoveLock(suite.ctx, username)
	assert.NoError(suite.T(), err)

	// 验证锁定信息已被删除
	_, err = suite.userRepo.GetLock(suite.ctx, username)
	assert.Equal(suite.T(), biz.ErrUserNotFound, err)
}

// TestRefreshToken 测试刷新令牌功能
func (suite *AuthIntegrationTestSuite) TestRefreshToken() {
	username := "tokenuser_" + suite.generateRandomString(8)
	tokenID := "token_" + suite.generateRandomString(16)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// 保存刷新令牌
	err := suite.userRepo.SaveRefreshToken(suite.ctx, username, tokenID, expiresAt)
	assert.NoError(suite.T(), err)

	// 获取刷新令牌
	retrievedUsername, valid, err := suite.userRepo.GetRefreshToken(suite.ctx, tokenID)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), valid)
	assert.Equal(suite.T(), username, retrievedUsername)

	// 使令牌无效
	err = suite.userRepo.InvalidateRefreshToken(suite.ctx, tokenID)
	assert.NoError(suite.T(), err)

	// 验证令牌已无效
	_, valid, err = suite.userRepo.GetRefreshToken(suite.ctx, tokenID)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), valid)
}

// TestCaptcha 测试验证码功能
func (suite *AuthIntegrationTestSuite) TestCaptcha() {
	captcha := &biz.Captcha{
		ID:       "captcha_" + suite.generateRandomString(16),
		Code:     "123456",
		Type:     "image",
		Target:   "",
		ExpireAt: time.Now().Add(5 * time.Minute),
		Used:     false,
	}

	// 保存验证码
	err := suite.userRepo.SaveCaptcha(suite.ctx, captcha)
	assert.NoError(suite.T(), err)

	// 获取验证码
	retrievedCaptcha, err := suite.userRepo.GetCaptcha(suite.ctx, captcha.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), captcha.ID, retrievedCaptcha.ID)
	assert.Equal(suite.T(), captcha.Code, retrievedCaptcha.Code)
	assert.Equal(suite.T(), captcha.Type, retrievedCaptcha.Type)
	assert.False(suite.T(), retrievedCaptcha.Used)

	// 标记验证码为已使用
	err = suite.userRepo.MarkCaptchaUsed(suite.ctx, captcha.ID)
	assert.NoError(suite.T(), err)

	// 验证验证码已被标记为已使用
	retrievedCaptcha, err = suite.userRepo.GetCaptcha(suite.ctx, captcha.ID)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), retrievedCaptcha.Used)
}

// TestConcurrentOperations 测试并发操作
func (suite *AuthIntegrationTestSuite) TestConcurrentOperations() {
	const numGoroutines = 10
	const numOperations = 5

	// 创建基础用户
	baseUsername := "concurrent_" + suite.generateRandomString(8)
	user := &biz.User{
		Username:  baseUsername,
		Password:  "$2a$10$hashedpassword",
		Email:     baseUsername + "@example.com",
		Phone:     "13800138000",
		Name:      "Concurrent Test User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := suite.userRepo.CreateUser(suite.ctx, user)
	require.NoError(suite.T(), err)

	// 并发测试锁定操作
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				// 保存锁定信息
				lock := &biz.AccountLock{
					Username:       baseUsername,
					FailedAttempts: int32(j + 1),
					LockUntil:      time.Now().Add(time.Duration(j+1) * time.Minute),
					LastAttempt:    time.Now(),
				}

				err := suite.userRepo.SaveLock(suite.ctx, lock)
				assert.NoError(suite.T(), err)

				// 获取锁定信息
				_, err = suite.userRepo.GetLock(suite.ctx, baseUsername)
				assert.NoError(suite.T(), err)

				time.Sleep(10 * time.Millisecond) // 短暂休眠
			}
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			suite.T().Fatal("Timeout waiting for concurrent operations to complete")
		}
	}
}

// generateRandomString 生成随机字符串
func (suite *AuthIntegrationTestSuite) generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// TestAuthIntegrationSuite 运行集成测试套件
func TestAuthIntegrationSuite(t *testing.T) {
	// 检查是否有集成测试环境
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(AuthIntegrationTestSuite))
}