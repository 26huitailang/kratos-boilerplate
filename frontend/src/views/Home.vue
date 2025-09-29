<template>
  <div class="home-page">
    <!-- Welcome Hero Section -->
    <section class="hero-section">
      <div class="hero-content">
        <div class="welcome-text">
          <h1 class="hero-title">
            Welcome back, {{ username || 'User' }}!
          </h1>
          <p class="hero-subtitle">
            Here's your dashboard overview for today
          </p>
        </div>
        <div class="quick-actions">
          <el-button type="primary" :icon="Plus" class="action-btn">
            Create New
          </el-button>
          <el-button :icon="Setting" class="action-btn">
            Settings
          </el-button>
        </div>
      </div>
    </section>

    <!-- Stats Cards -->
    <section class="stats-section">
      <div class="stats-grid">
        <div class="stat-card" v-for="stat in stats" :key="stat.id">
          <div class="stat-header">
            <div class="stat-icon" :style="{ backgroundColor: stat.color }">
              <el-icon><component :is="stat.icon" /></el-icon>
            </div>
            <div class="stat-trend" :class="stat.trend">
              <el-icon><component :is="stat.trendIcon" /></el-icon>
              <span>{{ stat.change }}%</span>
            </div>
          </div>
          <div class="stat-content">
            <h3 class="stat-value">{{ stat.value }}</h3>
            <p class="stat-label">{{ stat.label }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- Main Content Grid -->
    <section class="main-content">
      <div class="content-grid">
        <!-- Recent Activity -->
        <div class="content-card activity-card">
          <div class="card-header">
            <h2 class="card-title">
              <el-icon><Clock /></el-icon>
              Recent Activity
            </h2>
            <el-link type="primary" class="view-all-link">
              View All
            </el-link>
          </div>
          <div class="activity-list">
            <div class="activity-item" v-for="activity in recentActivities" :key="activity.id">
              <div class="activity-avatar">
                <el-avatar :src="activity.avatar" :icon="User" />
              </div>
              <div class="activity-content">
                <p class="activity-text">{{ activity.text }}</p>
                <span class="activity-time">{{ activity.time }}</span>
              </div>
              <div class="activity-status" :class="activity.status">
                <el-icon><Check /></el-icon>
              </div>
            </div>
          </div>
        </div>

        <!-- Quick Stats Chart -->
        <div class="content-card chart-card">
          <div class="card-header">
            <h2 class="card-title">
              <el-icon><TrendCharts /></el-icon>
              Performance Overview
            </h2>
            <el-dropdown>
              <el-button size="small" :icon="More" circle />
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item>Export Data</el-dropdown-item>
                  <el-dropdown-item>View Details</el-dropdown-item>
                  <el-dropdown-item>Configure</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
          <div class="chart-placeholder">
            <div class="chart-content">
              <div class="chart-bars">
                <div class="chart-bar" v-for="bar in chartData" :key="bar.id" 
                     :style="{ height: bar.height + '%', backgroundColor: bar.color }">
                </div>
              </div>
              <div class="chart-labels">
                <span v-for="label in chartLabels" :key="label">{{ label }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- System Status -->
        <div class="content-card status-card">
          <div class="card-header">
            <h2 class="card-title">
              <el-icon><Monitor /></el-icon>
              System Status
            </h2>
            <el-tag :type="systemStatus.type" size="small">
              {{ systemStatus.text }}
            </el-tag>
          </div>
          <div class="status-list">
            <div class="status-item" v-for="item in systemItems" :key="item.id">
              <div class="status-info">
                <span class="status-name">{{ item.name }}</span>
                <span class="status-detail">{{ item.detail }}</span>
              </div>
              <div class="status-indicator" :class="item.status">
                <el-icon><component :is="item.icon" /></el-icon>
              </div>
            </div>
          </div>
        </div>

        <!-- Notifications -->
        <div class="content-card notifications-card">
          <div class="card-header">
            <h2 class="card-title">
              <el-icon><Bell /></el-icon>
              Notifications
            </h2>
            <el-badge :value="notifications.length" class="notification-badge" />
          </div>
          <div class="notification-list">
            <div class="notification-item" v-for="notification in notifications" :key="notification.id">
              <div class="notification-icon" :class="notification.type">
                <el-icon><component :is="notification.icon" /></el-icon>
              </div>
              <div class="notification-content">
                <p class="notification-text">{{ notification.text }}</p>
                <span class="notification-time">{{ notification.time }}</span>
              </div>
              <el-button size="small" text @click="dismissNotification(notification.id)">
                <el-icon><Close /></el-icon>
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import { 
  Plus, 
  Setting, 
  User, 
  Clock, 
  Check, 
  TrendCharts, 
  More, 
  Monitor, 
  Bell, 
  Close,
  ArrowUp,
  ArrowDown,
  CircleCheck,
  Warning,
  InfoFilled
} from '@element-plus/icons-vue';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const authStore = useAuthStore();
const username = ref('');

// Mock data for demonstration
const stats = reactive([
  {
    id: 1,
    label: 'Total Users',
    value: '2,547',
    change: '+12.5',
    trend: 'positive',
    color: 'var(--primary-500)',
    icon: 'User',
    trendIcon: 'ArrowUp'
  },
  {
    id: 2,
    label: 'Active Sessions',
    value: '1,423',
    change: '+8.2',
    trend: 'positive',
    color: 'var(--success-500)',
    icon: 'Monitor',
    trendIcon: 'ArrowUp'
  },
  {
    id: 3,
    label: 'API Calls',
    value: '45.2K',
    change: '-2.1',
    trend: 'negative',
    color: 'var(--warning-500)',
    icon: 'TrendCharts',
    trendIcon: 'ArrowDown'
  },
  {
    id: 4,
    label: 'System Health',
    value: '98.5%',
    change: '+0.3',
    trend: 'positive',
    color: 'var(--success-500)',
    icon: 'CircleCheck',
    trendIcon: 'ArrowUp'
  }
]);

const recentActivities = reactive([
  {
    id: 1,
    text: 'New user registration completed',
    time: '2 minutes ago',
    avatar: '',
    status: 'success'
  },
  {
    id: 2,
    text: 'System backup completed successfully',
    time: '15 minutes ago',
    avatar: '',
    status: 'success'
  },
  {
    id: 3,
    text: 'API rate limit threshold reached',
    time: '1 hour ago',
    avatar: '',
    status: 'warning'
  }
]);

const chartData = reactive([
  { id: 1, height: 75, color: 'var(--primary-500)' },
  { id: 2, height: 45, color: 'var(--primary-400)' },
  { id: 3, height: 85, color: 'var(--primary-600)' },
  { id: 4, height: 35, color: 'var(--primary-300)' },
  { id: 5, height: 65, color: 'var(--primary-500)' },
  { id: 6, height: 90, color: 'var(--primary-700)' },
  { id: 7, height: 55, color: 'var(--primary-400)' }
]);

const chartLabels = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

const systemStatus = reactive({
  type: 'success',
  text: 'All Systems Operational'
});

const systemItems = reactive([
  {
    id: 1,
    name: 'Database',
    detail: 'Latency: 12ms',
    status: 'healthy',
    icon: 'CircleCheck'
  },
  {
    id: 2,
    name: 'API Gateway',
    detail: 'Response time: 45ms',
    status: 'healthy',
    icon: 'CircleCheck'
  },
  {
    id: 3,
    name: 'Cache Layer',
    detail: 'Hit rate: 89%',
    status: 'warning',
    icon: 'Warning'
  }
]);

const notifications = reactive([
  {
    id: 1,
    text: 'Security update available for your system',
    time: '1 hour ago',
    type: 'info',
    icon: 'InfoFilled'
  },
  {
    id: 2,
    text: 'Weekly system maintenance scheduled',
    time: '3 hours ago',
    type: 'warning',
    icon: 'Warning'
  }
]);

const dismissNotification = (id: number) => {
  const index = notifications.findIndex(n => n.id === id);
  if (index > -1) {
    notifications.splice(index, 1);
  }
};

const handleLogout = async () => {
  try {
    await authStore.logoutAction();
    ElMessage.success('Successfully logged out');
    router.push('/login');
  } catch (error) {
    ElMessage.error('Logout failed');
  }
};

onMounted(() => {
  // TODO: Fetch user info from backend
  username.value = authStore.userInfo?.username || 'Test User';
});
</script>

<style scoped>
.home-page {
  min-height: 100vh;
  background-color: var(--gray-50);
  padding: var(--space-6);
}

/* Hero Section */
.hero-section {
  margin-bottom: var(--space-8);
}

.hero-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: linear-gradient(135deg, var(--primary-500), var(--primary-600));
  border-radius: var(--rounded-2xl);
  padding: var(--space-8);
  color: white;
  box-shadow: var(--shadow-lg);
}

.welcome-text h1 {
  font-size: var(--text-3xl);
  font-weight: var(--font-bold);
  margin: 0 0 var(--space-2);
}

.hero-subtitle {
  font-size: var(--text-lg);
  opacity: 0.9;
  margin: 0;
}

.quick-actions {
  display: flex;
  gap: var(--space-3);
}

.action-btn {
  background-color: rgba(255, 255, 255, 0.2) !important;
  border-color: rgba(255, 255, 255, 0.3) !important;
  color: white !important;
}

.action-btn:hover {
  background-color: rgba(255, 255, 255, 0.3) !important;
}

/* Stats Section */
.stats-section {
  margin-bottom: var(--space-8);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: var(--space-6);
}

.stat-card {
  background: white;
  border-radius: var(--rounded-xl);
  padding: var(--space-6);
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--gray-200);
  transition: var(--transition-all);
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.stat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-4);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--rounded-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: var(--text-xl);
}

.stat-trend {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
}

.stat-trend.positive {
  color: var(--success-600);
}

.stat-trend.negative {
  color: var(--error-600);
}

.stat-value {
  font-size: var(--text-3xl);
  font-weight: var(--font-bold);
  color: var(--gray-900);
  margin: 0 0 var(--space-1);
}

.stat-label {
  font-size: var(--text-base);
  color: var(--gray-600);
  margin: 0;
}

/* Main Content */
.main-content {
  margin-bottom: var(--space-8);
}

.content-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: var(--space-6);
}

.content-card {
  background: white;
  border-radius: var(--rounded-xl);
  border: 1px solid var(--gray-200);
  box-shadow: var(--shadow-sm);
  overflow: hidden;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-6) var(--space-6) var(--space-4);
  border-bottom: 1px solid var(--gray-100);
}

.card-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--gray-900);
  margin: 0;
}

.view-all-link {
  font-size: var(--text-sm);
}

/* Activity Card */
.activity-list {
  padding: var(--space-4) var(--space-6) var(--space-6);
}

.activity-item {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-3) 0;
  border-bottom: 1px solid var(--gray-100);
}

.activity-item:last-child {
  border-bottom: none;
}

.activity-content {
  flex: 1;
}

.activity-text {
  font-size: var(--text-sm);
  color: var(--gray-800);
  margin: 0 0 var(--space-1);
}

.activity-time {
  font-size: var(--text-xs);
  color: var(--gray-500);
}

.activity-status {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.activity-status.success {
  background-color: var(--success-500);
}

.activity-status.warning {
  background-color: var(--warning-500);
}

/* Chart Card */
.chart-placeholder {
  padding: var(--space-4) var(--space-6) var(--space-6);
  height: 200px;
}

.chart-content {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.chart-bars {
  flex: 1;
  display: flex;
  align-items: end;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

.chart-bar {
  flex: 1;
  min-height: 20px;
  border-radius: var(--rounded) var(--rounded) 0 0;
  transition: var(--transition-all);
}

.chart-bar:hover {
  opacity: 0.8;
}

.chart-labels {
  display: flex;
  justify-content: space-between;
  font-size: var(--text-xs);
  color: var(--gray-500);
}

/* Status Card */
.status-list {
  padding: var(--space-4) var(--space-6) var(--space-6);
}

.status-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-3) 0;
  border-bottom: 1px solid var(--gray-100);
}

.status-item:last-child {
  border-bottom: none;
}

.status-info {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.status-name {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--gray-800);
}

.status-detail {
  font-size: var(--text-xs);
  color: var(--gray-500);
}

.status-indicator {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: var(--text-sm);
}

.status-indicator.healthy {
  background-color: var(--success-500);
}

.status-indicator.warning {
  background-color: var(--warning-500);
}

.status-indicator.error {
  background-color: var(--error-500);
}

/* Notifications Card */
.notification-list {
  padding: var(--space-4) var(--space-6) var(--space-6);
}

.notification-item {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3) 0;
  border-bottom: 1px solid var(--gray-100);
}

.notification-item:last-child {
  border-bottom: none;
}

.notification-icon {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.notification-icon.info {
  background-color: var(--primary-500);
}

.notification-icon.warning {
  background-color: var(--warning-500);
}

.notification-icon.error {
  background-color: var(--error-500);
}

.notification-content {
  flex: 1;
}

.notification-text {
  font-size: var(--text-sm);
  color: var(--gray-800);
  margin: 0 0 var(--space-1);
}

.notification-time {
  font-size: var(--text-xs);
  color: var(--gray-500);
}

/* Responsive Design */
@media (max-width: 1024px) {
  .content-grid {
    grid-template-columns: 1fr;
  }
  
  .stats-grid {
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  }
}

@media (max-width: 768px) {
  .home-page {
    padding: var(--space-4);
  }
  
  .hero-content {
    flex-direction: column;
    text-align: center;
    gap: var(--space-4);
    padding: var(--space-6);
  }
  
  .hero-title {
    font-size: var(--text-2xl) !important;
  }
  
  .stats-grid {
    grid-template-columns: 1fr;
    gap: var(--space-4);
  }
  
  .content-grid {
    gap: var(--space-4);
  }
  
  .content-card {
    border-radius: var(--rounded-lg);
  }
}

@media (max-width: 480px) {
  .quick-actions {
    flex-direction: column;
    width: 100%;
    gap: var(--space-2);
  }
  
  .stat-card,
  .content-card {
    margin: 0;
  }
  
  .card-header {
    padding: var(--space-4);
  }
  
  .activity-list,
  .chart-placeholder,
  .status-list,
  .notification-list {
    padding: var(--space-3) var(--space-4) var(--space-4);
  }
}
</style> 