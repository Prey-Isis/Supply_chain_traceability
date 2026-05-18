<!-- components/UserDialog.vue - 用户新增/编辑弹窗 -->
<template>
  <el-dialog
    v-model="showUserDialog"
    :title="editingUser ? '编辑用户' : '新增用户'"
    width="500px"
    @close="resetUserForm"
    :close-on-click-modal="false"
  >
    <el-form :model="userForm" label-width="80px">
      <!-- 账号：新增时可编辑，编辑时显示但不可修改 -->
      <el-form-item label="账号" required>
        <el-input
          v-if="!editingUser"
          v-model="userForm.Account"
          placeholder="请输入账号"
          maxlength="20"
        />
        <el-input v-else :model-value="userForm.Account" disabled />
      </el-form-item>
      
      <el-form-item label="用户名" required>
        <el-input v-model="userForm.UserName" placeholder="请输入用户名" maxlength="30" />
      </el-form-item>
      
      <!-- 密码：新增和编辑都显示 -->
      <el-form-item label="密码">
        <el-input
          v-model="userForm.PassWord"
          type="password"
          :placeholder="editingUser ? '留空则不修改密码' : '请输入密码'"
          show-password
          maxlength="30"
        />
        <!-- 编辑模式下提示用户密码可留空 -->
        <span v-if="editingUser" class="form-tip">留空则不修改密码</span>
      </el-form-item>
      
      <el-form-item label="角色" required>
        <el-select v-model="userForm.Role" placeholder="请选择角色" style="width: 100%">
          <el-option label="普通用户" value="user" />
          <el-option label="管理员" value="admin" />
          <el-option label="经理" value="manager" />
        </el-select>
      </el-form-item>
    </el-form>
    
    <template #footer>
      <el-button @click="showUserDialog = false">取消</el-button>
      <el-button type="primary" @click="handleSubmit" :loading="userFormLoading">
        {{ editingUser ? '保存修改' : '立即创建' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ElMessage } from 'element-plus'
import {
  showUserDialog, editingUser, userForm, userFormLoading,
  resetUserForm, userList
} from '../store'
import { createUserAPI, updateUserAPI, getUsersAPI } from '../api'

/** 提交用户表单 */
async function handleSubmit() {
  // 表单验证
  if (!userForm.UserName.trim()) {
    ElMessage.warning('请输入用户名')
    return
  }
  
  // 新增模式下的必填验证
  if (!editingUser.value) {
    if (!userForm.Account.trim()) {
      ElMessage.warning('请输入账号')
      return
    }
    if (!userForm.PassWord.trim()) {
      ElMessage.warning('请输入密码')
      return
    }
  }

  userFormLoading.value = true
  try {
    if (editingUser.value) {
      // 编辑模式：构建更新数据
      const updateData = {
        UserName: userForm.UserName,
        Role: userForm.Role
      }
      
      // 如果填写了密码，则包含密码字段
      if (userForm.PassWord.trim()) {
        updateData.PassWord = userForm.PassWord.trim()
      }
      
      await updateUserAPI(editingUser.value.Account, updateData)
      ElMessage.success('用户信息更新成功')
    } else {
      // 新增模式：创建用户
      await createUserAPI({
        UserName: userForm.UserName,
        Account: userForm.Account,
        PassWord: userForm.PassWord,
        Role: userForm.Role
      })
      ElMessage.success('用户创建成功')
    }

    // 关闭弹窗并刷新列表
    showUserDialog.value = false
    const res = await getUsersAPI()
    userList.value = res.data || []
  } catch (error) {
    console.error('提交用户失败:', error)
  } finally {
    userFormLoading.value = false
  }
}
</script>

<style scoped>
.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  display: inline-block;
}
</style>