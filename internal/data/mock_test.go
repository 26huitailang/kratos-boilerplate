package data

import (
	"context"
	"os"
	"testing"
	"time"

	"kratos-boilerplate/internal/biz"
	"kratos-boilerplate/internal/pkg/crypto"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// 验证repo实现了biz.UserRepo接口
var _ biz.UserRepo = (*userRepo)(nil)

// 测试NewUserRepo
func TestNewUserRepo(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	kmsManager := &mockKMSManager{}
	repo, err := NewUserRepo(data, logger, kmsManager)
	assert.NoError(t, err)
	assert.NotNil(t, repo)
}

// 模拟DB接口
type mockDB struct {
	mock.Mock
}

func (m *mockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
	called := m.Called(ctx, query, args)
	return called.Get(0), called.Error(1)
}

func (m *mockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) interface{} {
	return m.Called(ctx, query, args)
}

// 模拟加密器
type mockEncryptor struct {
	mock.Mock
}

func (m *mockEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	args := m.Called(plaintext)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	args := m.Called(ciphertext)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockEncryptor) Hash(plaintext []byte) string {
	args := m.Called(plaintext)
	return args.String(0)
}

// 测试加密相关操作
func TestCryptoOperations(t *testing.T) {
	encryptor, err := crypto.NewAESEncryptor(make([]byte, 32))
	assert.NoError(t, err)

	// 加密测试
	plaintext := []byte("test-data")
	encrypted, err := encryptor.Encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	// 解密测试
	decrypted, err := encryptor.Decrypt(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)

	// 哈希测试
	hash := encryptor.Hash(plaintext)
	assert.NotEmpty(t, hash)
}

// 测试内存存储的操作
func TestInMemoryStorage(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	data := &Data{}

	kmsManager := &mockKMSManager{}
	repo, err := NewUserRepo(data, logger, kmsManager)
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
