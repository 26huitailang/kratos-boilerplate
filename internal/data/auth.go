package data

import (
	"context"
	"database/sql"
	"time"

	"trae-demo/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type userRepo struct {
	data *Data
	log  *log.Helper
}

// NewUserRepo .
func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *userRepo) CreateUser(ctx context.Context, u *biz.User) error {
	query := `INSERT INTO users (username, password, created_at, updated_at) VALUES ($1, $2, $3, $4)`
	_, err := r.data.db.ExecContext(ctx, query, u.Username, u.Password, time.Now(), time.Now())
	return err
}

func (r *userRepo) GetUser(ctx context.Context, username string) (*biz.User, error) {
	user := &biz.User{}
	query := `SELECT id, username, password, created_at, updated_at FROM users WHERE username = $1`
	row := r.data.db.QueryRowContext(ctx, query, username)

	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, biz.ErrUserNotFound
	}
	return user, err
}

// GetUserByUsername 通过用户名获取用户（兼容旧接口）
func (r *userRepo) GetUserByUsername(username string) (*biz.User, error) {
	return r.GetUser(context.Background(), username)
}
