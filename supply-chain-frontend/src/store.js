// store.js - 全局响应式状态管理
import { ref, reactive, computed } from 'vue'

// ==================== 用户认证状态 ====================
export const isLoggedIn = ref(false)
export const token = ref('')
export const currentUser = reactive({
  account: '',
  userName: '',
  role: ''
})

// ==================== 页面切换状态 ====================
export const currentPage = ref('products')

// ==================== 产品状态 ====================
export const productList = ref([])
export const productSearch = ref('')
export const productPage = ref(1)
export const productPageSize = ref(10)

// 搜索过滤后的产品列表
export const filteredProducts = computed(() => {
  if (!productSearch.value) return productList.value
  return productList.value.filter(p =>
    p.Name && p.Name.includes(productSearch.value)
  )
})

// 分页后的产品列表
export const paginatedProducts = computed(() => {
  const start = (productPage.value - 1) * productPageSize.value
  return filteredProducts.value.slice(start, start + productPageSize.value)
})

// ==================== 产品对话框状态 ====================
export const showProductDialog = ref(false)
export const editingProduct = ref(null)
export const productForm = reactive({
  Product_Id: '',
  Name: '',
  Current_Holder: '',
  Status: 'produced' // 从 '1' 改为 'produced'（已生产）
})
export const formLoading = ref(false)

// ==================== 产品详情状态 ====================
export const showDetailDialog = ref(false)
export const selectedProduct = ref(null)
export const supplyHistory = ref([])

// ==================== 用户管理状态 ====================
export const userList = ref([])
export const showUserDialog = ref(false)
export const editingUser = ref(null)
export const userForm = reactive({
  Account: '',
  UserName: '',
  PassWord: '',
  Role: 'user'
})
export const userFormLoading = ref(false)

// ==================== 辅助函数 ====================
// 从 localStorage 恢复登录状态
export function restoreLoginState() {
  const savedToken = localStorage.getItem('token')
  const savedUser = localStorage.getItem('userInfo')
  if (savedToken && savedUser) {
    try {
      const user = JSON.parse(savedUser)
      token.value = savedToken
      Object.assign(currentUser, user)
      isLoggedIn.value = true
      return true
    } catch (e) {
      localStorage.clear()
    }
  }
  return false
}

// 清除登录状态
export function clearLoginState() {
  token.value = ''
  localStorage.removeItem('token')
  localStorage.removeItem('userInfo')
  isLoggedIn.value = false
  currentPage.value = 'products'
  currentUser.account = ''
  currentUser.userName = ''
  currentUser.role = ''
}

// 筛选产品时重置页码
export function filterProducts() {
  productPage.value = 1
}

// 重置产品表单
export function resetProductForm() {
  Object.assign(productForm, {
    Product_Id: '',
    Name: '',
    Current_Holder: '',
    Status: '1'
  })
  editingProduct.value = null
}

// 重置用户表单
export function resetUserForm() {
  Object.assign(userForm, {
    Account: '',
    UserName: '',
    PassWord: '',
    Role: 'user'
  })
  editingUser.value = null
}

// 操作类型映射
export function getActionText(action) {
  const map = {
    '1': '生产',
    '2': '加工',
    '3': '运输',
    '4': '上架'
  }
  return map[action] || action
}