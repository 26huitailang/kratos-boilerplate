package service

import (
	"context"

	v1 "xredline/api/auth/v1"
	"xredline/internal/biz"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/metadata"
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

// 获取验证码
func (s *AuthService) GetCaptcha(ctx context.Context, req *v1.GetCaptchaRequest) (*v1.GetCaptchaReply, error) {
	captcha, err := s.uc.GetCaptcha(ctx, req.CaptchaType, req.Target)
	if err != nil {
		return nil, errors.InternalServer("CAPTCHA_ERROR", err.Error())
	}

	return &v1.GetCaptchaReply{
		CaptchaId: captcha.ID,
		ImageData: captcha.Code, // 对于图片验证码，这里会返回base64编码的图片数据
	}, nil
}

// 验证验证码
func (s *AuthService) VerifyCaptcha(ctx context.Context, req *v1.VerifyCaptchaRequest) (*v1.VerifyCaptchaReply, error) {
	valid, err := s.uc.VerifyCaptcha(ctx, req.CaptchaId, req.CaptchaCode)
	if err != nil {
		switch err {
		case biz.ErrCaptchaExpired:
			return nil, errors.BadRequest("CAPTCHA_EXPIRED", err.Error())
		case biz.ErrCaptchaInvalid:
			return nil, errors.BadRequest("CAPTCHA_INVALID", err.Error())
		default:
			return nil, errors.InternalServer("CAPTCHA_ERROR", err.Error())
		}
	}

	return &v1.VerifyCaptchaReply{
		Success: valid,
	}, nil
}

// 用户注册
func (s *AuthService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterReply, error) {
	err := s.uc.Register(ctx, req.Username, req.Password, req.Email, req.Phone, req.CaptchaId, req.CaptchaCode)
	if err != nil {
		switch err {
		case biz.ErrUserExists:
			return nil, errors.BadRequest("USER_EXISTS", "用户名已存在")
		case biz.ErrCaptchaRequired:
			return nil, errors.BadRequest("CAPTCHA_REQUIRED", "验证码必填")
		case biz.ErrCaptchaInvalid:
			return nil, errors.BadRequest("CAPTCHA_INVALID", "验证码无效")
		case biz.ErrCaptchaExpired:
			return nil, errors.BadRequest("CAPTCHA_EXPIRED", "验证码已过期")
		default:
			return nil, errors.InternalServer("REGISTER_ERROR", err.Error())
		}
	}
	return &v1.RegisterReply{Message: "注册成功"}, nil
}

// 用户登录
func (s *AuthService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginReply, error) {
	tokenPair, err := s.uc.Login(ctx, req.Username, req.Password, req.CaptchaId, req.CaptchaCode, req.TotpCode)
	if err != nil {
		switch err {
		case biz.ErrUserNotFound:
			return nil, errors.NotFound("USER_NOT_FOUND", "用户不存在")
		case biz.ErrPasswordIncorrect:
			return nil, errors.Unauthorized("PASSWORD_INCORRECT", "密码错误")
		case biz.ErrCaptchaRequired:
			return nil, errors.BadRequest("CAPTCHA_REQUIRED", "验证码必填")
		case biz.ErrCaptchaInvalid:
			return nil, errors.BadRequest("CAPTCHA_INVALID", "验证码无效")
		case biz.ErrCaptchaExpired:
			return nil, errors.BadRequest("CAPTCHA_EXPIRED", "验证码已过期")
		case biz.ErrAccountLocked:
			return nil, errors.Forbidden("ACCOUNT_LOCKED", "账户已锁定")
		case biz.ErrTotpRequired:
			return nil, errors.BadRequest("TOTP_REQUIRED", "需要TOTP验证码")
		case biz.ErrTotpCodeInvalid:
			return nil, errors.BadRequest("TOTP_INVALID", "TOTP验证码无效")
		default:
			return nil, errors.InternalServer("LOGIN_ERROR", err.Error())
		}
	}

	return &v1.LoginReply{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// 退出登录
func (s *AuthService) Logout(ctx context.Context, req *v1.LogoutRequest) (*v1.LogoutReply, error) {
	// 从请求头中获取访问令牌
	md, ok := metadata.FromServerContext(ctx)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "未授权访问")
	}

	authorization := md.Get("Authorization")
	if authorization == "" {
		return nil, errors.Unauthorized("TOKEN_MISSING", "缺少访问令牌")
	}

	// 检查授权头格式
	if len(authorization) <= 7 || authorization[:7] != "Bearer " {
		return nil, errors.Unauthorized("INVALID_TOKEN_FORMAT", "访问令牌格式错误")
	}

	// 提取令牌
	token := authorization[7:]

	// 调用业务逻辑执行退出
	if err := s.uc.Logout(ctx, token); err != nil {
		switch err {
		case biz.ErrTokenInvalid:
			return nil, errors.Unauthorized("TOKEN_INVALID", "访问令牌无效")
		case biz.ErrTokenExpired:
			return nil, errors.Unauthorized("TOKEN_EXPIRED", "访问令牌已过期")
		default:
			return nil, errors.InternalServer("LOGOUT_ERROR", err.Error())
		}
	}

	return &v1.LogoutReply{
		Success: true,
	}, nil
}

// 刷新令牌
func (s *AuthService) RefreshToken(ctx context.Context, req *v1.RefreshTokenRequest) (*v1.RefreshTokenReply, error) {
	tokenPair, err := s.uc.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		switch err {
		case biz.ErrRefreshTokenInvalid:
			return nil, errors.Unauthorized("REFRESH_TOKEN_INVALID", "刷新令牌无效")
		case biz.ErrTokenExpired:
			return nil, errors.Unauthorized("REFRESH_TOKEN_EXPIRED", "刷新令牌已过期")
		case biz.ErrRefreshTokenReused:
			return nil, errors.Unauthorized("REFRESH_TOKEN_REUSED", "刷新令牌已被使用，可能存在安全风险")
		default:
			return nil, errors.InternalServer("REFRESH_ERROR", err.Error())
		}
	}

	return &v1.RefreshTokenReply{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// 查询账户锁定状态
func (s *AuthService) LockStatus(ctx context.Context, req *v1.LockStatusRequest) (*v1.LockStatusReply, error) {
	lock, err := s.uc.GetLockStatus(ctx, req.Username)
	if err != nil {
		return nil, errors.InternalServer("LOCK_STATUS_ERROR", err.Error())
	}

	var unlockTime int64
	if !lock.LockUntil.IsZero() {
		unlockTime = lock.LockUntil.Unix()
	}

	return &v1.LockStatusReply{
		Locked:         lock.LockUntil.After(s.uc.Now()),
		UnlockTime:     unlockTime,
		FailedAttempts: lock.FailedAttempts,
		MaxAttempts:    s.uc.GetMaxLoginAttempts(),
	}, nil
}
