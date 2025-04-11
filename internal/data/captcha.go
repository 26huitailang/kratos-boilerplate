package data

import (
	"context"
	"time"
	"xredline/internal/biz"
)

type captchaRepo struct {
	data *Data
}

func NewCaptchaRepo(data *Data) biz.CaptchaRepo {
	return &captchaRepo{data: data}
}

func (r *captchaRepo) SaveCaptcha(ctx context.Context, captcha *biz.Captcha) error {
	// 使用 Redis 存储验证码
	key := "captcha:" + captcha.ID
	return r.data.redis.Set(ctx, key, captcha, time.Until(captcha.ExpireAt)).Err()
}

func (r *captchaRepo) GetCaptcha(ctx context.Context, id string) (*biz.Captcha, error) {
	key := "captcha:" + id
	var captcha biz.Captcha
	err := r.data.redis.Get(ctx, key).Scan(&captcha)
	if err != nil {
		return nil, err
	}
	return &captcha, nil
}

func (r *captchaRepo) DeleteCaptcha(ctx context.Context, id string) error {
	key := "captcha:" + id
	return r.data.redis.Del(ctx, key).Err()
}

func (r *captchaRepo) MarkCaptchaUsed(ctx context.Context, id string) error {
	captcha, err := r.GetCaptcha(ctx, id)
	if err != nil {
		return err
	}
	captcha.Used = true
	return r.SaveCaptcha(ctx, captcha)
}
