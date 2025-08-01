package helpers

import (
	"fmt"
	"math/rand"
	"time"
)

// TestUser 表示测试用户
type TestUser struct {
	Username     string
	Password     string
	Email        string
	Phone        string
	Name         string
	AccessToken  string
	RefreshToken string
}

// NewTestUser 创建新的测试用户
func NewTestUser() *TestUser {
	suffix := generateRandomString(8)
	return &TestUser{
		Username: fmt.Sprintf("testuser_%s", suffix),
		Password: "TestPassword123!",
		Email:    fmt.Sprintf("testuser_%s@example.com", suffix),
		Phone:    fmt.Sprintf("138%s", generateRandomNumbers(8)),
		Name:     fmt.Sprintf("Test User %s", suffix),
	}
}

// NewTestUserWithPrefix 创建带前缀的测试用户
func NewTestUserWithPrefix(prefix string) *TestUser {
	suffix := generateRandomString(8)
	return &TestUser{
		Username: fmt.Sprintf("%s_%s", prefix, suffix),
		Password: "TestPassword123!",
		Email:    fmt.Sprintf("%s_%s@example.com", prefix, suffix),
		Phone:    fmt.Sprintf("138%s", generateRandomNumbers(8)),
		Name:     fmt.Sprintf("Test User %s %s", prefix, suffix),
	}
}

// Clone 克隆测试用户（用于创建相似但不同的用户）
func (u *TestUser) Clone() *TestUser {
	suffix := generateRandomString(8)
	return &TestUser{
		Username: fmt.Sprintf("%s_%s", u.Username, suffix),
		Password: u.Password,
		Email:    fmt.Sprintf("%s_%s@example.com", u.Username, suffix),
		Phone:    fmt.Sprintf("138%s", generateRandomNumbers(8)),
		Name:     fmt.Sprintf("%s Clone %s", u.Name, suffix),
	}
}

// WithPassword 设置密码
func (u *TestUser) WithPassword(password string) *TestUser {
	u.Password = password
	return u
}

// WithEmail 设置邮箱
func (u *TestUser) WithEmail(email string) *TestUser {
	u.Email = email
	return u
}

// WithPhone 设置手机号
func (u *TestUser) WithPhone(phone string) *TestUser {
	u.Phone = phone
	return u
}

// IsLoggedIn 检查用户是否已登录（有访问令牌）
func (u *TestUser) IsLoggedIn() bool {
	return u.AccessToken != ""
}

// ClearTokens 清除令牌
func (u *TestUser) ClearTokens() {
	u.AccessToken = ""
	u.RefreshToken = ""
}

// TestUserManager 管理测试用户
type TestUserManager struct {
	users []*TestUser
}

// NewTestUserManager 创建测试用户管理器
func NewTestUserManager() *TestUserManager {
	return &TestUserManager{
		users: make([]*TestUser, 0),
	}
}

// CreateUser 创建并管理测试用户
func (m *TestUserManager) CreateUser() *TestUser {
	user := NewTestUser()
	m.users = append(m.users, user)
	return user
}

// CreateUserWithPrefix 创建带前缀的测试用户
func (m *TestUserManager) CreateUserWithPrefix(prefix string) *TestUser {
	user := NewTestUserWithPrefix(prefix)
	m.users = append(m.users, user)
	return user
}

// GetUsers 获取所有管理的用户
func (m *TestUserManager) GetUsers() []*TestUser {
	return m.users
}

// CleanupAll 清理所有用户
func (m *TestUserManager) CleanupAll() {
	m.users = make([]*TestUser, 0)
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// generateRandomNumbers 生成随机数字字符串
func generateRandomNumbers(length int) string {
	const charset = "0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// ValidTestPasswords 有效的测试密码列表
var ValidTestPasswords = []string{
	"TestPassword123!",
	"SecurePass456@",
	"MyPassword789#",
	"StrongPass012$",
}

// InvalidTestPasswords 无效的测试密码列表
var InvalidTestPasswords = []string{
	"123",           // 太短
	"password",      // 没有大写字母和数字
	"PASSWORD123",   // 没有小写字母
	"Password",      // 没有数字
	"password123",   // 没有大写字母
}

// GetValidPassword 获取随机有效密码
func GetValidPassword() string {
	rand.Seed(time.Now().UnixNano())
	return ValidTestPasswords[rand.Intn(len(ValidTestPasswords))]
}

// GetInvalidPassword 获取随机无效密码
func GetInvalidPassword() string {
	rand.Seed(time.Now().UnixNano())
	return InvalidTestPasswords[rand.Intn(len(InvalidTestPasswords))]
}