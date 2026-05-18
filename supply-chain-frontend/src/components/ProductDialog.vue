<!-- components/ProductDialog.vue - 产品新增/编辑弹窗 -->
<template>
  <el-dialog
    v-model="showProductDialog"
    :title="editingProduct ? '编辑产品' : '新增产品'"
    width="550px"
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
        <el-input v-model="productForm.Current_Holder" placeholder="请输入当前持有者" maxlength="50" />
      </el-form-item>
      
      <!-- 状态改为供应链节点选择 -->
      <el-form-item label="当前状态" required>
        <el-select v-model="productForm.Status" placeholder="请选择产品当前状态" style="width: 100%">
          <el-option
            v-for="status in productStatusOptions"
            :key="status.value"
            :label="status.label"
            :value="status.value"
          >
            <span :style="{ color: status.color }">{{ status.icon }}</span>
            <span>{{ status.label }}</span>
          </el-option>
        </el-select>
        <!-- 显示当前状态说明 -->
        <span class="form-tip">{{ getStatusDescription(productForm.Status) }}</span>
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

// 产品状态选项（对应供应链节点）
const productStatusOptions = [
  { 
    value: 'produced', 
    label: '已生产', 
    icon: '🏭', 
    color: '#409EFF',
    description: '产品已完成生产，等待加工处理'
  },
  { 
    value: 'processing', 
    label: '加工中', 
    icon: '⚙️', 
    color: '#E6A23C',
    description: '产品正在进行加工处理'
  },
  { 
    value: 'processed', 
    label: '已加工', 
    icon: '✅', 
    color: '#67C23A',
    description: '产品加工完成，准备分销'
  },
  { 
    value: 'distributing', 
    label: '分销中', 
    icon: '🚚', 
    color: '#F56C6C',
    description: '产品正在分销运输途中'
  },
  { 
    value: 'retailing', 
    label: '零售中', 
    icon: '🏪', 
    color: '#909399',
    description: '产品已到达零售终端，等待消费者购买'
  },
  { 
    value: 'sold', 
    label: '已售出', 
    icon: '🎉', 
    color: '#67C23A',
    description: '产品已被消费者购买'
  },
  { 
    value: 'returned', 
    label: '已退货', 
    icon: '↩️', 
    color: '#F56C6C',
    description: '产品被退回，需要重新处理'
  }
]

// 获取状态描述
function getStatusDescription(status) {
  const option = productStatusOptions.find(opt => opt.value === status)
  return option ? option.description : ''
}

/** 提交产品表单 */
async function handleSubmit() {
  // 表单验证
  if (!productForm.Name.trim()) {
    ElMessage.warning('请输入产品名称')
    return
  }
  
  if (!productForm.Status) {
    ElMessage.warning('请选择产品状态')
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
      // 新增模式：创建产品（默认为"已生产"状态）
      const newId = String(Date.now()).slice(-6)
      await createProductAPI({
        Product_Id: newId,
        Name: productForm.Name,
        Current_Holder: productForm.Current_Holder,
        Status: productForm.Status || 'produced'
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

<style scoped>
.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  display: inline-block;
  line-height: 1.5;
}
</style>