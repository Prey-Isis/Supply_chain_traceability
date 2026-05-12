
# 需求：生成一个单文件Vue3应用

我需要一个Vue3应用，实现以下功能：

## 技术栈
- Vue 3 + Composition API
- Axios
- 不需要路由，用条件渲染切换"页面"
- 不需要Pinia，用Vue的响应式变量管理状态
- UI框架：Element Plus（或你自己喜欢的）

## 后端API

## 基础信息
- Base URL: `/api/v1`
- 认证方式: Bearer Token (JWT)
- Token刷新: `/api/v1/refresh-token`

## 具体接口

### 1. 用户认证
| 方法 | 路径 | 功能 | 是否需要认证 |
|------|------|------|-------------|
| POST | /login | 用户登录 | 否 |
| POST | /refresh-token | 刷新令牌 | 否 |
| POST | /logout | 登出 | 是 |
| GET | /user/current | 获取当前用户信息 | 是 |

**请求示例：**
```json
// POST http://localhost:8080/api/v1/login
{
    "Account": "11111111",
    "PassWord": "123456"
}

// 响应示例
{
    "code": 0,
    "message": "登录成功",
    "data": {
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50IjoiMTExMTExMTEiLCJyb2xlIjoiYWRtaW4iLCJleHAiOjE3NzY4Mzc5NjcsImlhdCI6MTc3Njc1MTU2NywiaXNzIjoic3VwcGx5LWNoYWluLXN5c3RlbSJ9.Sukx-_Q_4M1IWLDE98DjKh1kASFiM7x3x43TsTb5wdw",
        "account": "11111111",
        "userName": "系统管理员",
        "role": "admin",
        "expiresIn": 1776837967
    }
}

// POST http://localhost:8080/api/v1/refresh-token
{}

// 响应示例
{
    "code": 0,
    "message": "token刷新成功",
    "data": {
        "account": "11111111",
        "expiresIn": 1776837967,
        "role": "admin",
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50IjoiMTExMTExMTEiLCJyb2xlIjoiYWRtaW4iLCJleHAiOjE3NzY4Mzc5NjcsImlhdCI6MTc3Njc1MTU2NywiaXNzIjoic3VwcGx5LWNoYWluLXN5c3RlbSJ9.Sukx-_Q_4M1IWLDE98DjKh1kASFiM7x3x43TsTb5wdw"
    }
}

// GET http://localhost:8080/api/v1/user/current
{}

// 响应示例
{
    "code": 0,
    "message": "获取用户信息成功",
    "data": {
        "account": "11111111",
        "role": "admin",
        "userName": "系统管理员"
    }
}

// POST http://localhost:8080/api/v1/logout
{}

// 响应示例
{
    "code": 0,
    "message": "登出成功",
    "data": null
}
```

### 2. 产品管理（公开）
| 方法 | 路径 | 功能 | 是否需要认证 |
|------|------|------|-------------|
| GET | /products | 获取产品列表及其供应链记录 | 否（可选） |
| GET | /products/:product_id | 获取单个产品详情 | 否（可选） |
| GET | /products/:product_id/history | 获取单个产品供应链记录 | 否（可选） |
| GET | /supply-history | 获取所有供应历史 | 否（可选） |

**请求示例：**
```json
// POST http://localhost:8080/api/v1/products
{}

// 响应示例
{
    "code": 0,
    "message": "获取产品列表成功",
    "data": [
        {
            "Product_Id": "2",
            "Name": "苹果",
            "Current_Holder": "果园B",
            "Status": "1",
            "Create_Time": "2026-04-10T16:53:53+08:00",
            "Update_Time": "2026-04-10T17:18:10+08:00",
            "Supply_History": [
                {
                    "Product_Id": "2",
                    "Product_Name": "苹果",
                    "Node_Name": "果园B",
                    "Location": "湖南长沙",
                    "Action": "1",
                    "Operation_Role": "00000002",
                    "Description": "地里随便长的",
                    "Create_Time": "2026-04-10T16:53:53+08:00"
                },
                {
                    "Product_Id": "2",
                    "Product_Name": "苹果醋",
                    "Node_Name": "加工厂A",
                    "Location": "湖南长沙",
                    "Action": "2",
                    "Operation_Role": "00000002",
                    "Description": "随机加工的",
                    "Create_Time": "2026-04-10T17:22:51+08:00"
                },
                {
                    "Product_Id": "2",
                    "Product_Name": "苹果醋",
                    "Node_Name": "物流A",
                    "Location": "湖南长沙",
                    "Action": "3",
                    "Operation_Role": "00000002",
                    "Description": "转运",
                    "Create_Time": "2026-04-10T17:26:35+08:00"
                },
                {
                    "Product_Id": "2",
                    "Product_Name": "苹果醋",
                    "Node_Name": "超市A",
                    "Location": "湖南长沙",
                    "Action": "4",
                    "Operation_Role": "00000002",
                    "Description": "上架零售",
                    "Create_Time": "2026-04-10T17:26:35+08:00"
                },
                {
                    "Product_Id": "2",
                    "Product_Name": "苹果醋",
                    "Node_Name": "物流A",
                    "Location": "湖南长沙",
                    "Action": "3",
                    "Operation_Role": "00000002",
                    "Description": "转运",
                    "Create_Time": "2026-04-10T17:26:52+08:00"
                },
                {
                    "Product_Id": "2",
                    "Product_Name": "苹果醋",
                    "Node_Name": "超市A",
                    "Location": "湖南长沙",
                    "Action": "4",
                    "Operation_Role": "00000002",
                    "Description": "上架零售",
                    "Create_Time": "2026-04-10T17:26:52+08:00"
                }
            ]
        }
    ]
}

// GET http://localhost:8080/api/v1/products/2
{}

// 响应示例
{
    "code": 0,
    "message": "获取产品成功",
    "data": {
        "Product_Id": "2",
        "Name": "苹果",
        "Current_Holder": "果园B",
        "Status": "1",
        "Create_Time": "2026-04-10T16:53:53+08:00",
        "Update_Time": "2026-04-10T17:18:10+08:00",
        "Supply_History": [
            {
                "Product_Id": "2",
                "Product_Name": "苹果",
                "Node_Name": "果园B",
                "Location": "湖南长沙",
                "Action": "1",
                "Operation_Role": "00000002",
                "Description": "地里随便长的",
                "Create_Time": "2026-04-10T16:53:53+08:00"
            },
            {
                "Product_Id": "2",
                "Product_Name": "苹果醋",
                "Node_Name": "加工厂A",
                "Location": "湖南长沙",
                "Action": "2",
                "Operation_Role": "00000002",
                "Description": "随机加工的",
                "Create_Time": "2026-04-10T17:22:51+08:00"
            },
            {
                "Product_Id": "2",
                "Product_Name": "苹果醋",
                "Node_Name": "物流A",
                "Location": "湖南长沙",
                "Action": "3",
                "Operation_Role": "00000002",
                "Description": "转运",
                "Create_Time": "2026-04-10T17:26:35+08:00"
            },
            {
                "Product_Id": "2",
                "Product_Name": "苹果醋",
                "Node_Name": "超市A",
                "Location": "湖南长沙",
                "Action": "4",
                "Operation_Role": "00000002",
                "Description": "上架零售",
                "Create_Time": "2026-04-10T17:26:35+08:00"
            },
            {
                "Product_Id": "2",
                "Product_Name": "苹果醋",
                "Node_Name": "物流A",
                "Location": "湖南长沙",
                "Action": "3",
                "Operation_Role": "00000002",
                "Description": "转运",
                "Create_Time": "2026-04-10T17:26:52+08:00"
            },
            {
                "Product_Id": "2",
                "Product_Name": "苹果醋",
                "Node_Name": "超市A",
                "Location": "湖南长沙",
                "Action": "4",
                "Operation_Role": "00000002",
                "Description": "上架零售",
                "Create_Time": "2026-04-10T17:26:52+08:00"
            }
        ]
    }
}

// GET http://localhost:8080/api/v1/products/2/history
{}

// 响应示例
{
    "code": 0,
    "message": "获取供应链历史记录成功",
    "data": [
        {
            "Product_Id": "2",
            "Product_Name": "苹果",
            "Node_Name": "果园B",
            "Location": "湖南长沙",
            "Action": "1",
            "Operation_Role": "00000002",
            "Description": "地里随便长的",
            "Create_Time": "2026-04-10T16:53:53+08:00"
        },
        {
            "Product_Id": "2",
            "Product_Name": "苹果醋",
            "Node_Name": "加工厂A",
            "Location": "湖南长沙",
            "Action": "2",
            "Operation_Role": "00000002",
            "Description": "随机加工的",
            "Create_Time": "2026-04-10T17:22:51+08:00"
        },
        {
            "Product_Id": "2",
            "Product_Name": "苹果醋",
            "Node_Name": "物流A",
            "Location": "湖南长沙",
            "Action": "3",
            "Operation_Role": "00000002",
            "Description": "转运",
            "Create_Time": "2026-04-10T17:26:35+08:00"
        },
        {
            "Product_Id": "2",
            "Product_Name": "苹果醋",
            "Node_Name": "超市A",
            "Location": "湖南长沙",
            "Action": "4",
            "Operation_Role": "00000002",
            "Description": "上架零售",
            "Create_Time": "2026-04-10T17:26:35+08:00"
        },
        {
            "Product_Id": "2",
            "Product_Name": "苹果醋",
            "Node_Name": "物流A",
            "Location": "湖南长沙",
            "Action": "3",
            "Operation_Role": "00000002",
            "Description": "转运",
            "Create_Time": "2026-04-10T17:26:52+08:00"
        },
        {
            "Product_Id": "2",
            "Product_Name": "苹果醋",
            "Node_Name": "超市A",
            "Location": "湖南长沙",
            "Action": "4",
            "Operation_Role": "00000002",
            "Description": "上架零售",
            "Create_Time": "2026-04-10T17:26:52+08:00"
        }
    ]
}

// GET http://localhost:8080/api/v1/products
{}

// 响应示例
{
    "code": 0,
    "message": "获取产品列表成功",
    "data": [
        {
            "Product_Id": "2",
            "Name": "苹果",
            "Current_Holder": "果园B",
            "Status": "1",
            "Create_Time": "2026-04-10T16:53:53+08:00",
            "Update_Time": "2026-04-10T17:18:10+08:00",
            "Supply_History": [
                {
                    "Product_Id": "2",
                    "Product_Name": "苹果",
                    "Node_Name": "果园B",
                    "Location": "湖南长沙",
                    "Action": "1",
                    "Operation_Role": "00000002",
                    "Description": "地里随便长的",
                    "Create_Time": "2026-04-10T16:53:53+08:00"
                },
                {
                    "Product_Id": "2",
                    "Product_Name": "苹果醋",
                    "Node_Name": "加工厂A",
                    "Location": "湖南长沙",
                    "Action": "2",
                    "Operation_Role": "00000002",
                    "Description": "随机加工的",
                    "Create_Time": "2026-04-10T17:22:51+08:00"
                },
                {
                    "Product_Id": "2",
                    "Product_Name": "苹果醋",
                    "Node_Name": "物流A",
                    "Location": "湖南长沙",
                    "Action": "3",
                    "Operation_Role": "00000002",
                    "Description": "转运",
                    "Create_Time": "2026-04-10T17:26:35+08:00"
                },
                {
                    "Product_Id": "2",
                    "Product_Name": "苹果醋",
                    "Node_Name": "超市A",
                    "Location": "湖南长沙",
                    "Action": "4",
                    "Operation_Role": "00000002",
                    "Description": "上架零售",
                    "Create_Time": "2026-04-10T17:26:35+08:00"
                },
                {
                    "Product_Id": "2",
                    "Product_Name": "苹果醋",
                    "Node_Name": "物流A",
                    "Location": "湖南长沙",
                    "Action": "3",
                    "Operation_Role": "00000002",
                    "Description": "转运",
                    "Create_Time": "2026-04-10T17:26:52+08:00"
                },
                {
                    "Product_Id": "2",
                    "Product_Name": "苹果醋",
                    "Node_Name": "超市A",
                    "Location": "湖南长沙",
                    "Action": "4",
                    "Operation_Role": "00000002",
                    "Description": "上架零售",
                    "Create_Time": "2026-04-10T17:26:52+08:00"
                }
            ]
        }
    ]
}
```

### 3. 产品管理（需认证）
| 方法 | 路径 | 功能 | 所需角色 |
|------|------|------|----------|
| POST | /products | 创建产品 | 普通用户 |
| POST | /supply-history | 创建单条供应链记录 | 普通用户 |
| POST | /supply-history/batch | 批量创建供应链记录 | 普通用户 |
| PUT | /products/:product_id | 更新产品 | 普通用户 |
| PATCH | /products/:product_id/status | 更新产品状态 | 普通用户 |
| DELETE | /products/:product_id | 删除产品 | 管理员 |

**请求示例：**
```json
// POST http://localhost:8080/api/v1/products
{
    "Product_Id": "3",
    "Name": "苹果",
    "Current_Holder": "果园B",
    "Status": "生产",
    "Supply_History": [
        {
            "Product_Id": "2",
            "Product_Name": "苹果",
            "Node_Name": "果园B",
            "Location": "湖南长沙",
            "Action": "生产",
            "Operation_Role": "00000002",
            "Description": "地里随便长的"
        }
    ]
}

// 响应示例
{
    "code": 0,
    "message": "产品创建成功",
    "data": {
        "Product_Id": "3",
        "Name": "苹果",
        "Current_Holder": "果园B",
        "Status": "生产",
        "Create_Time": "",
        "Update_Time": "",
        "Supply_History": [
            {
                "Product_Id": "3",
                "Product_Name": "苹果",
                "Node_Name": "果园B",
                "Location": "湖南长沙",
                "Action": "生产",
                "Operation_Role": "00000002",
                "Description": "地里随便长的",
                "Create_Time": ""
            }
        ]
    }
}

// POST http://localhost:8080/api/v1/supply-history
{
    "Product_Id": "3",
    "Product_Name": "苹果醋",
    "Node_Name": "加工厂A",
    "Location": "湖南长沙",
    "Action": "2",
    "Operation_Role": "00000002",
    "Description": "随机加工的"
}

// 响应示例
{
    "code": 0,
    "message": "供应链历史记录创建成功",
    "data": {
        "Product_Id": "3",
        "Product_Name": "苹果醋",
        "Node_Name": "加工厂A",
        "Location": "湖南长沙",
        "Action": "2",
        "Operation_Role": "00000002",
        "Description": "随机加工的",
        "Create_Time": ""
    }
}

// POST http://localhost:8080/api/v1/supply-history/batch
{
    "Histories": [
            {
                "Product_Id": "3",
                "Product_Name": "苹果醋",
                "Node_Name": "物流A",
                "Location": "湖南长沙",
                "Action": "3",
                "Operation_Role": "00000002",
                "Description": "转运"
            },
            {
                "Product_Id": "3",
                "Product_Name": "苹果醋",
                "Node_Name": "超市A",
                "Location": "湖南长沙",
                "Action": "4",
                "Operation_Role": "00000002",
                "Description": "上架零售"
            }
        ]
}

// 响应示例
{
    "code": 0,
    "message": "批量创建供应链历史记录成功",
    "data": [
        {
            "Product_Id": "3",
            "Product_Name": "苹果醋",
            "Node_Name": "物流A",
            "Location": "湖南长沙",
            "Action": "3",
            "Operation_Role": "00000002",
            "Description": "转运",
            "Create_Time": ""
        },
        {
            "Product_Id": "3",
            "Product_Name": "苹果醋",
            "Node_Name": "超市A",
            "Location": "湖南长沙",
            "Action": "4",
            "Operation_Role": "00000002",
            "Description": "上架零售",
            "Create_Time": ""
        }
    ]
}

// PUT http://localhost:8080/api/v1/products/2
{
    "Name": "苹果",
    "Current_Holder": "果园B",
    "Status": "生产"
}

// 响应示例
{
    "code": 0,
    "message": "产品更新成功",
    "data": {
        "Product_Id": "2",
        "Name": "苹果",
        "Current_Holder": "果园B",
        "Status": "生产",
        "Create_Time": "",
        "Update_Time": "",
        "Supply_History": null
    }
}

// PATCH http://localhost:8080/api/v1/products/2/status
{
    "Status": "生产完成，待出库"
}

// 响应示例
{
    "code": 0,
    "message": "产品状态更新成功",
    "data": {
        "product_id": "2",
        "status": "生产完成，待出库"
    }
}

// DELETE http://localhost:8080/api/v1/products/3
{}

// 响应示例
{
    "code": 0,
    "message": "产品删除成功",
    "data": null
}
```

### 4. 用户管理（仅管理员）
| 方法 | 路径 | 功能 |
|------|------|------|
| POST | /admin/users | 创建用户 |
| GET | /admin/users | 获取用户列表 |
| GET | /admin/users/:account | 获取用户详情 |
| PUT | /admin/users/:account | 更新用户 |
| DELETE | /admin/users/:account | 删除用户 |

**请求示例：**
```json
// POST http://localhost:8080/api/v1/admin/users
{
    "UserName": "普通删除用户1",
    "Account": "00000003",
    "PassWord": "00000003",
    "Role": "user"
}

// 响应示例
{
    "code": 0,
    "message": "用户创建成功",
    "data": {
        "UserName": "普通删除用户1",
        "Account": "00000003",
        "PassWord": "",
        "Role": "user",
        "Create_Time": "",
        "Update_Time": ""
    }
}

// GET http://localhost:8080/api/v1/admin/users
{}

// 响应示例
{
    "code": 0,
    "message": "获取用户列表成功",
    "data": [
        {
            "UserName": "普通删除用户1",
            "Account": "00000003",
            "PassWord": "",
            "Role": "user",
            "Create_Time": "2026-04-21T14:31:52+08:00",
            "Update_Time": "2026-04-21T14:31:52+08:00"
        },
        {
            "UserName": "普通用户1",
            "Account": "00000001",
            "PassWord": "",
            "Role": "user",
            "Create_Time": "2026-04-08T15:26:55+08:00",
            "Update_Time": "2026-04-08T15:35:46+08:00"
        },
        {
            "UserName": "系统管理员",
            "Account": "11111111",
            "PassWord": "",
            "Role": "admin",
            "Create_Time": "2026-03-27T17:05:53+08:00",
            "Update_Time": "2026-03-27T17:05:53+08:00"
        }
    ]
}

// GET http://localhost:8080/api/v1/admin/users/00000001
{}

// 响应示例
{
    "code": 0,
    "message": "获取用户成功",
    "data": {
        "UserName": "普通用户1",
        "Account": "00000001",
        "PassWord": "",
        "Role": "user",
        "Create_Time": "2026-04-08T15:26:55+08:00",
        "Update_Time": "2026-04-08T15:35:46+08:00"
    }
}

// PUT http://localhost:8080/api/v1/admin/users/00000001
{
    "UserName": "普通VIP用户1",
    "PassWord": "00000001",
    "Role": "user"
}

// 响应示例
{
    "code": 0,
    "message": "用户更新成功",
    "data": {
        "UserName": "普通VIP用户1",
        "Account": "00000001",
        "PassWord": "",
        "Role": "user",
        "Create_Time": "",
        "Update_Time": ""
    }
}
// 如果修改前后信息一致将有如下响应
{
    "code": 1,
    "message": "更新用户失败：user not found",
    "data": null
}

// DELETE http://localhost:8080/api/v1/admin/users/00000003
{}

// 响应示例
{
    "code": 0,
    "message": "用户删除成功",
    "data": null
}
```

**403响应示例：**
```json
{
    "code": 1,
    "message": "权限不足，需要角色：admin",
    "data": null
}
```
**404响应示例：**
```json
{
    "code": 1,
    "message": "产品不存在",
    "data": null
}
```

## 需要实现的功能

### 1. 登录/认证
- 未登录时显示登录表单
- 登录后保存token到localStorage
- 自动在请求头添加 `Authorization: Bearer {token}`

### 2. 主布局（登录后显示）
- 顶部导航栏：显示用户名、角色、登出按钮
- 侧边菜单：根据角色显示不同菜单项
  - 普通用户：产品管理
  - 管理员：产品管理 + 用户管理
  - 经理：产品管理 + 报表查看

### 3. 产品列表页面
- 展示产品表格（ID、名称、状态、操作按钮）
- 支持搜索框（按产品名称）
- 支持分页
- 按钮：
  - 查看详情（弹窗或展开）
  - 编辑（管理员和普通用户）
  - 删除（仅管理员）
  - 新增产品（按钮在表格上方）

### 4. 用户管理页面（仅管理员可见）
- 展示用户表格（账号、角色、创建时间）
- 支持添加用户（表单弹窗）
- 支持编辑用户角色
- 支持删除用户

### 5. 产品表单
- 创建/编辑产品使用弹窗Dialog
- 字段：产品名称、描述、价格、状态

## 代码要求
- **所有代码写在一个 `App.vue` 文件中**
- 使用 `<script setup>` 语法
- 使用 `ref`、`reactive`、`computed`、`onMounted`
- 组件内部管理状态（不需要外部store）
- 样式使用scoped CSS，保持整洁
- 添加足够的注释说明

## 文件说明
可以有多个页面文件，但脚本代码需要在一个文件中

## 额外说明
- 用条件变量 currentPage 控制显示哪个"页面"（'products' / 'users' / 'reports'）
- 用 showProductDialog 控制产品表单弹窗
- API调用封装成函数，统一处理错误提示
- token过期时（401响应）自动清除token并跳转到登录界面
- 使用Element Plus UI库
- 实现登录、产品CRUD、用户管理（管理员）
- 根据角色显示不同菜单
- 不需要路由，用v-if切换页面
- 加上详细的中文注释
- 告诉我如何运行这个项目

### main.js
```javascript
import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'

const app = createApp(App)
app.use(ElementPlus)
app.mount('#app')