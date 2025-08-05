package data

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"kratos-boilerplate/internal/biz"
	"kratos-boilerplate/internal/pkg/crypto"
	"kratos-boilerplate/internal/pkg/kms"

	"github.com/go-kratos/kratos/v2/log"
)

type userRepo struct {
	data *Data
	log  *log.Helper
	enc  crypto.Encryptor
	kms  kms.KMSManager // KMS管理器

	// 内存存储验证码和账户锁定信息（实际项目应使用Redis等缓存）
	captchas      sync.Map // map[captchaID]*biz.Captcha
	accountLocks  sync.Map // map[username]*biz.AccountLock
	refreshTokens sync.Map // map[tokenID]struct{tokenUsername string, used bool}
}

// 刷新令牌信息
type refreshTokenInfo struct {
	username string
	used     bool
	expireAt time.Time
}

// NewUserRepo .
func NewUserRepo(data *Data, logger log.Logger, kmsManager kms.KMSManager) (biz.UserRepo, error) {
	// 使用KMS系统获取加密服务
	cryptoService := kmsManager.GetCryptoService()

	// 创建兼容的加密器包装
	enc := &kmsEncryptorWrapper{
		cryptoService: cryptoService,
	}

	return &userRepo{
		data: data,
		log:  log.NewHelper(logger),
		enc:  enc,
		kms:  kmsManager,
	}, nil
}

// kmsEncryptorWrapper KMS加密器包装，实现crypto.Encryptor接口
type kmsEncryptorWrapper struct {
	cryptoService kms.CryptoService
}

// Encrypt 加密数据
func (w *kmsEncryptorWrapper) Encrypt(data []byte) ([]byte, error) {
	encryptedField, err := w.cryptoService.EncryptField(context.Background(), "user_data", data)
	if err != nil {
		return nil, err
	}

	// 将加密结果序列化为字节数组
	// 这里简化处理，实际应该使用更完善的序列化方式
	return []byte(fmt.Sprintf("%s:%s:%x", encryptedField.EncryptedData.KeyVersion, encryptedField.EncryptedData.Algorithm, encryptedField.EncryptedData.Ciphertext)), nil
}

// Decrypt 解密数据
func (w *kmsEncryptorWrapper) Decrypt(data []byte) ([]byte, error) {
	// 解析加密数据格式
	parts := strings.Split(string(data), ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid encrypted data format")
	}

	// 将十六进制字符串转换回字节数组
	ciphertext, err := hex.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext format: %w", err)
	}

	encryptedField := &kms.EncryptedField{
		FieldName: "user_data",
		EncryptedData: &kms.EncryptedData{
			KeyVersion: parts[0],
			Algorithm:  parts[1],
			Ciphertext: ciphertext,
		},
	}

	decrypted, err := w.cryptoService.DecryptField(context.Background(), encryptedField)
	if err != nil {
		return nil, err
	}

	return decrypted, nil
}

// Hash 计算哈希值
func (w *kmsEncryptorWrapper) Hash(data []byte) string {
	hash := w.cryptoService.HashField(data)
	return hash
}

func (r *userRepo) CreateUser(ctx context.Context, u *biz.User) error {
	// 加密敏感信息
	emailEnc, err := r.enc.Encrypt([]byte(u.Email))
	if err != nil {
		return err
	}
	emailHash := r.enc.Hash([]byte(u.Email))

	phoneEnc, err := r.enc.Encrypt([]byte(u.Phone))
	if err != nil {
		return err
	}
	phoneHash := r.enc.Hash([]byte(u.Phone))

	nameEnc, err := r.enc.Encrypt([]byte(u.Name))
	if err != nil {
		return err
	}
	nameHash := r.enc.Hash([]byte(u.Name))

	query := `
		INSERT INTO users (
			username, password, 
			email_encrypted, email_hash,
			phone_encrypted, phone_hash,
			name_encrypted, name_hash,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	err = r.data.db.QueryRowContext(ctx, query,
		u.Username, u.Password,
		emailEnc, emailHash,
		phoneEnc, phoneHash,
		nameEnc, nameHash,
		time.Now(), time.Now(),
	).Scan(&u.ID)
	return err
}

func (r *userRepo) GetUser(ctx context.Context, username string) (*biz.User, error) {
	query := `
		SELECT id, username, password, 
			email_encrypted, phone_encrypted, name_encrypted,
			created_at, updated_at 
		FROM users 
		WHERE username = $1
	`
	user := &biz.User{}
	var emailEnc, phoneEnc, nameEnc []byte
	err := r.data.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Password,
		&emailEnc, &phoneEnc, &nameEnc,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, biz.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	// 解密敏感信息
	if emailEnc != nil {
		email, err := r.enc.Decrypt(emailEnc)
		if err != nil {
			return nil, err
		}
		user.Email = string(email)
	}

	if phoneEnc != nil {
		phone, err := r.enc.Decrypt(phoneEnc)
		if err != nil {
			return nil, err
		}
		user.Phone = string(phone)
	}

	if nameEnc != nil {
		name, err := r.enc.Decrypt(nameEnc)
		if err != nil {
			return nil, err
		}
		user.Name = string(name)
	}

	return user, nil
}

func (r *userRepo) GetUserByEmail(ctx context.Context, email string) (*biz.User, error) {
	emailHash := r.enc.Hash([]byte(email))
	query := `
		SELECT id, username, password, 
			email_encrypted, phone_encrypted, name_encrypted,
			created_at, updated_at 
		FROM users 
		WHERE email_hash = $1
	`
	return r.getUserByQuery(ctx, query, emailHash)
}

func (r *userRepo) GetUserByPhone(ctx context.Context, phone string) (*biz.User, error) {
	phoneHash := r.enc.Hash([]byte(phone))
	query := `
		SELECT id, username, password, 
			email_encrypted, phone_encrypted, name_encrypted,
			created_at, updated_at 
		FROM users 
		WHERE phone_hash = $1
	`
	return r.getUserByQuery(ctx, query, phoneHash)
}

func (r *userRepo) GetUserByName(ctx context.Context, name string) (*biz.User, error) {
	nameHash := r.enc.Hash([]byte(name))
	query := `
		SELECT id, username, password, 
			email_encrypted, phone_encrypted, name_encrypted,
			created_at, updated_at 
		FROM users 
		WHERE name_hash = $1
	`
	return r.getUserByQuery(ctx, query, nameHash)
}

func (r *userRepo) getUserByQuery(ctx context.Context, query string, param string) (*biz.User, error) {
	user := &biz.User{}
	var emailEnc, phoneEnc, nameEnc []byte
	err := r.data.db.QueryRowContext(ctx, query, param).Scan(
		&user.ID, &user.Username, &user.Password,
		&emailEnc, &phoneEnc, &nameEnc,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, biz.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	// 解密敏感信息
	if emailEnc != nil {
		email, err := r.enc.Decrypt(emailEnc)
		if err != nil {
			return nil, err
		}
		user.Email = string(email)
	}

	if phoneEnc != nil {
		phone, err := r.enc.Decrypt(phoneEnc)
		if err != nil {
			return nil, err
		}
		user.Phone = string(phone)
	}

	if nameEnc != nil {
		name, err := r.enc.Decrypt(nameEnc)
		if err != nil {
			return nil, err
		}
		user.Name = string(name)
	}

	return user, nil
}

func (r *userRepo) UpdateUser(ctx context.Context, u *biz.User) error {
	// 加密敏感信息
	emailEnc, err := r.enc.Encrypt([]byte(u.Email))
	if err != nil {
		return err
	}
	emailHash := r.enc.Hash([]byte(u.Email))

	phoneEnc, err := r.enc.Encrypt([]byte(u.Phone))
	if err != nil {
		return err
	}
	phoneHash := r.enc.Hash([]byte(u.Phone))

	nameEnc, err := r.enc.Encrypt([]byte(u.Name))
	if err != nil {
		return err
	}
	nameHash := r.enc.Hash([]byte(u.Name))

	query := `
		UPDATE users SET 
			email_encrypted = $1, email_hash = $2,
			phone_encrypted = $3, phone_hash = $4,
			name_encrypted = $5, name_hash = $6,
			updated_at = $7
		WHERE id = $8
	`
	_, err = r.data.db.ExecContext(ctx, query,
		emailEnc, emailHash,
		phoneEnc, phoneHash,
		nameEnc, nameHash,
		time.Now(),
		u.ID,
	)
	return err
}

// 验证码相关方法
func (r *userRepo) SaveCaptcha(ctx context.Context, captcha *biz.Captcha) error {
	r.captchas.Store(captcha.ID, captcha)
	return nil
}

func (r *userRepo) GetCaptcha(ctx context.Context, captchaID string) (*biz.Captcha, error) {
	value, ok := r.captchas.Load(captchaID)
	if !ok {
		return nil, fmt.Errorf("验证码不存在")
	}

	captcha, ok := value.(*biz.Captcha)
	if !ok {
		return nil, fmt.Errorf("验证码类型错误")
	}

	return captcha, nil
}

func (r *userRepo) MarkCaptchaUsed(ctx context.Context, captchaID string) error {
	value, ok := r.captchas.Load(captchaID)
	if !ok {
		return fmt.Errorf("验证码不存在")
	}

	captcha, ok := value.(*biz.Captcha)
	if !ok {
		return fmt.Errorf("验证码类型错误")
	}

	captcha.Used = true
	r.captchas.Store(captchaID, captcha)

	return nil
}

// 账户锁定相关方法
func (r *userRepo) GetLock(ctx context.Context, username string) (*biz.AccountLock, error) {
	value, ok := r.accountLocks.Load(username)
	if !ok {
		return nil, biz.ErrUserNotFound
	}

	lock, ok := value.(*biz.AccountLock)
	if !ok {
		return nil, fmt.Errorf("账户锁定信息类型错误")
	}

	return lock, nil
}

func (r *userRepo) SaveLock(ctx context.Context, lock *biz.AccountLock) error {
	r.accountLocks.Store(lock.Username, lock)
	return nil
}

func (r *userRepo) RemoveLock(ctx context.Context, username string) error {
	r.accountLocks.Delete(username)
	return nil
}

// 刷新令牌相关方法
func (r *userRepo) SaveRefreshToken(ctx context.Context, username, tokenID string, expiresAt time.Time) error {
	r.refreshTokens.Store(tokenID, &refreshTokenInfo{
		username: username,
		used:     false,
		expireAt: expiresAt,
	})
	return nil
}

func (r *userRepo) GetRefreshToken(ctx context.Context, tokenID string) (string, bool, error) {
	value, ok := r.refreshTokens.Load(tokenID)
	if !ok {
		return "", false, fmt.Errorf("刷新令牌不存在")
	}

	info, ok := value.(*refreshTokenInfo)
	if !ok {
		return "", false, fmt.Errorf("刷新令牌信息类型错误")
	}

	// 检查令牌是否过期
	if info.expireAt.Before(time.Now()) {
		r.refreshTokens.Delete(tokenID)
		return "", false, fmt.Errorf("刷新令牌已过期")
	}

	// 返回用户名和令牌是否已被使用
	return info.username, info.used, nil
}

func (r *userRepo) InvalidateRefreshToken(ctx context.Context, tokenID string) error {
	value, ok := r.refreshTokens.Load(tokenID)
	if !ok {
		return fmt.Errorf("刷新令牌不存在")
	}

	info, ok := value.(*refreshTokenInfo)
	if !ok {
		return fmt.Errorf("刷新令牌信息类型错误")
	}

	// 标记为已使用
	info.used = true
	r.refreshTokens.Store(tokenID, info)

	return nil
}

func (r *userRepo) InvalidateAllRefreshTokens(ctx context.Context, username string) error {
	// 遍历所有令牌，将该用户的所有令牌标记为已使用
	r.refreshTokens.Range(func(key, value interface{}) bool {
		info, ok := value.(*refreshTokenInfo)
		if ok && info.username == username {
			info.used = true
			r.refreshTokens.Store(key, info)
		}
		return true
	})

	return nil
}
