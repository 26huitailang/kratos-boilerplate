package service

import (
	"context"

	v1 "xredline/api/auth/v1"
	"xredline/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type AuthService struct {
	v1.UnimplementedAuthServer

	uc  *biz.AuthUsecase
	log *log.Helper
}

func NewAuthService(uc *biz.AuthUsecase, logger log.Logger) *AuthService {
	return &AuthService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *AuthService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterReply, error) {
	err := s.uc.Register(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	return &v1.RegisterReply{Message: "注册成功"}, nil
}

func (s *AuthService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginReply, error) {
	token, err := s.uc.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	return &v1.LoginReply{Token: token}, nil
}
