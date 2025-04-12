<template>
  <div class="login-container">
    <el-card class="login-card">
      <template #header>
        <h2>用户登录</h2>
      </template>
      
      <el-form
        ref="loginFormRef"
        :model="loginForm"
        :rules="loginRules"
        label-width="80px"
        @keyup.enter="handleLogin"
      >
        <el-form-item label="用户名" prop="username">
          <el-input
            v-model="loginForm.username"
            placeholder="请输入用户名"
            :prefix-icon="User"
          />
        </el-form-item>

        <el-form-item label="密码" prop="password">
          <el-input
            v-model="loginForm.password"
            type="password"
            placeholder="请输入密码"
            :prefix-icon="Lock"
            show-password
          />
        </el-form-item>

        <el-form-item label="验证码" prop="captchaCode">
          <div class="captcha-container">
            <el-input
              v-model="loginForm.captchaCode"
              placeholder="请输入验证码"
              :prefix-icon="Key"
            />
            <img
              v-if="captchaImage"
              :src="captchaImage"
              class="captcha-image"
              @click="refreshCaptcha"
              alt="验证码"
            />
          </div>
        </el-form-item>

        <el-form-item v-if="showTotp" label="TOTP" prop="totpCode">
          <el-input
            v-model="loginForm.totpCode"
            placeholder="请输入TOTP验证码"
            :prefix-icon="Lock"
          />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleLogin" :loading="loading">
            登录
          </el-button>
          <el-button @click="handleRegister">注册</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import { User, Lock, Key } from '@element-plus/icons-vue';
import { useAuthStore } from '@/stores/auth';
import { getCaptcha } from '@/api/auth';

const router = useRouter();
const authStore = useAuthStore();
const loginFormRef = ref();
const loading = ref(false);
const showTotp = ref(false);
const captchaImage = ref('');
const captchaId = ref('');

const loginForm = reactive({
  username: '',
  password: '',
  captchaCode: '',
  totpCode: '',
});

const loginRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '长度在 3 到 20 个字符', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, message: '密码长度不能小于8位', trigger: 'blur' },
  ],
  captchaCode: [
    { required: true, message: '请输入验证码', trigger: 'blur' },
  ],
  totpCode: [
    { required: true, message: '请输入TOTP验证码', trigger: 'blur' },
  ],
};

// 获取验证码
const refreshCaptcha = async () => {
  try {
    const response = await getCaptcha('image');
    captchaId.value = response.data.captcha_id;
    captchaImage.value = `data:image/png;base64,${response.data.image_data}`;
  } catch (error: any) {
    console.error('验证码获取失败:', error);
    const errorMessage = error.response?.data?.message || error.message || '获取验证码失败';
    ElMessage.error(`验证码获取失败: ${errorMessage}`);
  }
};

// 处理登录
const handleLogin = async () => {
  if (!loginFormRef.value) return;
  
  await loginFormRef.value.validate(async (valid: boolean) => {
    if (valid) {
      loading.value = true;
      try {
        await authStore.loginAction({
          username: loginForm.username,
          password: loginForm.password,
          captchaId: captchaId.value,
          captchaCode: loginForm.captchaCode,
          totpCode: loginForm.totpCode,
        });
        ElMessage.success('登录成功');
        router.push('/');
      } catch (error: any) {
        if (error.response?.data?.message === 'TOTP_REQUIRED') {
          showTotp.value = true;
          ElMessage.warning('请输入TOTP验证码');
        } else {
          ElMessage.error(error.response?.data?.message || '登录失败');
          refreshCaptcha();
        }
      } finally {
        loading.value = false;
      }
    }
  });
};

// 处理注册
const handleRegister = () => {
  router.push('/register');
};

onMounted(() => {
  refreshCaptcha();
});
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background-color: #f5f7fa;
}

.login-card {
  width: 400px;
}

.captcha-container {
  display: flex;
  gap: 10px;
}

.captcha-image {
  height: 40px;
  cursor: pointer;
}
</style> 