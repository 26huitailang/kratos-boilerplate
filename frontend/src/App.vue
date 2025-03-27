<script setup lang="ts">
// 组件逻辑
</script>

<template>
  <el-container>
    <el-header>
      <h1>Vue 3 + TypeScript 示例</h1>
    </el-header>
    <el-main>
      <router-view></router-view>
    </el-main>
  </el-container>
</template>

<style>
.el-header {
  background-color: #409EFF;
  color: white;
  line-height: 60px;
  text-align: center;
}

.el-main {
  padding: 20px;
}
</style>
