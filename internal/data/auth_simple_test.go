package data

import (
	"context"
	"os"
	"testing"
	"time"

	"kratos-boilerplate/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试NewUserRepo - 简化版本
func TestNewUserRepoSimple(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	repo, err := NewUserRepo(data, logger)

	require.NoError(t, err)
	assert.NotNil(t, repo)
}

// 测试内存存储操作 - 简化版本
func TestInMemoryStorageSimple(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	repo, err := NewUserRepo(data, logger)
	require.NoError(t, err)

	userRepo := repo.(*userRepo)
	ctx := context.Background()

	// 测试验证码存储
	captcha := &biz.Captcha{
		ID:       "test-id",
		Code:     "123456",
		Type:     "image",
		ExpireAt: time.Now().Add(5 * time.Minute),
	}

	err = userRepo.SaveCaptcha(ctx, captcha)
	assert.NoError(t, err)

	savedCaptcha, err := userRepo.GetCaptcha(ctx, "test-id")
	assert.NoError(t, err)
	assert.Equal(t, captcha, savedCaptcha)

	err = userRepo.MarkCaptchaUsed(ctx, "test-id")
	assert.NoError(t, err)

	markedCaptcha, err := userRepo.GetCaptcha(ctx, "test-id")
	assert.NoError(t, err)
	assert.True(t, markedCaptcha.Used)

	// 测试账户锁定存储
	lock := &biz.AccountLock{
		Username:       "testuser",
		FailedAttempts: 1,
		LastAttempt:    time.Now(),
	}

	err = userRepo.SaveLock(ctx, lock)
	assert.NoError(t, err)

	savedLock, err := userRepo.GetLock(ctx, "testuser")
	assert.NoError(t, err)
	assert.Equal(t, lock, savedLock)

	err = userRepo.RemoveLock(ctx, "testuser")
	assert.NoError(t, err)

	_, err = userRepo.GetLock(ctx, "testuser")
	assert.Equal(t, biz.ErrUserNotFound, err)

	// 测试刷新令牌存储
	now := time.Now()
	expiry := now.Add(time.Hour)

	err = userRepo.SaveRefreshToken(ctx, "testuser", "token-id", expiry)
	assert.NoError(t, err)

	username, used, err := userRepo.GetRefreshToken(ctx, "token-id")
	assert.NoError(t, err)
	assert.Equal(t, "testuser", username)
	assert.False(t, used)

	err = userRepo.InvalidateRefreshToken(ctx, "token-id")
	assert.NoError(t, err)

	_, used, err = userRepo.GetRefreshToken(ctx, "token-id")
	assert.NoError(t, err)
	assert.True(t, used)
}

// 测试刷新令牌过期 - 简化版本
func TestRefreshTokenExpiredSimple(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	repo, err := NewUserRepo(data, logger)
	require.NoError(t, err)

	userRepo := repo.(*userRepo)
	ctx := context.Background()

	// 保存一个已过期的令牌
	expiredTime := time.Now().Add(-time.Hour)
	err = userRepo.SaveRefreshToken(ctx, "testuser", "expired-token", expiredTime)
	assert.NoError(t, err)

	// 尝试获取过期令牌
	_, _, err = userRepo.GetRefreshToken(ctx, "expired-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "刷新令牌已过期")
}

// 测试InvalidateAllRefreshTokens - 简化版本
func TestInvalidateAllRefreshTokensSimple(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	repo, err := NewUserRepo(data, logger)
	require.NoError(t, err)

	userRepo := repo.(*userRepo)
	ctx := context.Background()

	// 保存多个令牌
	now := time.Now()
	expiry := now.Add(time.Hour)

	err = userRepo.SaveRefreshToken(ctx, "user1", "token1", expiry)
	assert.NoError(t, err)
	err = userRepo.SaveRefreshToken(ctx, "user1", "token2", expiry)
	assert.NoError(t, err)
	err = userRepo.SaveRefreshToken(ctx, "user2", "token3", expiry)
	assert.NoError(t, err)

	// 使user1的所有令牌失效
	err = userRepo.InvalidateAllRefreshTokens(ctx, "user1")
	assert.NoError(t, err)

	// 验证user1的令牌都已失效
	_, used1, err := userRepo.GetRefreshToken(ctx, "token1")
	assert.NoError(t, err)
	assert.True(t, used1)

	_, used2, err := userRepo.GetRefreshToken(ctx, "token2")
	assert.NoError(t, err)
	assert.True(t, used2)

	// 验证user2的令牌未受影响
	_, used3, err := userRepo.GetRefreshToken(ctx, "token3")
	assert.NoError(t, err)
	assert.False(t, used3)
}

// 测试验证码不存在的情况
func TestGetCaptchaNotFound(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	repo, err := NewUserRepo(data, logger)
	require.NoError(t, err)

	userRepo := repo.(*userRepo)
	ctx := context.Background()

	// 尝试获取不存在的验证码
	_, err = userRepo.GetCaptcha(ctx, "nonexistent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "验证码不存在")
}

// 测试标记不存在的验证码为已使用
func TestMarkCaptchaUsedNotFound(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	repo, err := NewUserRepo(data, logger)
	require.NoError(t, err)

	userRepo := repo.(*userRepo)
	ctx := context.Background()

	// 尝试标记不存在的验证码为已使用
	err = userRepo.MarkCaptchaUsed(ctx, "nonexistent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "验证码不存在")
}

// 测试获取不存在的账户锁定
func TestGetLockNotFound(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	repo, err := NewUserRepo(data, logger)
	require.NoError(t, err)

	userRepo := repo.(*userRepo)
	ctx := context.Background()

	// 尝试获取不存在的账户锁定
	_, err = userRepo.GetLock(ctx, "nonexistent-user")
	assert.Equal(t, biz.ErrUserNotFound, err)
}

// 测试获取不存在的刷新令牌
func TestGetRefreshTokenNotFound(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	repo, err := NewUserRepo(data, logger)
	require.NoError(t, err)

	userRepo := repo.(*userRepo)
	ctx := context.Background()

	// 尝试获取不存在的刷新令牌
	_, _, err = userRepo.GetRefreshToken(ctx, "nonexistent-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "刷新令牌不存在")
}

// 测试使不存在的刷新令牌失效
func TestInvalidateRefreshTokenNotFound(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	repo, err := NewUserRepo(data, logger)
	require.NoError(t, err)

	userRepo := repo.(*userRepo)
	ctx := context.Background()

	// 尝试使不存在的刷新令牌失效
	err = userRepo.InvalidateRefreshToken(ctx, "nonexistent-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "刷新令牌不存在")
}
