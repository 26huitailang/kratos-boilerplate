<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'
import { 
  Menu, 
  User, 
  Setting, 
  SwitchButton, 
  House, 
  ArrowDown, 
  Close, 
  Sunny, 
  Moon 
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const isMobileMenuOpen = ref(false)
const isDarkMode = ref(false)

// Check if current route requires auth layout
const showMainLayout = computed(() => {
  return authStore.isAuthenticated && route.name !== 'Login' && route.name !== 'Register'
})

const toggleMobileMenu = () => {
  isMobileMenuOpen.value = !isMobileMenuOpen.value
}

const toggleTheme = () => {
  isDarkMode.value = !isDarkMode.value
  document.documentElement.setAttribute('data-theme', isDarkMode.value ? 'dark' : 'light')
}

const handleLogout = async () => {
  try {
    await authStore.logoutAction()
    ElMessage.success('Successfully logged out')
    router.push('/login')
  } catch (error) {
    ElMessage.error('Logout failed')
  }
}
</script>

<template>
  <!-- Full-screen auth layout for login/register -->
  <div v-if="!showMainLayout" class="auth-layout">
    <router-view />
  </div>

  <!-- Main application layout -->
  <div v-else class="app-layout">
    <!-- Header Navigation -->
    <header class="app-header">
      <div class="header-container">
        <!-- Logo and Brand -->
        <div class="brand-section">
          <div class="logo">
            <h1 class="brand-title">Kratos Boilerplate</h1>
          </div>
        </div>

        <!-- Desktop Navigation -->
        <nav class="desktop-nav">
          <router-link to="/" class="nav-link" :class="{ active: route.name === 'Home' }">
            <el-icon><House /></el-icon>
            <span>Dashboard</span>
          </router-link>
        </nav>

        <!-- User Actions -->
        <div class="user-section">
          <!-- Theme Toggle -->
          <el-button 
            :icon="isDarkMode ? Sunny : Moon" 
            circle 
            size="small" 
            @click="toggleTheme"
            class="theme-toggle"
          />
          
          <!-- User Menu -->
          <el-dropdown trigger="click" placement="bottom-end">
            <div class="user-avatar">
              <el-avatar :icon="User" />
              <span class="username">{{ authStore.userInfo?.username || 'User' }}</span>
              <el-icon><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item :icon="User">
                  Profile
                </el-dropdown-item>
                <el-dropdown-item :icon="Setting">
                  Settings
                </el-dropdown-item>
                <el-dropdown-item divided :icon="SwitchButton" @click="handleLogout">
                  Logout
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>

          <!-- Mobile Menu Toggle -->
          <el-button 
            :icon="Menu" 
            circle 
            size="small" 
            @click="toggleMobileMenu"
            class="mobile-menu-toggle"
          />
        </div>
      </div>
    </header>

    <!-- Mobile Menu Overlay -->
    <div v-if="isMobileMenuOpen" class="mobile-menu-overlay" @click="toggleMobileMenu">
      <div class="mobile-menu" @click.stop>
        <div class="mobile-menu-header">
          <h2>Menu</h2>
          <el-button :icon="Close" circle size="small" @click="toggleMobileMenu" />
        </div>
        <nav class="mobile-nav">
          <router-link to="/" class="mobile-nav-link" @click="toggleMobileMenu">
            <el-icon><House /></el-icon>
            Dashboard
          </router-link>
        </nav>
      </div>
    </div>

    <!-- Main Content Area -->
    <main class="app-main">
      <div class="main-container">
        <router-view v-slot="{ Component }">
          <transition name="page" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
    </main>
  </div>
</template>

<style scoped>
/* Auth Layout */
.auth-layout {
  min-height: 100vh;
  background: linear-gradient(135deg, var(--primary-50) 0%, var(--primary-100) 100%);
}

/* Main App Layout */
.app-layout {
  min-height: 100vh;
  background-color: var(--gray-50);
  display: flex;
  flex-direction: column;
}

/* Header Styles */
.app-header {
  background: linear-gradient(135deg, var(--primary-600) 0%, var(--primary-700) 100%);
  box-shadow: var(--shadow-md);
  position: sticky;
  top: 0;
  z-index: var(--z-fixed);
  backdrop-filter: blur(10px);
}

.header-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 var(--space-4);
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

/* Brand Section */
.brand-section {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.logo {
  display: flex;
  align-items: center;
}

.brand-title {
  color: white;
  font-size: var(--text-xl);
  font-weight: var(--font-bold);
  margin: 0;
  letter-spacing: -0.025em;
}

/* Desktop Navigation */
.desktop-nav {
  display: flex;
  gap: var(--space-2);
}

.nav-link {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-4);
  color: rgba(255, 255, 255, 0.9);
  text-decoration: none;
  border-radius: var(--rounded-lg);
  transition: var(--transition-all);
  font-weight: var(--font-medium);
}

.nav-link:hover {
  background-color: rgba(255, 255, 255, 0.1);
  color: white;
  transform: translateY(-1px);
}

.nav-link.active {
  background-color: rgba(255, 255, 255, 0.15);
  color: white;
}

/* User Section */
.user-section {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.theme-toggle {
  background-color: rgba(255, 255, 255, 0.1) !important;
  border-color: rgba(255, 255, 255, 0.2) !important;
  color: white !important;
}

.theme-toggle:hover {
  background-color: rgba(255, 255, 255, 0.2) !important;
}

.user-avatar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2);
  border-radius: var(--rounded-lg);
  cursor: pointer;
  transition: var(--transition-all);
  color: white;
}

.user-avatar:hover {
  background-color: rgba(255, 255, 255, 0.1);
}

.username {
  font-weight: var(--font-medium);
  font-size: var(--text-sm);
}

.mobile-menu-toggle {
  display: none;
  background-color: rgba(255, 255, 255, 0.1) !important;
  border-color: rgba(255, 255, 255, 0.2) !important;
  color: white !important;
}

/* Mobile Menu */
.mobile-menu-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  z-index: var(--z-modal);
  display: none;
}

.mobile-menu {
  position: absolute;
  top: 0;
  right: 0;
  width: 280px;
  height: 100vh;
  background-color: white;
  box-shadow: var(--shadow-xl);
  padding: var(--space-6);
  overflow-y: auto;
}

.mobile-menu-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-6);
  padding-bottom: var(--space-4);
  border-bottom: 1px solid var(--gray-200);
}

.mobile-nav {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.mobile-nav-link {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  color: var(--gray-700);
  text-decoration: none;
  border-radius: var(--rounded-lg);
  transition: var(--transition-all);
  font-weight: var(--font-medium);
}

.mobile-nav-link:hover {
  background-color: var(--gray-100);
  color: var(--primary-600);
}

/* Main Content */
.app-main {
  flex: 1;
  padding: var(--space-6) 0;
}

.main-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 var(--space-4);
}

/* Page Transitions */
.page-enter-active,
.page-leave-active {
  transition: all 0.3s ease;
}

.page-enter-from {
  opacity: 0;
  transform: translateY(20px);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-20px);
}

/* Responsive Design */
@media (max-width: 768px) {
  .desktop-nav {
    display: none;
  }
  
  .mobile-menu-toggle {
    display: flex !important;
  }
  
  .mobile-menu-overlay {
    display: block;
  }
  
  .user-avatar .username {
    display: none;
  }
  
  .brand-title {
    font-size: var(--text-lg);
  }
  
  .header-container {
    padding: 0 var(--space-3);
  }
  
  .main-container {
    padding: 0 var(--space-3);
  }
  
  .app-main {
    padding: var(--space-4) 0;
  }
}

@media (max-width: 480px) {
  .brand-title {
    font-size: var(--text-base);
  }
  
  .header-container {
    height: 56px;
  }
}
</style>
