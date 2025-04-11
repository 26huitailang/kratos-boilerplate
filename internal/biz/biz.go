package biz

import (
	"time"
	"xredline/internal/conf"

	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewGreeterUsecase, NewAuthUsecase, NewAuthConfig)

// NewAuthConfig creates a new AuthConfig from conf.Auth
func NewAuthConfig(auth *conf.Auth) AuthConfig {
	return AuthConfig{
		JWTSecretKey:           auth.JwtSecretKey,
		AccessTokenExpiration:  auth.AccessTokenExpiration.AsDuration(),
		RefreshTokenExpiration: auth.RefreshTokenExpiration.AsDuration(),
		CaptchaEnabled:         auth.CaptchaEnabled,
		CaptchaExpiration:      auth.CaptchaExpiration.AsDuration(),
		MaxLoginAttempts:       auth.MaxLoginAttempts,
		LockDuration:           time.Minute * 30, // 默认锁定30分钟
		TOTPEnabled:            false,            // 默认不启用TOTP
	}
}
