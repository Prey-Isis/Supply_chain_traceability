import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 解决跨域问题
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
})
