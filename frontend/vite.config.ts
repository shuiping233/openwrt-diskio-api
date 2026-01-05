import { defineConfig } from 'vite'
import vueDevTools from 'vite-plugin-vue-devtools'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(),
  VitePWA({
    registerType: 'autoUpdate', // 自动检测更新
    includeAssets: ['favicon.ico', 'apple-touch-icon.png', 'mask-icon.svg'],
    devOptions: {
      enabled: true
    },
    manifest: {
      name: 'System Monitor',
      short_name: 'Monitor',
      description: '系统 I/O 监控仪表盘',
      theme_color: '#0f172a',
      background_color: '#0f172a',
      display: 'standalone',
      icons: [
        {
          src: 'favicon-192x192.png',
          // src: 'https://vitejs.dev/pwa-192x192.png', // 临时测试用
          sizes: '192x192',
          type: 'image/png'
        },
        {
          src: 'favicon-512x512.png',
          // src: 'https://vitejs.dev/pwa-512x512.png', // 临时测试用
          sizes: '512x512',
          type: 'image/png'
        }
      ]
    }
    // manifest : false // 使用现有的 manifest.webmanifest 文件
  }),
  // VitePWA({
  //   registerType: 'autoUpdate',
  //   devOptions: {
  //     enabled: true
  //   }
  // }),
  vueDevTools(),
  tailwindcss()],
  server: {
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
