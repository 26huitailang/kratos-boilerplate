import { defineStore } from 'pinia';
import { login, logout, refreshToken, getLockStatus } from '@/api/auth';
import type { LoginResponse, LockStatusResponse } from '@/types/api';
import { ref } from 'vue';

export const useAuthStore = defineStore('auth', () => {
    const accessToken = ref<string | null>(localStorage.getItem('access_token'));
    const refreshTokenValue = ref<string | null>(localStorage.getItem('refresh_token'));
    const isAuthenticated = ref<boolean>(!!accessToken.value);

    // 登录
    const loginAction = async (data: {
        username: string;
        password: string;
        captchaId: string;
        captchaCode: string;
        totpCode?: string;
    }) => {
        try {
            const response = await login(data);
            const { access_token, refresh_token } = response.data;
            accessToken.value = access_token;
            refreshTokenValue.value = refresh_token;
            isAuthenticated.value = true;

            // 存储token
            localStorage.setItem('access_token', access_token);
            localStorage.setItem('refresh_token', refresh_token);

            return response;
        } catch (error) {
            throw error;
        }
    };

    // 退出登录
    const logoutAction = async () => {
        try {
            await logout();
            accessToken.value = null;
            refreshTokenValue.value = null;
            isAuthenticated.value = false;

            // 清除token
            localStorage.removeItem('access_token');
            localStorage.removeItem('refresh_token');
        } catch (error) {
            throw error;
        }
    };

    // 刷新token
    const refreshTokenAction = async () => {
        if (!refreshTokenValue.value) {
            throw new Error('No refresh token available');
        }

        try {
            const response = await refreshToken(refreshTokenValue.value);
            const { access_token, refresh_token } = response.data;
            accessToken.value = access_token;
            refreshTokenValue.value = refresh_token;

            // 更新存储的token
            localStorage.setItem('access_token', access_token);
            localStorage.setItem('refresh_token', refresh_token);

            return response;
        } catch (error) {
            // 刷新失败，清除token
            accessToken.value = null;
            refreshTokenValue.value = null;
            isAuthenticated.value = false;
            localStorage.removeItem('access_token');
            localStorage.removeItem('refresh_token');
            throw error;
        }
    };

    // 检查账户锁定状态
    const checkLockStatus = async (username: string) => {
        try {
            const response = await getLockStatus(username);
            return response.data;
        } catch (error) {
            throw error;
        }
    };

    return {
        accessToken,
        refreshTokenValue,
        isAuthenticated,
        loginAction,
        logoutAction,
        refreshTokenAction,
        checkLockStatus,
    };
}); 