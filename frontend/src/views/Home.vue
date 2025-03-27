<template>
  <div class="home">
    <el-row class="mb-4">
      <el-button type="primary">主要按钮</el-button>
      <el-button type="success">成功按钮</el-button>
      <el-button type="info">信息按钮</el-button>
      <el-button type="warning">警告按钮</el-button>
      <el-button type="danger">危险按钮</el-button>
    </el-row>
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>计数器</span>
          <el-button type="primary" @click="store.increment">增加</el-button>
        </div>
      </template>
      <div class="text item">
        当前计数: {{ store.count }}
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { useMainStore } from '../stores'

const store = useMainStore()
</script>

<style scoped>
.home {
  padding: 20px;
}
.mb-4 {
  margin-bottom: 16px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style> 