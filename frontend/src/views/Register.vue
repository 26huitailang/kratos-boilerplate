<template>
  <div class="register-page">
    <!-- Background Elements -->
    <div class="background-elements">
      <div class="bg-circle bg-circle-1"></div>
      <div class="bg-circle bg-circle-2"></div>
      <div class="bg-circle bg-circle-3"></div>
    </div>

    <!-- Main Content -->
    <div class="register-container">
      <!-- Branding Section -->
      <div class="branding-section">
        <div class="brand-logo">
          <div class="logo-icon">
            <el-icon size="48"><UserFilled /></el-icon>
          </div>
          <h1 class="brand-title">Join Us Today</h1>
          <p class="brand-subtitle">Create your account and start your journey with our platform</p>
        </div>
        
        <div class="benefits-list">
          <div class="benefit-item">
            <el-icon color="var(--success-500)"><Check /></el-icon>
            <div>
              <h3>Secure & Private</h3>
              <p>Your data is protected with enterprise-grade security</p>
            </div>
          </div>
          <div class="benefit-item">
            <el-icon color="var(--success-500)"><Check /></el-icon>
            <div>
              <h3>Easy Setup</h3>
              <p>Get started in minutes with our simple registration process</p>
            </div>
          </div>
          <div class="benefit-item">
            <el-icon color="var(--success-500)"><Check /></el-icon>
            <div>
              <h3>24/7 Support</h3>
              <p>Our team is here to help you succeed</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Registration Form Section -->
      <div class="form-section">
        <div class="form-container">
          <div class="form-header">
            <h2 class="form-title">Create Account</h2>
            <p class="form-subtitle">Fill in your information to get started</p>
          </div>

          <el-form
            ref="registerFormRef"
            :model="registerForm"
            :rules="registerRules"
            class="register-form"
            @keyup.enter="handleRegister"
          >
            <!-- Username Field -->
            <el-form-item prop="username" class="form-item">
              <el-input
                v-model="registerForm.username"
                placeholder="Choose a username"
                :prefix-icon="User"
                size="large"
                class="form-input"
              />
            </el-form-item>

            <!-- Email Field -->
            <el-form-item prop="email" class="form-item">
              <el-input
                v-model="registerForm.email"
                placeholder="Enter your email address"
                :prefix-icon="Message"
                size="large"
                class="form-input"
              />
            </el-form-item>

            <!-- Phone Field -->
            <el-form-item prop="phone" class="form-item">
              <el-input
                v-model="registerForm.phone"
                placeholder="Enter your phone number"
                :prefix-icon="Phone"
                size="large"
                class="form-input"
              />
            </el-form-item>

            <!-- Password Field -->
            <el-form-item prop="password" class="form-item">
              <el-input
                v-model="registerForm.password"
                type="password"
                placeholder="Create a strong password"
                :prefix-icon="Lock"
                size="large"
                show-password
                class="form-input"
              />
            </el-form-item>

            <!-- Confirm Password Field -->
            <el-form-item prop="confirmPassword" class="form-item">
              <el-input
                v-model="registerForm.confirmPassword"
                type="password"
                placeholder="Confirm your password"
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
                  v-model="registerForm.captchaCode"
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

            <!-- Terms Agreement -->
            <div class="terms-agreement">
              <el-checkbox v-model="agreeToTerms" class="terms-checkbox">
                I agree to the 
                <el-link type="primary" class="terms-link">Terms of Service</el-link>
                and 
                <el-link type="primary" class="terms-link">Privacy Policy</el-link>
              </el-checkbox>
            </div>

            <!-- Submit Button -->
            <el-form-item class="submit-item">
              <el-button
                type="primary"
                size="large"
                class="submit-button"
                :loading="loading"
                :disabled="!agreeToTerms"
                @click="handleRegister"
              >
                <template v-if="!loading">
                  <el-icon><Right /></el-icon>
                </template>
                {{ loading ? 'Creating Account...' : 'Create Account' }}
              </el-button>
            </el-form-item>

            <!-- Login Link -->
            <div class="login-link">
              <span>Already have an account?</span>
              <el-link type="primary" @click="handleLogin">
                Sign in now
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
  Message, 
  Phone, 
  Key, 
  UserFilled, 
  Check, 
  Right, 
  Refresh 
} from '@element-plus/icons-vue';
import { register, getCaptcha } from '@/api/auth';

const router = useRouter();
const registerFormRef = ref();
const loading = ref(false);
const captchaImage = ref('');
const captchaId = ref('');
const agreeToTerms = ref(false);

const registerForm = reactive({
  username: '',
  password: '',
  confirmPassword: '',
  email: '',
  phone: '',
  captchaCode: '',
});

const validatePass = (rule: any, value: string, callback: any) => {
  if (value === '') {
    callback(new Error('请输入密码'));
  } else if (value.length < 8) {
    callback(new Error('密码长度不能小于8位'));
  } else if (!/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]{8,}$/.test(value)) {
    callback(new Error('密码必须包含大小写字母、数字和特殊字符'));
  } else {
    if (registerForm.confirmPassword !== '') {
      registerFormRef.value?.validateField('confirmPassword');
    }
    callback();
  }
};

const validatePass2 = (rule: any, value: string, callback: any) => {
  if (value === '') {
    callback(new Error('请再次输入密码'));
  } else if (value !== registerForm.password) {
    callback(new Error('两次输入密码不一致!'));
  } else {
    callback();
  }
};

const registerRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '长度在 3 到 20 个字符', trigger: 'blur' },
  ],
  password: [
    { required: true, validator: validatePass, trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, validator: validatePass2, trigger: 'blur' },
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱地址', trigger: 'blur' },
  ],
  phone: [
    { required: true, message: '请输入手机号', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号', trigger: 'blur' },
  ],
  captchaCode: [
    { required: true, message: '请输入验证码', trigger: 'blur' },
  ],
};

// 获取验证码
const refreshCaptcha = async () => {
  try {
    const response = await getCaptcha('image');
    captchaId.value = response.data.captcha_id;
    captchaImage.value = `data:image/png;base64,${response.data.image_data}`;
  } catch (error) {
    ElMessage.error('获取验证码失败');
  }
};

// 处理注册
const handleRegister = async () => {
  if (!registerFormRef.value) return;
  
  await registerFormRef.value.validate(async (valid: boolean) => {
    if (valid) {
      loading.value = true;
      try {
        await register({
          username: registerForm.username,
          password: registerForm.password,
          email: registerForm.email,
          phone: registerForm.phone,
          captchaId: captchaId.value,
          captchaCode: registerForm.captchaCode,
        });
        ElMessage.success('注册成功');
        router.push('/login');
      } catch (error: any) {
        ElMessage.error(error.response?.data?.message || '注册失败');
        refreshCaptcha();
      } finally {
        loading.value = false;
      }
    }
  });
};

// 处理登录
const handleLogin = () => {
  router.push('/login');
};

onMounted(() => {
  refreshCaptcha();
});
</script>

<style scoped>
.register-page {
  min-height: 100vh;
  position: relative;
  background: linear-gradient(135deg, var(--success-50) 0%, var(--success-100) 50%, var(--primary-100) 100%);
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
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.1), rgba(52, 211, 153, 0.1));
  animation: float 8s ease-in-out infinite;
}

.bg-circle-1 {
  width: 180px;
  height: 180px;
  top: 15%;
  right: 15%;
  animation-delay: 0s;
}

.bg-circle-2 {
  width: 220px;
  height: 220px;
  bottom: 20%;
  left: 5%;
  animation-delay: 3s;
}

.bg-circle-3 {
  width: 120px;
  height: 120px;
  top: 40%;
  left: 80%;
  animation-delay: 6s;
}

@keyframes float {
  0%, 100% {
    transform: translateY(0) rotate(0deg);
  }
  50% {
    transform: translateY(-30px) rotate(180deg);
  }
}

/* Main Container */
.register-container {
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
  margin-bottom: var(--space-10);
}

.logo-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  background: linear-gradient(135deg, var(--success-500), var(--success-600));
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

.benefits-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.benefit-item {
  display: flex;
  align-items: flex-start;
  gap: var(--space-4);
}

.benefit-item h3 {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--gray-800);
  margin: 0 0 var(--space-1);
}

.benefit-item p {
  font-size: var(--text-base);
  color: var(--gray-600);
  margin: 0;
  line-height: var(--leading-relaxed);
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
  max-width: 450px;
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
.register-form {
  width: 100%;
}

.form-item {
  margin-bottom: var(--space-4);
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
  box-shadow: 0 0 0 2px rgba(16, 185, 129, 0.2);
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

/* Terms Agreement */
.terms-agreement {
  margin-bottom: var(--space-6);
}

.terms-checkbox :deep(.el-checkbox__label) {
  color: var(--gray-600);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
}

.terms-link {
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
  background: linear-gradient(135deg, var(--success-500), var(--success-600));
  border: none;
  box-shadow: var(--shadow-md);
  transition: var(--transition-all);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
}

.submit-button:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
  background: linear-gradient(135deg, var(--success-600), var(--success-700));
}

.submit-button:active {
  transform: translateY(0);
}

.submit-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Login Link */
.login-link {
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
  .register-container {
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
  
  .benefits-list {
    display: none;
  }
}

@media (max-width: 768px) {
  .register-container {
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
  
  .form-item {
    margin-bottom: var(--space-3);
  }
}
</style> 