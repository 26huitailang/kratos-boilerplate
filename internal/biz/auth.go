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
	ErrPasswordIncorrect = errors.New("password incorrect")
	ErrUserExists        = errors.New("user already exists")
)

type User struct {
	ID        int64
	Username  string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserRepo interface {
	CreateUser(context.Context, *User) error
	GetUser(context.Context, string) (*User, error)
	GetUserByUsername(string) (*User, error)
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
	user, err := uc.repo.GetUserByUsername(username)
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
