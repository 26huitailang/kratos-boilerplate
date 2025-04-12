<template>
  <div class="home-container">
    <el-container>
      <el-header>
        <div class="header-content">
          <h1>Cross Redline</h1>
          <div class="user-info">
            <span>{{ username }}</span>
            <el-button type="danger" @click="handleLogout">退出登录</el-button>
          </div>
        </div>
      </el-header>
      <el-main>
        <el-card class="welcome-card">
          <template #header>
            <h2>欢迎使用 Cross Redline</h2>
          </template>
          <p>这是一个基于 Vue 3 + TypeScript + Element Plus 的前端项目</p>
        </el-card>
      </el-main>
    </el-container>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const authStore = useAuthStore();
const username = ref('');

const handleLogout = async () => {
  try {
    await authStore.logoutAction();
    ElMessage.success('退出登录成功');
    router.push('/login');
  } catch (error) {
    ElMessage.error('退出登录失败');
  }
};

onMounted(() => {
  // TODO: 从后端获取用户信息
  username.value = '测试用户';
});
</script>

<style scoped>
.home-container {
  height: 100vh;
}

.el-header {
  background-color: #409EFF;
  color: white;
  line-height: 60px;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  max-width: 1200px;
  margin: 0 auto;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 20px;
}

.el-main {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

.welcome-card {
  margin-top: 20px;
}
</style> 