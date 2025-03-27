package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserExists        = errors.New("user already exists")
	ErrPasswordIncorrect = errors.New("password incorrect")
)

// User 用户模型
type User struct {
	ID        int64
	Username  string
	Password  string
	Email     string
	Phone     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserInfo 用户信息（匿名化后的）
type UserInfo struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"` // 匿名化的邮箱
	Phone     string    `json:"phone"` // 匿名化的手机号
	Name      string    `json:"name"`  // 匿名化的姓名
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserRepo 用户仓储接口
type UserRepo interface {
	CreateUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, username string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByPhone(ctx context.Context, phone string) (*User, error)
	GetUserByName(ctx context.Context, name string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
}

type AuthUsecase struct {
	repo UserRepo
	log  *log.Helper
}

func NewAuthUsecase(repo UserRepo, logger log.Logger) *AuthUsecase {
	return &AuthUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *AuthUsecase) Register(ctx context.Context, username, password string) error {
	// 1. 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %v", err)
	}

	// 2. 创建用户
	if err := uc.repo.CreateUser(ctx, &User{
		Username:  username,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		return fmt.Errorf("创建用户失败: %v", err)
	}

	return nil
}

func (uc *AuthUsecase) Login(ctx context.Context, username, password string) (string, error) {
	// 1. 获取用户
	user, err := uc.repo.GetUser(ctx, username)
	if err != nil {
		return "", fmt.Errorf("用户不存在: %v", err)
	}

	// 2. 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", fmt.Errorf("密码错误")
	}

	// 3. 生成 JWT token
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // 24小时过期
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 这里使用一个简单的密钥，实际应用中应该从配置中读取
	signedToken, err := token.SignedString([]byte("your-secret-key"))
	if err != nil {
		return "", fmt.Errorf("生成token失败: %v", err)
	}

	return signedToken, nil
}
