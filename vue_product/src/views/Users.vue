<!-- views/Users.vue - 用户管理页面（仅管理员可见） -->
<template>
  <div>
    <!-- 页面头部 -->
    <div class="page-header">
      <h2>用户管理</h2>
      <el-button type="primary" @click="handleAddUser">
        <el-icon><Plus /></el-icon> 新增用户
      </el-button>
    </div>

    <!-- 用户表格 -->
    <el-table :data="userList" border stripe v-loading="loading">
      <el-table-column prop="Account" label="账号" width="150" align="center" />
      <el-table-column prop="UserName" label="用户名" min-width="150" />
      <el-table-column prop="Role" label="角色" width="120" align="center">
        <template #default="scope">
          <el-tag :type="scope.row.Role === 'admin' ? 'danger' : 'primary'">
            {{ scope.row.Role === 'admin' ? '管理员' : '普通用户' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="Create_Time" label="创建时间" min-width="180" />
      <el-table-column prop="Update_Time" label="更新时间" min-width="180" />
      <el-table-column label="操作" width="180" align="center" fixed="right">
        <template #default="scope">
          <el-button size="small" type="warning" @click="handleEditUser(scope.row)">
            编辑
          </el-button>
          <el-button
            size="small"
            type="danger"
            @click="handleDeleteUser(scope.row)"
            :disabled="scope.row.Account === currentUser.account"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  userList, currentUser,
  showUserDialog, editingUser, userForm,
  resetUserForm
} from '../store'
import { getUsersAPI, deleteUserAPI } from '../api'

const loading = ref(false)

/** 加载用户列表 */
async function fetchUsers() {
  loading.value = true
  try {
    const res = await getUsersAPI()
    userList.value = res.data || []
  } catch (error) {
    userList.value = []
  } finally {
    loading.value = false
  }
}

/** 新增用户 */
function handleAddUser() {
  resetUserForm()
  showUserDialog.value = true
}

/** 编辑用户 */
function handleEditUser(row) {
  editingUser.value = row
  Object.assign(userForm, {
    Account: row.Account,
    UserName: row.UserName,
    Role: row.Role,
    PassWord: ''
  })
  showUserDialog.value = true
}

/** 删除用户 */
async function handleDeleteUser(row) {
  // 不允许删除自己
  if (row.Account === currentUser.account) {
    ElMessage.warning('不能删除当前登录账号')
    return
  }
  try {
    await ElMessageBox.confirm(
      `确定要删除用户「${row.UserName}」(${row.Account}) 吗？`,
      '删除确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await deleteUserAPI(row.Account)
    ElMessage.success('用户已成功删除')
    await fetchUsers()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除用户失败:', error)
    }
  }
}
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.page-header h2 {
  margin: 0;
  font-size: 20px;
  color: #303133;
}
</style>