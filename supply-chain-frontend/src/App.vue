<!-- App.vue - 根组件，控制页面切换 -->
<template>
  <!-- 未登录显示登录页 -->
  <Login v-if="!isLoggedIn" />
  <!-- 已登录显示主布局 -->
  <Layout v-else />
</template>

<script setup>
import { onMounted } from 'vue'
import Login from './views/Login.vue'
import Layout from './views/Layout.vue'
import { isLoggedIn, restoreLoginState } from './store'

// 应用启动时尝试恢复登录状态
onMounted(() => {
  restoreLoginState()
})
</script>

<style>
/* 全局样式 */
body {
  margin: 0;
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Microsoft YaHei', Arial, sans-serif;
}
#app {
  height: 100vh;
}
</style>