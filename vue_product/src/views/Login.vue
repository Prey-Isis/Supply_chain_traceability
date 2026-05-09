<!-- views/Login.vue - 登录页面 -->
<template>
  <div class="login-container">
    <el-card class="login-card" shadow="always">
      <h2 class="login-title">🚚 供应链管理系统</h2>
      <el-form :model="loginForm" label-width="80px" @keyup.enter="handleLogin">
        <el-form-item label="账号">
          <el-input v-model="loginForm.account" placeholder="请输入账号" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input
            v-model="loginForm.password"
            type="password"
            placeholder="请输入密码"
            show-password
          />
        </el-form-item>
        <el-form-item>
          <el-button
            type="primary"
            @click="handleLogin"
            :loading="loginLoading"
            style="width: 100%"
          >
            登 录
          </el-button>
        </el-form-item>
      </el-form>
      <div class="login-hint">
        测试账号：11111111 / 123456 (管理员)
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { loginAPI } from '../api'
import {
  token, currentUser, isLoggedIn, currentPage,
  productList, userList, restoreLoginState
} from '../store'

// 登录表单
const loginForm = reactive({
  account: '11111111',
  password: '123456'
})
const loginLoading = ref(false)

/**
 * 处理登录
 * 调用登录API，成功后保存token和用户信息到localStorage
 */
async function handleLogin() {
  // 表单验证
  if (!loginForm.account.trim() || !loginForm.password.trim()) {
    ElMessage.warning('请输入账号和密码')
    return
  }

  loginLoading.value = true
  try {
    const res = await loginAPI(loginForm.account, loginForm.password)

    // 保存认证信息
    token.value = res.data.token
    Object.assign(currentUser, {
      account: res.data.account,
      userName: res.data.userName,
      role: res.data.role
    })

    // 持久化存储
    localStorage.setItem('token', token.value)
    localStorage.setItem('userInfo', JSON.stringify({
      account: currentUser.account,
      userName: currentUser.userName,
      role: currentUser.role
    }))

    isLoggedIn.value = true
    currentPage.value = 'products'
    
    // 清空数据列表，让Layout组件重新加载
    productList.value = []
    userList.value = []
    
    ElMessage.success(`欢迎回来，${currentUser.userName}！`)
  } catch (error) {
    // 错误已在api拦截器中处理
    console.error('登录失败:', error)
  } finally {
    loginLoading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}
.login-card {
  width: 420px;
  padding: 10px;
}
.login-title {
  text-align: center;
  margin-bottom: 30px;
  color: #303133;
  font-size: 22px;
}
.login-hint {
  text-align: center;
  color: #999;
  font-size: 13px;
  margin-top: 10px;
}
</style>