<template>
  <div class="register-container">
    <el-card class="register-card">
      <template #header>
        <h2>用户注册</h2>
      </template>
      
      <el-form
        ref="registerFormRef"
        :model="registerForm"
        :rules="registerRules"
        label-width="100px"
        @keyup.enter="handleRegister"
      >
        <el-form-item label="用户名" prop="username">
          <el-input
            v-model="registerForm.username"
            placeholder="请输入用户名"
            :prefix-icon="User"
          />
        </el-form-item>

        <el-form-item label="密码" prop="password">
          <el-input
            v-model="registerForm.password"
            type="password"
            placeholder="请输入密码"
            :prefix-icon="Lock"
            show-password
          />
        </el-form-item>

        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input
            v-model="registerForm.confirmPassword"
            type="password"
            placeholder="请再次输入密码"
            :prefix-icon="Lock"
            show-password
          />
        </el-form-item>

        <el-form-item label="邮箱" prop="email">
          <el-input
            v-model="registerForm.email"
            placeholder="请输入邮箱"
            :prefix-icon="Message"
          />
        </el-form-item>

        <el-form-item label="手机号" prop="phone">
          <el-input
            v-model="registerForm.phone"
            placeholder="请输入手机号"
            :prefix-icon="Phone"
          />
        </el-form-item>

        <el-form-item label="验证码" prop="captchaCode">
          <div class="captcha-container">
            <el-input
              v-model="registerForm.captchaCode"
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

        <el-form-item>
          <el-button type="primary" @click="handleRegister" :loading="loading">
            注册
          </el-button>
          <el-button @click="handleLogin">返回登录</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import { User, Lock, Message, Phone, Key } from '@element-plus/icons-vue';
import { register, getCaptcha } from '@/api/auth';

const router = useRouter();
const registerFormRef = ref();
const loading = ref(false);
const captchaImage = ref('');
const captchaId = ref('');

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
.register-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background-color: #f5f7fa;
}

.register-card {
  width: 500px;
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