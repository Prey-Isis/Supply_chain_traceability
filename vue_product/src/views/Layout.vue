<!-- views/Layout.vue - 主布局（顶部导航 + 侧边菜单 + 内容区） -->
<template>
  <el-container class="main-container">
    <!-- 顶部导航栏 -->
    <el-header class="app-header">
      <div class="header-left">
        <h3>🚚 供应链管理系统</h3>
      </div>
      <div class="header-right">
        <el-tag type="success" style="margin-right: 12px">
          {{ currentUser.userName }}
        </el-tag>
        <el-tag :type="roleTagType">
          {{ roleText }}
        </el-tag>
        <el-button type="danger" text @click="handleLogout" style="margin-left: 20px">
          退出登录
        </el-button>
      </div>
    </el-header>

    <el-container>
      <!-- 侧边菜单栏 -->
      <el-aside width="200px" class="app-aside">
        <el-menu
          :default-active="currentPage"
          @select="handleMenuSelect"
          background-color="#304156"
          text-color="#bfcbd9"
          active-text-color="#409EFF"
        >
          <!-- 产品管理（所有角色可见） -->
          <el-menu-item index="products">
            <el-icon><Box /></el-icon>
            <span>产品管理</span>
          </el-menu-item>
          <!-- 用户管理（仅管理员可见） -->
          <el-menu-item v-if="currentUser.role === 'admin'" index="users">
            <el-icon><User /></el-icon>
            <span>用户管理</span>
          </el-menu-item>
          <!-- 报表查看（仅经理可见） -->
          <el-menu-item v-if="currentUser.role === 'manager'" index="reports">
            <el-icon><DataAnalysis /></el-icon>
            <span>报表查看</span>
          </el-menu-item>
        </el-menu>
      </el-aside>

      <!-- 右侧内容区域 -->
      <el-main class="app-main">
        <!-- 产品管理页面 -->
        <Products v-if="currentPage === 'products'" />
        <!-- 用户管理页面 -->
        <Users v-if="currentPage === 'users'" />
        <!-- 报表页面 -->
        <div v-if="currentPage === 'reports'" class="placeholder-page">
          <el-empty description="报表功能开发中，敬请期待...">
            <el-button type="primary" @click="currentPage = 'products'">
              返回产品列表
            </el-button>
          </el-empty>
        </div>
      </el-main>
    </el-container>

    <!-- 全局弹窗组件 -->
    <ProductDialog />
    <ProductDetail />
    <UserDialog />
  </el-container>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { logoutAPI } from '../api'
import {
  currentUser, currentPage, isLoggedIn,
  clearLoginState, productList, userList
} from '../store'
import { getProductsAPI, getUsersAPI } from '../api'
import Products from './Products.vue'
import Users from './Users.vue'
import ProductDialog from '../components/ProductDialog.vue'
import ProductDetail from '../components/ProductDetail.vue'
import UserDialog from '../components/UserDialog.vue'

// 角色标签类型
const roleTagType = computed(() => {
  switch (currentUser.role) {
    case 'admin': return 'danger'
    case 'manager': return 'warning'
    default: return 'primary'
  }
})

// 角色显示文本
const roleText = computed(() => {
  const roleMap = { admin: '管理员', manager: '经理', user: '普通用户' }
  return roleMap[currentUser.role] || '未知角色'
})

/** 加载产品列表 */
async function loadProducts() {
  try {
    const res = await getProductsAPI()
    productList.value = res.data || []
  } catch (error) {
    productList.value = []
  }
}

/** 加载用户列表 */
async function loadUsers() {
  try {
    const res = await getUsersAPI()
    userList.value = res.data || []
  } catch (error) {
    userList.value = []
  }
}

/** 菜单切换处理 */
function handleMenuSelect(index) {
  currentPage.value = index
  // 切换页面时加载对应数据
  if (index === 'products') {
    loadProducts()
  } else if (index === 'users') {
    loadUsers()
  }
}

/** 退出登录 */
async function handleLogout() {
  try {
    await logoutAPI()
  } catch (error) {
    // 即使接口失败也清除本地状态
  } finally {
    clearLoginState()
    ElMessage.success('已退出登录')
  }
}

// 组件挂载时加载数据
onMounted(() => {
  loadProducts()
  if (currentUser.role === 'admin') {
    loadUsers()
  }
})
</script>

<style scoped>
.main-container {
  height: 100vh;
}
.app-header {
  background: #2b3a4a;
  color: white;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
}
.header-left h3 {
  margin: 0;
  font-size: 18px;
}
.header-right {
  display: flex;
  align-items: center;
}
.app-aside {
  background-color: #304156;
  min-height: calc(100vh - 60px);
}
.app-main {
  background: #f0f2f5;
  padding: 20px;
}
.placeholder-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 400px;
}
</style>