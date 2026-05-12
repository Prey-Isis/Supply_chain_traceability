<!-- components/ProductDialog.vue - 产品新增/编辑弹窗 -->
<template>
  <el-dialog
    v-model="showProductDialog"
    :title="editingProduct ? '编辑产品' : '新增产品'"
    width="500px"
    @close="resetProductForm"
    :close-on-click-modal="false"
  >
    <el-form :model="productForm" label-width="100px">
      <!-- 产品ID（编辑时显示但不可修改） -->
      <el-form-item v-if="editingProduct" label="产品ID">
        <el-input :model-value="productForm.Product_Id" disabled />
      </el-form-item>
      <el-form-item label="产品名称" required>
        <el-input v-model="productForm.Name" placeholder="请输入产品名称" maxlength="50" />
      </el-form-item>
      <el-form-item label="当前持有者">
        <el-input v-model="productForm.Current_Holder" placeholder="请输入持有者" maxlength="50" />
      </el-form-item>
      <el-form-item label="状态">
        <el-select v-model="productForm.Status" placeholder="请选择状态" style="width: 100%">
          <el-option label="在售" value="1" />
          <el-option label="下架" value="0" />
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="showProductDialog = false">取消</el-button>
      <el-button type="primary" @click="handleSubmit" :loading="formLoading">
        {{ editingProduct ? '保存修改' : '立即创建' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ElMessage } from 'element-plus'
import {
  showProductDialog, editingProduct, productForm, formLoading,
  resetProductForm, productList
} from '../store'
import { createProductAPI, updateProductAPI, getProductsAPI } from '../api'

/** 提交产品表单 */
async function handleSubmit() {
  // 表单验证
  if (!productForm.Name.trim()) {
    ElMessage.warning('请输入产品名称')
    return
  }

  formLoading.value = true
  try {
    if (editingProduct.value) {
      // 编辑模式：更新产品
      await updateProductAPI(editingProduct.value.Product_Id, {
        Name: productForm.Name,
        Current_Holder: productForm.Current_Holder,
        Status: productForm.Status
      })
      ElMessage.success('产品更新成功')
    } else {
      // 新增模式：创建产品
      const newId = String(Date.now()).slice(-6)
      await createProductAPI({
        Product_Id: newId,
        Name: productForm.Name,
        Current_Holder: productForm.Current_Holder,
        Status: productForm.Status
      })
      ElMessage.success('产品创建成功')
    }

    // 关闭弹窗并刷新列表
    showProductDialog.value = false
    const res = await getProductsAPI()
    productList.value = res.data || []
  } catch (error) {
    console.error('提交产品失败:', error)
  } finally {
    formLoading.value = false
  }
}
</script>