import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 解决跨域问题
export default defineConfig({
  plugins: [vue()],
  // ★ 部署路径前缀：构建产物中所有资源引用都会带 /supply_chain/ 前缀
  //   例如 /supply_chain/assets/index-xxx.js
  //   用于一个服务器部署多个项目，通过路径区分
  base: '/supply_chain/',
  server: {
    port: 3000,
    proxy: {
      // ★ 开发环境代理也要匹配新前缀
      //   /supply_chain/api/xxx → 去掉前缀 → http://localhost:8080/api/xxx
      '/supply_chain/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: path => path.replace(/^\/supply_chain/, '')
      }
    }
  }
})
