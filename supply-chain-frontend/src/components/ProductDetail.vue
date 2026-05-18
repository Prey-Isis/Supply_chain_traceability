<!-- components/ProductDetail.vue - 产品详情与供应链历史弹窗 -->
<template>
  <el-dialog
    v-model="showDetailDialog"
    title="产品详情"
    width="800px"
    @close="handleClose"
  >
    <template v-if="selectedProduct">
      <!-- 产品基本信息 -->
      <el-descriptions :column="2" border>
        <el-descriptions-item label="产品ID" label-align="right">
          {{ selectedProduct.Product_Id }}
        </el-descriptions-item>
        <el-descriptions-item label="产品名称" label-align="right">
          {{ selectedProduct.Name }}
        </el-descriptions-item>
        <el-descriptions-item label="当前持有者" label-align="right">
          {{ selectedProduct.Current_Holder || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="当前状态" label-align="right">
          <el-tag :type="getStatusTagType(selectedProduct.Status)" size="large">
            {{ getStatusIcon(selectedProduct.Status) }} {{ getStatusText(selectedProduct.Status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="创建时间" label-align="right">
          {{ selectedProduct.Create_Time || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="更新时间" label-align="right">
          {{ selectedProduct.Update_Time || '-' }}
        </el-descriptions-item>
      </el-descriptions>

      <!-- 供应链流程图 -->
      <h4 class="section-title">🔄 供应链流程追踪</h4>
      <div class="supply-chain-flow">
        <el-steps :active="getCurrentStep(selectedProduct.Status)" finish-status="success" align-center>
          <el-step title="生产者" description="产品生产" icon="🏭" />
          <el-step title="加工商" description="产品加工" icon="⚙️" />
          <el-step title="分销商" description="物流分销" icon="🚚" />
          <el-step title="零售商" description="终端零售" icon="🏪" />
          <el-step title="消费者" description="最终购买" icon="🎉" />
        </el-steps>
      </div>

      <!-- 供应链历史记录 -->
      <div class="supply-history-header">
        <h4 class="section-title">📦 供应链详细记录</h4>
        <el-button type="primary" size="small" @click="openAddSupplyDialog">
          <el-icon><Plus /></el-icon> 添加记录
        </el-button>
      </div>
      
      <div v-if="supplyHistory.length > 0" class="timeline-container">
        <el-timeline>
          <el-timeline-item
            v-for="(item, index) in supplyHistory"
            :key="index"
            :timestamp="formatTime(item.Create_Time)"
            placement="top"
            :color="getTimelineColor(item.Action)"
          >
            <el-card shadow="hover" class="history-card">
              <!-- 记录头部：显示节点名称和操作类型 -->
              <div class="history-header">
                <div class="node-info">
                  <span class="node-icon">{{ getNodeIcon(item.Node_Name) }}</span>
                  <span class="node-name">{{ item.Node_Name || '-' }}</span>
                </div>
                <span class="action-tag">
                  <el-tag size="small" :type="getActionTagType(item.Action)">
                    {{ getActionText(item.Action) }}
                  </el-tag>
                </span>
              </div>
              
              <!-- 记录详情 -->
              <div class="history-body">
                <el-row :gutter="16">
                  <el-col :span="12">
                    <p><strong>位置：</strong>{{ item.Location || '-' }}</p>
                  </el-col>
                  <el-col :span="12">
                    <p><strong>产品名称：</strong>{{ item.Product_Name || '-' }}</p>
                  </el-col>
                </el-row>
                <p><strong>描述：</strong>{{ item.Description || '无' }}</p>
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </div>
      <el-empty v-else description="暂无供应链记录" />
    </template>
    <template v-else>
      <el-empty description="未选择产品" />
    </template>

    <!-- 添加供应链记录的子对话框 -->
    <el-dialog
      v-model="showAddSupplyDialog"
      title="添加供应链记录"
      width="500px"
      append-to-body
      :close-on-click-modal="false"
      @close="resetSupplyForm"
    >
      <el-form :model="supplyForm" label-width="100px">
        <el-form-item label="产品ID">
          <el-input :model-value="selectedProduct?.Product_Id" disabled />
        </el-form-item>
        
        <el-form-item label="产品名称">
          <el-input 
            v-model="supplyForm.Product_Name" 
            placeholder="请输入产品名称"
            maxlength="50"
          />
        </el-form-item>
        
        <el-form-item label="节点名称" required>
          <el-input 
            v-model="supplyForm.Node_Name" 
            placeholder="如：果园A、加工厂B、物流C"
            maxlength="50"
          />
        </el-form-item>
        
        <el-form-item label="位置">
          <el-input 
            v-model="supplyForm.Location" 
            placeholder="如：湖南长沙"
            maxlength="100"
          />
        </el-form-item>
        
        <el-form-item label="操作类型" required>
          <el-select v-model="supplyForm.Action" placeholder="请选择操作类型" style="width: 100%">
            <el-option label="生产" value="1">
              <span>🏭 生产</span>
            </el-option>
            <el-option label="加工" value="2">
              <span>⚙️ 加工</span>
            </el-option>
            <el-option label="运输" value="3">
              <span>🚚 运输</span>
            </el-option>
            <el-option label="上架" value="4">
              <span>🏪 上架</span>
            </el-option>
            <el-option label="售出" value="5">
              <span>🎉 售出</span>
            </el-option>
          </el-select>
        </el-form-item>
        
        <el-form-item label="操作角色">
          <el-select v-model="supplyForm.Operation_Role" placeholder="请选择操作角色" style="width: 100%">
            <el-option label="生产者" value="00000001" />
            <el-option label="加工商" value="00000002" />
            <el-option label="分销商" value="00000003" />
            <el-option label="零售商" value="00000004" />
            <el-option label="消费者" value="00000005" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="描述">
          <el-input
            v-model="supplyForm.Description"
            type="textarea"
            :rows="3"
            placeholder="请输入操作描述"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="showAddSupplyDialog = false">取消</el-button>
        <el-button type="primary" @click="submitSupplyRecord" :loading="supplyLoading">
          确认添加
        </el-button>
      </template>
    </el-dialog>
  </el-dialog>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { 
  showDetailDialog, selectedProduct, supplyHistory, 
  getActionText, productList 
} from '../store'
import { 
  createSupplyHistoryAPI, 
  getProductHistoryAPI, 
  getProductsAPI 
} from '../api'

// ==================== 添加供应链记录相关 ====================
const showAddSupplyDialog = ref(false)
const supplyLoading = ref(false)

// 供应链记录表单
const supplyForm = reactive({
  Product_Name: '',
  Node_Name: '',
  Location: '',
  Action: '',
  Operation_Role: '',
  Description: ''
})

// 打开添加供应链记录对话框
function openAddSupplyDialog() {
  // 预填充产品相关信息
  supplyForm.Product_Name = selectedProduct.value?.Name || ''
  supplyForm.Node_Name = ''
  supplyForm.Location = ''
  supplyForm.Action = ''
  supplyForm.Operation_Role = ''
  supplyForm.Description = ''
  
  showAddSupplyDialog.value = true
}

// 重置供应链表单
function resetSupplyForm() {
  supplyForm.Product_Name = ''
  supplyForm.Node_Name = ''
  supplyForm.Location = ''
  supplyForm.Action = ''
  supplyForm.Operation_Role = ''
  supplyForm.Description = ''
}

// 提交供应链记录
async function submitSupplyRecord() {
  // 表单验证
  if (!supplyForm.Node_Name.trim()) {
    ElMessage.warning('请输入节点名称')
    return
  }
  
  if (!supplyForm.Action) {
    ElMessage.warning('请选择操作类型')
    return
  }

  supplyLoading.value = true
  try {
    await createSupplyHistoryAPI({
      Product_Id: selectedProduct.value.Product_Id,
      Product_Name: supplyForm.Product_Name || selectedProduct.value.Name,
      Node_Name: supplyForm.Node_Name,
      Location: supplyForm.Location,
      Action: supplyForm.Action,
      Operation_Role: supplyForm.Operation_Role,
      Description: supplyForm.Description
    })
    
    ElMessage.success('供应链记录添加成功')
    showAddSupplyDialog.value = false
    
    // 刷新供应链历史记录
    await refreshSupplyHistory()
  } catch (error) {
    console.error('添加供应链记录失败:', error)
  } finally {
    supplyLoading.value = false
  }
}

// 刷新供应链历史
async function refreshSupplyHistory() {
  try {
    const res = await getProductHistoryAPI(selectedProduct.value.Product_Id)
    supplyHistory.value = res.data || []
  } catch (error) {
    // 保持现有数据
  }
}

// 关闭详情弹窗时的处理
function handleClose() {
  showAddSupplyDialog.value = false
  resetSupplyForm()
}

// ==================== 状态映射相关 ====================
// 状态选项映射
const statusMap = {
  'produced': { text: '已生产', icon: '🏭', tagType: 'primary' },
  'processing': { text: '加工中', icon: '⚙️', tagType: 'warning' },
  'processed': { text: '已加工', icon: '✅', tagType: 'success' },
  'distributing': { text: '分销中', icon: '🚚', tagType: 'danger' },
  'retailing': { text: '零售中', icon: '🏪', tagType: 'info' },
  'sold': { text: '已售出', icon: '🎉', tagType: 'success' },
  'returned': { text: '已退货', icon: '↩️', tagType: 'danger' },
  '1': { text: '在售', icon: '🏪', tagType: 'success' },
  '0': { text: '下架', icon: '📦', tagType: 'info' }
}

// 节点图标映射
function getNodeIcon(nodeName) {
  if (!nodeName) return '📍'
  
  const iconMap = {
    '果园': '🌳',
    '农场': '🌾',
    '工厂': '🏭',
    '加工厂': '⚙️',
    '物流': '🚚',
    '快递': '📦',
    '超市': '🏪',
    '商店': '🛒',
    '仓库': '🏗️',
    '批发': '📊',
    '零售': '💳'
  }
  
  for (const [key, icon] of Object.entries(iconMap)) {
    if (nodeName.includes(key)) {
      return icon
    }
  }
  
  return '📍'
}

// 获取状态文本
function getStatusText(status) {
  return statusMap[status]?.text || status || '未知状态'
}

// 获取状态图标
function getStatusIcon(status) {
  return statusMap[status]?.icon || '📋'
}

// 获取状态标签类型
function getStatusTagType(status) {
  return statusMap[status]?.tagType || 'info'
}

// 获取当前供应链步骤
function getCurrentStep(status) {
  const stepMap = {
    'produced': 0,
    'processing': 1,
    'processed': 2,
    'distributing': 3,
    'retailing': 4,
    'sold': 5,
    'returned': -1
  }
  return stepMap[status] !== undefined ? stepMap[status] : 0
}

// 获取时间线颜色
function getTimelineColor(action) {
  const colorMap = {
    '1': '#409EFF',
    '2': '#E6A23C',
    '3': '#F56C6C',
    '4': '#67C23A',
    '5': '#909399'
  }
  return colorMap[action] || '#409EFF'
}

// 获取操作标签类型
function getActionTagType(action) {
  const typeMap = {
    '1': '',
    '2': 'warning',
    '3': 'danger',
    '4': 'success',
    '5': 'info'
  }
  return typeMap[action] || ''
}

// 格式化时间
function formatTime(timeStr) {
  if (!timeStr) return '-'
  return timeStr.replace('T', ' ').replace('+08:00', '')
}
</script>

<style scoped>
.section-title {
  margin: 0;
  font-size: 16px;
  color: #303133;
  border-left: 3px solid #409EFF;
  padding-left: 10px;
}

.supply-chain-flow {
  margin: 20px 0;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
}

.supply-history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 24px;
  margin-bottom: 16px;
}

.timeline-container {
  max-height: 400px;
  overflow-y: auto;
  padding-right: 10px;
}

.history-card {
  margin-bottom: 8px;
  transition: transform 0.2s ease;
}

.history-card:hover {
  transform: translateY(-2px);
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid #ebeef5;
}

.node-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.node-icon {
  font-size: 20px;
}

.node-name {
  font-size: 15px;
  font-weight: bold;
  color: #303133;
}

.action-tag {
  flex-shrink: 0;
}

.history-body {
  padding: 4px 0;
}

.history-body p {
  margin: 8px 0;
  font-size: 14px;
  line-height: 1.6;
  color: #606266;
}

.history-body strong {
  color: #303133;
  margin-right: 4px;
}

/* 自定义滚动条 */
.timeline-container::-webkit-scrollbar {
  width: 6px;
}

.timeline-container::-webkit-scrollbar-thumb {
  background: #dcdfe6;
  border-radius: 3px;
}

.timeline-container::-webkit-scrollbar-thumb:hover {
  background: #c0c4cc;
}

.timeline-container::-webkit-scrollbar-track {
  background: #f5f7fa;
}
</style>