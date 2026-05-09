<!-- views/Products.vue - 产品管理页面 -->
<template>
  <div>
    <!-- 页面头部 -->
    <div class="page-header">
      <h2>产品列表</h2>
      <el-button type="primary" @click="handleAddProduct">
        <el-icon><Plus /></el-icon> 新增产品
      </el-button>
    </div>

    <!-- 搜索框 -->
    <el-input
      v-model="productSearch"
      placeholder="按产品名称搜索..."
      clearable
      style="width: 320px; margin-bottom: 16px"
      @input="filterProducts"
    />

    <!-- 产品表格 -->
    <el-table :data="paginatedProducts" border stripe style="width: 100%" v-loading="loading">
      <el-table-column prop="Product_Id" label="产品ID" width="100" align="center" />
      <el-table-column prop="Name" label="产品名称" min-width="150" />
      <el-table-column prop="Status" label="状态" width="100" align="center">
        <template #default="scope">
          <el-tag :type="scope.row.Status === '1' ? 'success' : 'info'">
            {{ scope.row.Status === '1' ? '在售' : '下架' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="Current_Holder" label="当前持有者" min-width="150" />
      <el-table-column prop="Create_Time" label="创建时间" min-width="170" />
      <el-table-column label="操作" width="340" align="center" fixed="right">
        <template #default="scope">
          <el-button size="small" type="primary" @click="handleViewDetail(scope.row)">
            详情
          </el-button>
          <el-button size="small" type="warning" @click="handleEditProduct(scope.row)">
            编辑
          </el-button>
          <el-button size="small" type="info" @click="handleViewSupply(scope.row)">
            供应链
          </el-button>
          <el-button
            v-if="currentUser.role === 'admin'"
            size="small"
            type="danger"
            @click="handleDeleteProduct(scope.row)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页组件 -->
    <div class="pagination-container">
      <el-pagination
        v-model:current-page="productPage"
        :page-size="productPageSize"
        :total="filteredProducts.length"
        layout="total, prev, pager, next"
        background
      />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  productList, productSearch, productPage, productPageSize,
  filteredProducts, paginatedProducts, currentUser,
  showProductDialog, editingProduct, productForm,
  showDetailDialog, selectedProduct, supplyHistory,
  filterProducts, resetProductForm, getActionText
} from '../store'
import {
  getProductsAPI, getProductDetailAPI, getProductHistoryAPI,
  deleteProductAPI
} from '../api'

const loading = ref(false)

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

/** 查看产品详情 */
async function handleViewDetail(row) {
  selectedProduct.value = row
  try {
    const res = await getProductHistoryAPI(row.Product_Id)
    supplyHistory.value = res.data || []
  } catch (error) {
    supplyHistory.value = row.Supply_History || []
  }
  showDetailDialog.value = true
}

/** 查看供应链历史 */
function handleViewSupply(row) {
  handleViewDetail(row)
}

/** 删除产品 */
async function handleDeleteProduct(row) {
  try {
    await ElMessageBox.confirm(
      `确定要删除产品「${row.Name}」吗？此操作不可恢复。`,
      '删除确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await deleteProductAPI(row.Product_Id)
    ElMessage.success('产品已成功删除')
    await fetchProducts()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除产品失败:', error)
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
.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}
</style>