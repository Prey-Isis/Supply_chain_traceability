<!-- components/ProductDetail.vue - 产品详情与供应链历史弹窗 -->
<template>
  <el-dialog
    v-model="showDetailDialog"
    title="产品详情"
    width="700px"
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
          {{ selectedProduct.Current_Holder }}
        </el-descriptions-item>
        <el-descriptions-item label="状态" label-align="right">
          <el-tag :type="selectedProduct.Status === '1' ? 'success' : 'info'">
            {{ selectedProduct.Status === '1' ? '在售' : '下架' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="创建时间" label-align="right">
          {{ selectedProduct.Create_Time || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="更新时间" label-align="right">
          {{ selectedProduct.Update_Time || '-' }}
        </el-descriptions-item>
      </el-descriptions>

      <!-- 供应链历史 -->
      <h4 class="section-title">📦 供应链历史记录</h4>
      <div v-if="supplyHistory.length > 0" class="timeline-container">
        <el-timeline>
          <el-timeline-item
            v-for="(item, index) in supplyHistory"
            :key="index"
            :timestamp="item.Create_Time || '-'"
            placement="top"
            :color="index === 0 ? '#409EFF' : '#67C23A'"
          >
            <el-card shadow="hover">
              <div class="history-item">
                <p><strong>节点名称：</strong>{{ item.Node_Name || '-' }}</p>
                <p><strong>位置：</strong>{{ item.Location || '-' }}</p>
                <p><strong>操作类型：</strong>
                  <el-tag size="small">{{ getActionText(item.Action) }}</el-tag>
                </p>
                <p><strong>产品名称：</strong>{{ item.Product_Name || '-' }}</p>
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
  </el-dialog>
</template>

<script setup>
import { showDetailDialog, selectedProduct, supplyHistory, getActionText } from '../store'
</script>

<style scoped>
.section-title {
  margin-top: 24px;
  margin-bottom: 16px;
  font-size: 16px;
  color: #303133;
  border-left: 3px solid #409EFF;
  padding-left: 10px;
}
.timeline-container {
  max-height: 400px;
  overflow-y: auto;
}
.history-item p {
  margin: 6px 0;
  font-size: 14px;
}
</style>