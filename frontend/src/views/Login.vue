<template>
  <div class="login-page">
    <!-- Background Elements -->
    <div class="background-elements">
      <div class="bg-circle bg-circle-1"></div>
      <div class="bg-circle bg-circle-2"></div>
      <div class="bg-circle bg-circle-3"></div>
    </div>

    <!-- Main Content -->
    <div class="login-container">
      <!-- Branding Section -->
      <div class="branding-section">
        <div class="brand-logo">
          <div class="logo-icon">
            <el-icon size="48"><User /></el-icon>
          </div>
          <h1 class="brand-title">Kratos Boilerplate</h1>
          <p class="brand-subtitle">A modern microservice platform built with Go and Vue.js</p>
        </div>
        
        <div class="feature-list">
          <div class="feature-item">
            <el-icon color="var(--primary-500)"><Check /></el-icon>
            <span>Secure Authentication</span>
          </div>
          <div class="feature-item">
            <el-icon color="var(--primary-500)"><Check /></el-icon>
            <span>Modern UI/UX</span>
          </div>
          <div class="feature-item">
            <el-icon color="var(--primary-500)"><Check /></el-icon>
            <span>Responsive Design</span>
          </div>
        </div>
      </div>

      <!-- Login Form Section -->
      <div class="form-section">
        <div class="form-container">
          <div class="form-header">
            <h2 class="form-title">Welcome Back</h2>
            <p class="form-subtitle">Sign in to your account to continue</p>
          </div>

          <el-form
            ref="loginFormRef"
            :model="loginForm"
            :rules="loginRules"
            class="login-form"
            @keyup.enter="handleLogin"
          >
            <!-- Username Field -->
            <el-form-item prop="username" class="form-item">
              <el-input
                v-model="loginForm.username"
                placeholder="Enter your username"
                :prefix-icon="User"
                size="large"
                class="form-input"
              />
            </el-form-item>

            <!-- Password Field -->
            <el-form-item prop="password" class="form-item">
              <el-input
                v-model="loginForm.password"
                type="password"
                placeholder="Enter your password"
                :prefix-icon="Lock"
                size="large"
                show-password
                class="form-input"
              />
            </el-form-item>

            <!-- Captcha Field -->
            <el-form-item prop="captchaCode" class="form-item">
              <div class="captcha-row">
                <el-input
                  v-model="loginForm.captchaCode"
                  placeholder="Enter captcha code"
                  :prefix-icon="Key"
                  size="large"
                  class="captcha-input"
                />
                <div class="captcha-image-container" @click="refreshCaptcha">
                  <img
                    v-if="captchaImage"
                    :src="captchaImage"
                    class="captcha-image"
                    alt="Captcha"
                  />
                  <div v-else class="captcha-placeholder">
                    <el-icon><Refresh /></el-icon>
                    <span>Click to load</span>
                  </div>
                </div>
              </div>
            </el-form-item>

            <!-- TOTP Field (conditional) -->
            <el-form-item v-if="showTotp" prop="totpCode" class="form-item">
              <el-input
                v-model="loginForm.totpCode"
                placeholder="Enter TOTP verification code"
                :prefix-icon="Shield"
                size="large"
                class="form-input"
              />
            </el-form-item>

            <!-- Remember Me -->
            <div class="form-options">
              <el-checkbox v-model="rememberMe" class="remember-checkbox">
                Remember me
              </el-checkbox>
              <el-link type="primary" class="forgot-link">
                Forgot password?
              </el-link>
            </div>

            <!-- Submit Button -->
            <el-form-item class="submit-item">
              <el-button
                type="primary"
                size="large"
                class="submit-button"
                :loading="loading"
                @click="handleLogin"
              >
                <template v-if="!loading">
                  <el-icon><Right /></el-icon>
                </template>
                {{ loading ? 'Signing In...' : 'Sign In' }}
              </el-button>
            </el-form-item>

            <!-- Register Link -->
            <div class="register-link">
              <span>Don't have an account?</span>
              <el-link type="primary" @click="handleRegister">
                Create one now
              </el-link>
            </div>
          </el-form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import { 
  User, 
  Lock, 
  Key, 
  Check, 
  Right, 
  Refresh, 
  Shield 
} from '@element-plus/icons-vue';
import { useAuthStore } from '@/stores/auth';
import { getCaptcha } from '@/api/auth';

const router = useRouter();
const authStore = useAuthStore();
const loginFormRef = ref();
const loading = ref(false);
const showTotp = ref(false);
const captchaImage = ref('');
const captchaId = ref('');
const rememberMe = ref(false);

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
.login-page {
  min-height: 100vh;
  position: relative;
  background: linear-gradient(135deg, var(--primary-50) 0%, var(--primary-100) 50%, var(--primary-200) 100%);
  overflow: hidden;
}

/* Background Elements */
.background-elements {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.bg-circle {
  position: absolute;
  border-radius: 50%;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.1), rgba(147, 197, 253, 0.1));
  animation: float 6s ease-in-out infinite;
}

.bg-circle-1 {
  width: 200px;
  height: 200px;
  top: 10%;
  left: 10%;
  animation-delay: 0s;
}

.bg-circle-2 {
  width: 150px;
  height: 150px;
  top: 60%;
  right: 10%;
  animation-delay: 2s;
}

.bg-circle-3 {
  width: 100px;
  height: 100px;
  bottom: 20%;
  left: 20%;
  animation-delay: 4s;
}

@keyframes float {
  0%, 100% {
    transform: translateY(0) rotate(0deg);
  }
  50% {
    transform: translateY(-20px) rotate(180deg);
  }
}

/* Main Container */
.login-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
  min-height: 100vh;
  max-width: 1400px;
  margin: 0 auto;
  position: relative;
  z-index: 1;
}

/* Branding Section */
.branding-section {
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: var(--space-12) var(--space-8);
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.1), rgba(255, 255, 255, 0.05));
  backdrop-filter: blur(10px);
}

.brand-logo {
  text-align: center;
  margin-bottom: var(--space-12);
}

.logo-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  background: linear-gradient(135deg, var(--primary-500), var(--primary-600));
  border-radius: var(--rounded-2xl);
  color: white;
  margin: 0 auto var(--space-6);
  box-shadow: var(--shadow-lg);
}

.brand-title {
  font-size: var(--text-4xl);
  font-weight: var(--font-bold);
  color: var(--gray-900);
  margin: 0 0 var(--space-4);
  letter-spacing: -0.025em;
}

.brand-subtitle {
  font-size: var(--text-lg);
  color: var(--gray-600);
  margin: 0;
  line-height: var(--leading-relaxed);
}

.feature-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.feature-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  font-size: var(--text-lg);
  color: var(--gray-700);
  font-weight: var(--font-medium);
}

/* Form Section */
.form-section {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-8);
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
}

.form-container {
  width: 100%;
  max-width: 400px;
}

.form-header {
  text-align: center;
  margin-bottom: var(--space-8);
}

.form-title {
  font-size: var(--text-3xl);
  font-weight: var(--font-bold);
  color: var(--gray-900);
  margin: 0 0 var(--space-2);
}

.form-subtitle {
  font-size: var(--text-base);
  color: var(--gray-600);
  margin: 0;
}

/* Form Styles */
.login-form {
  width: 100%;
}

.form-item {
  margin-bottom: var(--space-5);
}

.form-input :deep(.el-input__wrapper) {
  border-radius: var(--rounded-lg);
  box-shadow: var(--shadow-sm);
  transition: var(--transition-all);
}

.form-input :deep(.el-input__wrapper):hover {
  box-shadow: var(--shadow-md);
}

.form-input :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.2);
}

/* Captcha Styles */
.captcha-row {
  display: flex;
  gap: var(--space-3);
  align-items: flex-start;
}

.captcha-input {
  flex: 1;
}

.captcha-image-container {
  width: 120px;
  height: 40px;
  border-radius: var(--rounded-lg);
  overflow: hidden;
  cursor: pointer;
  transition: var(--transition-transform);
  border: 1px solid var(--gray-300);
}

.captcha-image-container:hover {
  transform: scale(1.05);
}

.captcha-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.captcha-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  background-color: var(--gray-100);
  color: var(--gray-500);
  font-size: var(--text-xs);
  gap: var(--space-1);
}

/* Form Options */
.form-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-6);
}

.remember-checkbox :deep(.el-checkbox__label) {
  color: var(--gray-600);
  font-size: var(--text-sm);
}

.forgot-link {
  font-size: var(--text-sm);
}

/* Submit Button */
.submit-item {
  margin-bottom: var(--space-4);
}

.submit-button {
  width: 100%;
  height: 48px;
  border-radius: var(--rounded-lg);
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
  background: linear-gradient(135deg, var(--primary-500), var(--primary-600));
  border: none;
  box-shadow: var(--shadow-md);
  transition: var(--transition-all);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
}

.submit-button:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
  background: linear-gradient(135deg, var(--primary-600), var(--primary-700));
}

.submit-button:active {
  transform: translateY(0);
}

/* Register Link */
.register-link {
  text-align: center;
  font-size: var(--text-sm);
  color: var(--gray-600);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
}

/* Responsive Design */
@media (max-width: 1024px) {
  .login-container {
    grid-template-columns: 1fr;
    grid-template-rows: auto 1fr;
  }
  
  .branding-section {
    padding: var(--space-8) var(--space-6);
    text-align: center;
  }
  
  .brand-title {
    font-size: var(--text-3xl);
  }
  
  .feature-list {
    flex-direction: row;
    justify-content: center;
    flex-wrap: wrap;
  }
  
  .feature-item {
    font-size: var(--text-base);
  }
}

@media (max-width: 768px) {
  .login-container {
    padding: var(--space-4);
  }
  
  .branding-section {
    padding: var(--space-6) var(--space-4);
  }
  
  .form-section {
    padding: var(--space-6) var(--space-4);
  }
  
  .brand-title {
    font-size: var(--text-2xl);
  }
  
  .form-title {
    font-size: var(--text-2xl);
  }
  
  .feature-list {
    display: none;
  }
  
  .logo-icon {
    width: 60px;
    height: 60px;
  }
}

@media (max-width: 480px) {
  .captcha-row {
    flex-direction: column;
    gap: var(--space-2);
  }
  
  .captcha-image-container {
    width: 100%;
    max-width: 200px;
    margin: 0 auto;
  }
  
  .form-options {
    flex-direction: column;
    gap: var(--space-3);
    align-items: flex-start;
  }
}
</style> 