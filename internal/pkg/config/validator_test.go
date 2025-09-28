package config

import (
	"os"
	"testing"
	"time"

	"kratos-boilerplate/internal/conf"

	"google.golang.org/protobuf/types/known/durationpb"
)

func TestConfigValidator_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *conf.Bootstrap
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "valid development config",
			config: &conf.Bootstrap{
				Server: &conf.Server{
					Http: &conf.Server_HTTP{
						Addr:    "0.0.0.0:8000",
						Timeout: durationpb.New(time.Second),
					},
					Grpc: &conf.Server_GRPC{
						Addr:    "0.0.0.0:9000",
						Timeout: durationpb.New(time.Second),
					},
				},
				Data: &conf.Data{
					Database: &conf.Data_Database{
						Driver: "postgres",
						Source: "postgresql://user:pass@localhost:5432/testdb?sslmode=disable",
					},
					Redis: &conf.Data_Redis{
						Addr:         "127.0.0.1:6379",
						ReadTimeout:  durationpb.New(time.Second),
						WriteTimeout: durationpb.New(time.Second),
					},
				},
				Auth: &conf.Auth{
					JwtSecretKey:           "dev-jwt-secret-key-change-in-production",
					AccessTokenExpiration:  durationpb.New(24 * time.Hour),
					RefreshTokenExpiration: durationpb.New(7 * 24 * time.Hour),
					CaptchaEnabled:         true,
					CaptchaExpiration:      durationpb.New(5 * time.Minute),
					MaxLoginAttempts:       5,
					LockDuration:           durationpb.New(30 * time.Minute),
					TotpEnabled:            false,
				},
			},
			envVars: map[string]string{
				"ENVIRONMENT": "development",
			},
			wantErr: false,
		},
		{
			name: "invalid config - missing server",
			config: &conf.Bootstrap{
				Data: &conf.Data{
					Database: &conf.Data_Database{
						Driver: "postgres",
						Source: "postgresql://user:pass@localhost:5432/testdb",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid config - weak JWT secret in production",
			config: &conf.Bootstrap{
				Server: &conf.Server{
					Http: &conf.Server_HTTP{
						Addr:    "0.0.0.0:8000",
						Timeout: durationpb.New(time.Second),
					},
					Grpc: &conf.Server_GRPC{
						Addr:    "0.0.0.0:9000",
						Timeout: durationpb.New(time.Second),
					},
				},
				Data: &conf.Data{
					Database: &conf.Data_Database{
						Driver: "postgres",
						Source: "postgresql://user:pass@prod-db:5432/testdb?sslmode=require",
					},
					Redis: &conf.Data_Redis{
						Addr:         "redis-cluster:6379",
						ReadTimeout:  durationpb.New(time.Second),
						WriteTimeout: durationpb.New(time.Second),
					},
				},
				Auth: &conf.Auth{
					JwtSecretKey:           "weak",
					AccessTokenExpiration:  durationpb.New(time.Hour),
					RefreshTokenExpiration: durationpb.New(7 * 24 * time.Hour),
					CaptchaEnabled:         true,
					CaptchaExpiration:      durationpb.New(5 * time.Minute),
					MaxLoginAttempts:       5,
					LockDuration:           durationpb.New(30 * time.Minute),
					TotpEnabled:            true,
				},
			},
			envVars: map[string]string{
				"ENVIRONMENT":    "production",
				"REDIS_PASSWORD": "redis-secret",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置环境变量
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			validator := NewConfigValidator(tt.config)
			err := validator.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateJWTSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		envVars map[string]string
		wantErr bool
	}{
		{
			name:   "valid development secret",
			secret: "dev-jwt-secret-key",
			envVars: map[string]string{
				"ENVIRONMENT": "development",
			},
			wantErr: false,
		},
		{
			name:   "valid production secret",
			secret: "Very-Strong-Production-JWT-Secret-Key-With-Numbers-123-And-Symbols!@#",
			envVars: map[string]string{
				"ENVIRONMENT": "production",
			},
			wantErr: false,
		},
		{
			name:   "invalid production secret - too short",
			secret: "short",
			envVars: map[string]string{
				"ENVIRONMENT": "production",
			},
			wantErr: true,
		},
		{
			name:   "invalid production secret - default key",
			secret: "your-super-secret-jwt-key-here",
			envVars: map[string]string{
				"ENVIRONMENT": "production",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置环境变量
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			validator := &ConfigValidator{}
			err := validator.validateJWTSecret(tt.secret)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateJWTSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsComplexSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		want   bool
	}{
		{
			name:   "complex secret",
			secret: "MySecretKey123!",
			want:   true,
		},
		{
			name:   "simple secret - only lowercase",
			secret: "mysecretkey",
			want:   false,
		},
		{
			name:   "medium complexity - missing special chars",
			secret: "MySecretKey123",
			want:   true,
		},
		{
			name:   "very complex secret",
			secret: "MyVerySecretKey123!@#$%",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isComplexSecret(tt.secret); got != tt.want {
				t.Errorf("isComplexSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}
