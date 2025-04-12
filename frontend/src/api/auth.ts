import axios from 'axios';
import type { ApiResponse, CaptchaResponse, LoginResponse, RegisterResponse, LockStatusResponse } from '@/types/api';

// 创建axios实例
const request = axios.create({
    baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
    timeout: 10000,
    headers: {
        'Content-Type': 'application/json',
    },
});

// 请求拦截器
request.interceptors.request.use(
    (config) => {
        const token = localStorage.getItem('access_token');
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => {
        return Promise.reject(error);
    }
);

// 响应拦截器
request.interceptors.response.use(
    (response) => {
        return response.data;
    },
    (error) => {
        if (error.response?.status === 401) {
            // 处理token过期
            localStorage.removeItem('access_token');
            localStorage.removeItem('refresh_token');
            window.location.href = '/login';
        }
        return Promise.reject(error);
    }
);

// 获取验证码
export const getCaptcha = (type: string, target?: string) => {
    return request.get<ApiResponse<CaptchaResponse>>('/v1/auth/captcha', {
        params: {
            captcha_type: type,
            target,
        },
    });
};

// 验证验证码
export const verifyCaptcha = (captchaId: string, captchaCode: string) => {
    return request.post<ApiResponse<{ success: boolean }>>('/v1/auth/captcha/verify', {
        captcha_id: captchaId,
        captcha_code: captchaCode,
    });
};

// 用户注册
export const register = (data: {
    username: string;
    password: string;
    email: string;
    phone: string;
    captchaId: string;
    captchaCode: string;
}) => {
    return request.post<ApiResponse<RegisterResponse>>('/v1/auth/register', data);
};

// 用户登录
export const login = (data: {
    username: string;
    password: string;
    captchaId: string;
    captchaCode: string;
    totpCode?: string;
}) => {
    return request.post<ApiResponse<LoginResponse>>('/v1/auth/login', data);
};

// 退出登录
export const logout = () => {
    return request.post<ApiResponse<{ success: boolean }>>('/v1/auth/logout');
};

// 刷新令牌
export const refreshToken = (refreshToken: string) => {
    return request.post<ApiResponse<LoginResponse>>('/v1/auth/refresh', {
        refresh_token: refreshToken,
    });
};

// 查询账户锁定状态
export const getLockStatus = (username: string) => {
    return request.get<ApiResponse<LockStatusResponse>>(`/v1/auth/lock-status/${username}`);
}; 