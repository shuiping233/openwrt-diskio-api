import { defineConfig } from 'vite'
import vueDevTools from 'vite-plugin-vue-devtools'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), vueDevTools(),tailwindcss()],
  server:{
    proxy: {
    // 匹配所有以 /metric 开头的请求
    '/metric': {
      target: 'http://127.0.0.1:8080', // 转发给 Go 后端的地址
      changeOrigin: true,
      rewrite: (path) => path // 保持路径不变
    }
  }
  }
})
