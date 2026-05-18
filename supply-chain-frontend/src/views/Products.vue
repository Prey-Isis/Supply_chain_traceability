<!-- views/Products.vue - 产品管理页面 -->
<template>
  <div>
    <!-- 页面头部 -->
    <div class="page-header">
      <h2>产品列表</h2>
      <div class="header-actions">
        <el-button type="primary" @click="handleAddProduct">
          <el-icon><Plus /></el-icon> 新增产品
        </el-button>
        <el-button @click="refreshProducts" :loading="loading">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>
    </div>

    <!-- 搜索和筛选区域 -->
    <div class="filter-bar">
      <el-input
        v-model="productSearch"
        placeholder="按产品名称搜索..."
        clearable
        style="width: 280px"
        @input="filterProducts"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      
      <el-select
        v-model="statusFilter"
        placeholder="按状态筛选"
        clearable
        style="width: 180px; margin-left: 12px"
        @change="handleStatusFilter"
      >
        <el-option
          v-for="status in statusFilterOptions"
          :key="status.value"
          :label="status.label"
          :value="status.value"
        >
          <span>{{ status.icon }} {{ status.label }}</span>
        </el-option>
      </el-select>
    </div>

    <!-- 产品表格 -->
    <el-table 
      :data="paginatedProducts" 
      border 
      stripe 
      style="width: 100%" 
      v-loading="loading"
      :empty-text="'暂无产品数据'"
    >
      <el-table-column prop="Product_Id" label="产品ID" width="100" align="center" />
      
      <el-table-column prop="Name" label="产品名称" min-width="150">
        <template #default="scope">
          <span class="product-name">{{ scope.row.Name }}</span>
        </template>
      </el-table-column>
      
      <el-table-column prop="Status" label="当前状态" width="130" align="center">
        <template #default="scope">
          <el-tag :type="getStatusTagType(scope.row.Status)" effect="plain">
            {{ getStatusIcon(scope.row.Status) }} {{ getStatusText(scope.row.Status) }}
          </el-tag>
        </template>
      </el-table-column>
      
      <el-table-column prop="Current_Holder" label="当前持有者" min-width="150">
        <template #default="scope">
          <span class="holder-info">
            <el-icon><User /></el-icon>
            {{ scope.row.Current_Holder || '未指定' }}
          </span>
        </template>
      </el-table-column>
      
      <el-table-column prop="Create_Time" label="创建时间" width="170">
        <template #default="scope">
          {{ formatTableTime(scope.row.Create_Time) }}
        </template>
      </el-table-column>
      
      <el-table-column prop="Update_Time" label="更新时间" width="170">
        <template #default="scope">
          {{ formatTableTime(scope.row.Update_Time) }}
        </template>
      </el-table-column>
      
      <el-table-column label="操作" width="260" align="center" fixed="right">
        <template #default="scope">
          <el-button size="small" type="primary" @click="handleViewDetail(scope.row)">
            <el-icon><View /></el-icon> 详情
          </el-button>
          <el-button size="small" type="warning" @click="handleEditProduct(scope.row)">
            <el-icon><Edit /></el-icon> 编辑
          </el-button>
          <el-button
            v-if="currentUser.role === 'admin'"
            size="small"
            type="danger"
            @click="handleDeleteProduct(scope.row)"
          >
            <el-icon><Delete /></el-icon> 删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页组件 -->
    <div class="pagination-container">
      <el-pagination
        v-model:current-page="productPage"
        :page-size="productPageSize"
        :total="displayedProducts.length"
        layout="total, sizes, prev, pager, next, jumper"
        :page-sizes="[10, 20, 50, 100]"
        background
        @size-change="handleSizeChange"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  productList, productSearch, productPage, productPageSize,
  filteredProducts, currentUser,
  showProductDialog, editingProduct, productForm,
  showDetailDialog, selectedProduct, supplyHistory,
  filterProducts, resetProductForm
} from '../store'
import {
  getProductsAPI, getProductHistoryAPI,
  deleteProductAPI
} from '../api'

const loading = ref(false)
const statusFilter = ref('')

// ==================== 状态筛选选项 ====================
const statusFilterOptions = [
  { value: 'produced', label: '已生产', icon: '🏭' },
  { value: 'processing', label: '加工中', icon: '⚙️' },
  { value: 'processed', label: '已加工', icon: '✅' },
  { value: 'distributing', label: '分销中', icon: '🚚' },
  { value: 'retailing', label: '零售中', icon: '🏪' },
  { value: 'sold', label: '已售出', icon: '🎉' },
  { value: 'returned', label: '已退货', icon: '↩️' }
]

// ==================== 状态映射函数 ====================
const statusMap = {
  'produced': { text: '已生产', icon: '🏭', tagType: '' },
  'processing': { text: '加工中', icon: '⚙️', tagType: 'warning' },
  'processed': { text: '已加工', icon: '✅', tagType: 'success' },
  'distributing': { text: '分销中', icon: '🚚', tagType: 'danger' },
  'retailing': { text: '零售中', icon: '🏪', tagType: 'info' },
  'sold': { text: '已售出', icon: '🎉', tagType: 'success' },
  'returned': { text: '已退货', icon: '↩️', tagType: 'danger' },
  '1': { text: '在售', icon: '🏪', tagType: 'success' },
  '0': { text: '下架', icon: '📦', tagType: 'info' }
}

function getStatusText(status) {
  return statusMap[status]?.text || status || '未知状态'
}

function getStatusIcon(status) {
  return statusMap[status]?.icon || '📋'
}

function getStatusTagType(status) {
  return statusMap[status]?.tagType || ''
}

// ==================== 时间格式化函数 ====================
function formatTableTime(timeStr) {
  if (!timeStr) return '-'
  return timeStr.replace('T', ' ').replace('+08:00', '').substring(0, 19)
}

// ==================== 状态筛选后的产品列表 ====================
const displayedProducts = computed(() => {
  let list = filteredProducts.value
  if (statusFilter.value) {
    list = list.filter(p => p.Status === statusFilter.value)
  }
  return list
})

// 分页后的产品列表
const paginatedProducts = computed(() => {
  const start = (productPage.value - 1) * productPageSize.value
  return displayedProducts.value.slice(start, start + productPageSize.value)
})

// 状态筛选处理
function handleStatusFilter() {
  productPage.value = 1 // 重置页码
}

// 每页条数变化处理
function handleSizeChange() {
  productPage.value = 1 // 重置页码
}

// ==================== 数据加载 ====================
/** 加载产品列表 */
async function fetchProducts() {
  loading.value = true
  try {
    const res = await getProductsAPI()
    productList.value = res.data || []
  } catch (error) {
    productList.value = []
  } finally {
    loading.value = false
  }
}

/** 刷新产品列表 */
async function refreshProducts() {
  await fetchProducts()
  ElMessage.success('产品列表已刷新')
}

// ==================== 产品操作 ====================
/** 新增产品 */
function handleAddProduct() {
  resetProductForm()
  showProductDialog.value = true
}

/** 编辑产品 */
function handleEditProduct(row) {
  editingProduct.value = row
  Object.assign(productForm, {
    Product_Id: row.Product_Id,
    Name: row.Name,
    Current_Holder: row.Current_Holder,
    Status: row.Status
  })
  showProductDialog.value = true
}

/** 
 * 查看产品详情和供应链历史
 * 合并了原来的"详情"和"供应链"两个按钮的功能
 */
async function handleViewDetail(row) {
  selectedProduct.value = row
  try {
    const res = await getProductHistoryAPI(row.Product_Id)
    supplyHistory.value = res.data || []
  } catch (error) {
    // 如果API调用失败，使用产品对象中的Supply_History
    supplyHistory.value = row.Supply_History || []
  }
  showDetailDialog.value = true
}

/** 删除产品 */
async function handleDeleteProduct(row) {
  try {
    await ElMessageBox.confirm(
      `确定要删除产品「${row.Name}」(ID: ${row.Product_Id}) 吗？\n此操作不可恢复，相关的供应链记录也将被删除。`,
      '删除确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )
    
    await deleteProductAPI(row.Product_Id)
    ElMessage.success(`产品「${row.Name}」已成功删除`)
    await fetchProducts()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      console.error('删除产品失败:', error)
    }
  }
}

// ==================== 生命周期 ====================
onMounted(() => {
  fetchProducts()
})
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
  font-weight: 600;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.filter-bar {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 8px;
}

.product-name {
  font-weight: 500;
  color: #303133;
}

.holder-info {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #606266;
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
  padding: 10px 0;
}

/* 表格行悬停效果 */
:deep(.el-table__row:hover) {
  background-color: #f5f7fa !important;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .filter-bar {
    flex-direction: column;
  }
  
  .filter-bar .el-input,
  .filter-bar .el-select {
    width: 100% !important;
    margin-left: 0 !important;
  }
}
</style>