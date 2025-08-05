package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kratos-boilerplate/internal/biz"
	"kratos-boilerplate/internal/pkg/crypto"
	"kratos-boilerplate/internal/pkg/kms"
)

// 测试用户数据库操作
func TestUserDatabaseOperations(t *testing.T) {
	// 创建sqlmock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 创建数据层
	data := &Data{
		db:    db,
		redis: nil, // 不需要Redis
	}

	// 创建mock加密服务
	mockCrypto := &mockCryptoService{}

	// 创建用户仓储
	logger := log.NewStdLogger(os.Stdout)
	// 创建模拟的KMS管理器
	kmsManager := &mockKMSManager{}
	userRepo := &userRepo{
		data: data,
		log:  log.NewHelper(logger),
		enc:  mockCrypto, // 直接使用mock加密服务
		kms:  kmsManager,
		captchas:     sync.Map{},
		accountLocks: sync.Map{},
		refreshTokens: sync.Map{},
	}

	ctx := context.Background()

	t.Run("CreateUser", func(t *testing.T) {
		assert.NoError(t, mock.ExpectationsWereMet()) // 前置清理

		user := &biz.User{
			Username: "testuser",
			Password: "hashedpassword",
			Email:    "test@example.com",
			Phone:    "13800138000",
			Name:     "测试用户",
		}

		// 设置mock期望 - 使用ExpectQuery因为CreateUser使用QueryRowContext
		mock.ExpectQuery("INSERT INTO users").
			WithArgs(
				sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(),
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		// 执行测试
		err := userRepo.CreateUser(ctx, user)
		assert.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet()) // 后置清理
	})

	t.Run("GetUser", func(t *testing.T) {
		assert.NoError(t, mock.ExpectationsWereMet()) // 前置清理

		username := "testuser"
		expectedUser := &biz.User{
			ID:       1,
			Username: username,
			Password: "hashedpassword",
			Email:    "test@example.com",
			Phone:    "13800138000",
			Name:     "测试用户",
		}

		// 设置mock期望 - 返回有效的加密数据
		encryptedEmail := []byte("encrypted_email_data")
		encryptedPhone := []byte("encrypted_phone_data")
		encryptedName := []byte("encrypted_name_data")
		
		rows := sqlmock.NewRows([]string{"id", "username", "password", "email_encrypted", "phone_encrypted", "name_encrypted", "created_at", "updated_at"}).
			AddRow(expectedUser.ID, expectedUser.Username, expectedUser.Password, encryptedEmail, encryptedPhone, encryptedName, time.Now(), time.Now())

		mock.ExpectQuery("SELECT (.+) FROM users").
			WithArgs(username).
			WillReturnRows(rows)

		// 执行测试
		user, err := userRepo.GetUser(ctx, username)
		// 现在应该成功解密
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser.Username, user.Username)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "13800138000", user.Phone)
		assert.Equal(t, "测试用户", user.Name)

		assert.NoError(t, mock.ExpectationsWereMet()) // 后置清理
	})

	t.Run("GetUserByXxx", func(t *testing.T) {
		tests := []struct {
			name      string
			input     string
			queryType string // email/phone/name
			field     string
		}{
			{"ByEmail", "test@example.com", "email", "Email"},
			{"ByPhone", "13800138000", "phone", "Phone"},
			{"ByName", "测试用户", "name", "Name"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.NoError(t, mock.ExpectationsWereMet()) // 前置清理

				expectedUser := &biz.User{
					ID:       1,
					Username: "testuser",
					Password: "hashedpassword",
					Email:    "test@example.com",
					Phone:    "13800138000",
					Name:     "测试用户",
				}

				// 使用mock加密服务生成加密数据
				mockCrypto := &mockCryptoService{}
				encryptedEmailField, _ := mockCrypto.EncryptField(ctx, "email", []byte(expectedUser.Email))
				encryptedPhoneField, _ := mockCrypto.EncryptField(ctx, "phone", []byte(expectedUser.Phone))
				encryptedNameField, _ := mockCrypto.EncryptField(ctx, "name", []byte(expectedUser.Name))
				
				// 序列化加密数据为字节数组（简化处理）
				encryptedEmail := []byte("encrypted_email_data")
				encryptedPhone := []byte("encrypted_phone_data")
				encryptedName := []byte("encrypted_name_data")
				_ = encryptedEmailField
				_ = encryptedPhoneField
				_ = encryptedNameField

				rows := sqlmock.NewRows([]string{"id", "username", "password", "email_encrypted", "phone_encrypted", "name_encrypted", "created_at", "updated_at"}).
					AddRow(expectedUser.ID, expectedUser.Username, expectedUser.Password, encryptedEmail, encryptedPhone, encryptedName, time.Now(), time.Now())

				var hash string
				switch tt.queryType {
				case "email":
					hash = mockCrypto.HashField([]byte(tt.input))
				case "phone":
					hash = mockCrypto.HashField([]byte(tt.input))
				case "name":
					hash = mockCrypto.HashField([]byte(tt.input))
				}

				mock.ExpectQuery("SELECT (.+) FROM users").
					WithArgs(hash).
					WillReturnRows(rows)

				var user *biz.User
				var err error
				switch tt.queryType {
				case "email":
					user, err = userRepo.GetUserByEmail(ctx, tt.input)
				case "phone":
					user, err = userRepo.GetUserByPhone(ctx, tt.input)
				case "name":
					user, err = userRepo.GetUserByName(ctx, tt.input)
				}
				// 由于使用了mock加密服务，应该能成功解密
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, expectedUser.Username, user.Username)

				assert.NoError(t, mock.ExpectationsWereMet()) // 后置清理
			})
		}
	})

	t.Run("UpdateUser", func(t *testing.T) {
		assert.NoError(t, mock.ExpectationsWereMet()) // 前置清理

		user := &biz.User{
			ID:       1,
			Username: "updateduser",
			Password: "newhashedpassword",
			Email:    "updated@example.com",
			Phone:    "13900139000",
			Name:     "更新用户",
		}

		// 设置mock期望 - 8个参数：emailEnc, emailHash, phoneEnc, phoneHash, nameEnc, nameHash, time.Now(), u.ID
		mock.ExpectExec("UPDATE users").
			WithArgs(
				sqlmock.AnyArg(), sqlmock.AnyArg(), // emailEnc, emailHash
				sqlmock.AnyArg(), sqlmock.AnyArg(), // phoneEnc, phoneHash
				sqlmock.AnyArg(), sqlmock.AnyArg(), // nameEnc, nameHash
				sqlmock.AnyArg(), user.ID, // time.Now(), u.ID
			).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// 执行测试
		err := userRepo.UpdateUser(ctx, user)
		assert.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet()) // 后置清理
	})

	t.Run("GetUser_NotFound", func(t *testing.T) {
		assert.NoError(t, mock.ExpectationsWereMet()) // 前置清理

		username := "notfounduser"

		// 设置mock期望 - 没有找到用户
		mock.ExpectQuery("SELECT (.+) FROM users").
			WithArgs(username).
			WillReturnError(sql.ErrNoRows)

		// 执行测试
		user, err := userRepo.GetUser(ctx, username)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, biz.ErrUserNotFound, err)

		assert.NoError(t, mock.ExpectationsWereMet()) // 后置清理
	})

	t.Run("CreateUser_DuplicateUsername", func(t *testing.T) {
		assert.NoError(t, mock.ExpectationsWereMet()) // 前置清理
		user := &biz.User{
			Username: "duplicateuser",
			Password: "hashedpassword",
			Email:    "duplicate@example.com",
			Phone:    "13800138000",
			Name:     "重复用户",
		}
		expectedErr := fmt.Errorf("duplicate key value violates unique constraint")
		// 设置mock期望 - 重复用户名错误，使用ExpectQuery因为CreateUser使用QueryRowContext
		mock.ExpectQuery("INSERT INTO users").
			WithArgs(
				sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(),
			).
			WillReturnError(expectedErr)
		// 执行测试
		err := userRepo.CreateUser(ctx, user)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.NoError(t, mock.ExpectationsWereMet()) // 后置清理
	})
}

// mockKMSManager 是用于测试的模拟KMS管理器
type mockKMSManager struct{}

func (m *mockKMSManager) Initialize(ctx context.Context, config *kms.Config) error {
	return nil
}

func (m *mockKMSManager) GetActiveDataKey(ctx context.Context) (*kms.DataKey, error) {
	return &kms.DataKey{
		ID:        "test-key-id",
		Version:   "v1",
		Algorithm: "AES-GCM-256",
		Key:       []byte("0123456789abcdef0123456789abcdef"),
		IsActive:  true,
	}, nil
}

func (m *mockKMSManager) GetDataKeyByVersion(ctx context.Context, version string) (*kms.DataKey, error) {
	return &kms.DataKey{
		ID:        "test-key-id",
		Version:   version,
		Algorithm: "AES-GCM-256",
		Key:       []byte("0123456789abcdef0123456789abcdef"),
		IsActive:  true,
	}, nil
}

func (m *mockKMSManager) RotateDataKey(ctx context.Context) (*kms.DataKey, error) {
	return &kms.DataKey{
		ID:        "test-key-id",
		Version:   "v2",
		Algorithm: "AES-GCM-256",
		Key:       []byte("0123456789abcdef0123456789abcdef"),
		IsActive:  true,
	}, nil
}

func (m *mockKMSManager) GetCryptoService() kms.CryptoService {
	return &mockCryptoService{}
}

func (m *mockKMSManager) Close() error {
	return nil
}

// mockCryptoService 是用于测试的模拟加密服务
type mockCryptoService struct{}

// 确保mockCryptoService实现了crypto.Encryptor接口
var _ crypto.Encryptor = (*mockCryptoService)(nil)

func (m *mockCryptoService) EncryptField(ctx context.Context, fieldName string, plaintext []byte) (*kms.EncryptedField, error) {
	return &kms.EncryptedField{
		FieldName: fieldName,
		EncryptedData: &kms.EncryptedData{
			KeyVersion: "v1",
			Algorithm:  "AES-256-GCM",
			Ciphertext: plaintext, // 简化测试，直接返回明文
		},
		Hash: "mock-hash",
	}, nil
}

func (m *mockCryptoService) DecryptField(ctx context.Context, encryptedField *kms.EncryptedField) ([]byte, error) {
	return encryptedField.EncryptedData.Ciphertext, nil // 简化测试，直接返回密文
}

// 为了兼容旧的crypto接口，添加Decrypt方法
func (m *mockCryptoService) Decrypt(data []byte) ([]byte, error) {
	// 简化处理：根据加密数据返回对应的明文
	switch string(data) {
	case "encrypted_email_data":
		return []byte("test@example.com"), nil
	case "encrypted_phone_data":
		return []byte("13800138000"), nil
	case "encrypted_name_data":
		return []byte("测试用户"), nil
	default:
		return data, nil // 如果不是我们的测试数据，直接返回
	}
}

// 为了兼容旧的crypto接口，添加Hash方法
func (m *mockCryptoService) Hash(data []byte) string {
	return m.HashField(data)
}

// 为了兼容crypto.Encryptor接口，添加Encrypt方法
func (m *mockCryptoService) Encrypt(data []byte) ([]byte, error) {
	// 简化处理：根据明文返回对应的加密数据
	switch string(data) {
	case "test@example.com":
		return []byte("encrypted_email_data"), nil
	case "13800138000":
		return []byte("encrypted_phone_data"), nil
	case "测试用户":
		return []byte("encrypted_name_data"), nil
	default:
		return []byte("encrypted_" + string(data)), nil
	}
}

func (m *mockCryptoService) HashField(data []byte) string {
	// 使用简单的hash算法来模拟，确保与测试期望一致
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func (m *mockCryptoService) EncryptBatch(ctx context.Context, fields map[string][]byte) (map[string]*kms.EncryptedField, error) {
	result := make(map[string]*kms.EncryptedField)
	for fieldName, data := range fields {
		result[fieldName] = &kms.EncryptedField{
			FieldName: fieldName,
			EncryptedData: &kms.EncryptedData{
				KeyVersion: "v1",
				Algorithm:  "AES-256-GCM",
				Ciphertext: data,
			},
			Hash: "mock-hash",
		}
	}
	return result, nil
}

func (m *mockCryptoService) DecryptBatch(ctx context.Context, encryptedFields map[string]*kms.EncryptedField) (map[string][]byte, error) {
	result := make(map[string][]byte)
	for fieldName, encryptedField := range encryptedFields {
		result[fieldName] = encryptedField.EncryptedData.Ciphertext
	}
	return result, nil
}
