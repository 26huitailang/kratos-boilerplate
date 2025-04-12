// 验证码响应
export interface CaptchaResponse {
    captcha_id: string;
    image_data: string;
}

// 登录响应
export interface LoginResponse {
    access_token: string;
    refresh_token: string;
    expires_in: number;
}

// 注册响应
export interface RegisterResponse {
    message: string;
}

// 账户锁定状态响应
export interface LockStatusResponse {
    locked: boolean;
    unlock_time: number;
    failed_attempts: number;
    max_attempts: number;
}

// 通用响应
export interface ApiResponse<T> {
    code: number;
    message: string;
    data: T;
} 