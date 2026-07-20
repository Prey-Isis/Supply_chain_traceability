// api.js - 所有后端 API 请求封装
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { token, clearLoginState } from './store'

// 创建 axios 实例
const api = axios.create({
  baseURL: '/api/v1',
  timeout: 15000
})

// ==================== 请求拦截器：自动添加 Token ====================
api.interceptors.request.use(config => {
  if (token.value) {
    config.headers.Authorization = `Bearer ${token.value}`
  }
  return config
}, error => {
  return Promise.reject(error)
})

// ==================== 响应拦截器：统一处理错误 ====================
api.interceptors.response.use(
  response => {
    const res = response.data
    if (res.code !== 0) {
      ElMessage.error(res.message || '请求失败')
      return Promise.reject(new Error(res.message))
    }
    return res
  },
  error => {
    if (error.response?.status === 401) {
      ElMessage.error('登录已过期，请重新登录')
      clearLoginState()
    } else if (error.response?.status === 403) {
      ElMessage.error('权限不足，无法执行此操作')
    } else if (error.code === 'ECONNABORTED') {
      ElMessage.error('请求超时，请重试')
    } else {
      ElMessage.error(error.message || '网络错误，请检查连接')
    }
    return Promise.reject(error)
  }
)

// ==================== 用户认证 API ====================
/** 用户登录 */
export function loginAPI(account, password) {
  return api.post('/login', { Account: account, PassWord: password })
}

/** 刷新令牌 */
export function refreshTokenAPI() {
  return api.post('/refresh-token')
}

/** 用户登出 */
export function logoutAPI() {
  return api.post('/logout')
}

/** 获取当前用户信息 */
export function getCurrentUserAPI() {
  return api.get('/user/current')
}

// ==================== 产品管理 API（公开） ====================
/** 获取产品列表 */
export function getProductsAPI() {
  return api.get('/products/concurrent')
}

/** 获取单个产品详情 */
export function getProductDetailAPI(productId) {
  return api.get(`/products/concurrent/${productId}`)
}

/** 获取产品供应链历史 */
export function getProductHistoryAPI(productId) {
  return api.get(`/products/${productId}/history`)
}

/** 获取所有供应历史 */
export function getAllSupplyHistoryAPI() {
  return api.get('/supply-history')
}

// ==================== 产品管理 API（需认证） ====================
/** 创建产品 */
export function createProductAPI(data) {
  return api.post('/products', data)
}

/** 更新产品 */
export function updateProductAPI(productId, data) {
  return api.put(`/products/${productId}`, data)
}

/** 更新产品状态 */
export function updateProductStatusAPI(productId, status) {
  return api.patch(`/products/${productId}/status`, { Status: status })
}

/** 删除产品（仅管理员） */
export function deleteProductAPI(productId) {
  return api.delete(`/products/${productId}`)
}

/** 创建单条供应链记录 */
export function createSupplyHistoryAPI(data) {
  return api.post('/supply-history', data)
}

/** 批量创建供应链记录 */
export function batchCreateSupplyHistoryAPI(data) {
  return api.post('/supply-history/batch', data)
}

// ==================== 用户管理 API（仅管理员） ====================
/** 获取用户列表 */
export function getUsersAPI() {
  return api.get('/admin/users')
}

/** 获取用户详情 */
export function getUserDetailAPI(account) {
  return api.get(`/admin/users/${account}`)
}

/** 创建用户 */
export function createUserAPI(data) {
  return api.post('/admin/users', data)
}

/** 更新用户 */
export function updateUserAPI(account, data) {
  return api.put(`/admin/users/${account}`, data)
}

/** 删除用户 */
export function deleteUserAPI(account) {
  return api.delete(`/admin/users/${account}`)
}

export default api