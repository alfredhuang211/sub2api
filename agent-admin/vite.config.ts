import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'node:path'

export default defineConfig(({ command, mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const backendUrl = env.VITE_DEV_PROXY_TARGET || 'http://localhost:3101'
  const devPort = Number(env.VITE_DEV_PORT || 3100)
  const localDemoModule =
    command === 'serve'
      ? resolve(__dirname, 'src/api/localDemo.dev.ts')
      : resolve(__dirname, 'src/api/localDemo.disabled.ts')

  return {
    plugins: [vue()],
    resolve: {
      alias: [
        { find: '@/api/localDemo', replacement: localDemoModule },
        { find: '@', replacement: resolve(__dirname, 'src') }
      ]
    },
    build: {
      outDir: 'dist',
      emptyOutDir: true
    },
    server: {
      host: '0.0.0.0',
      port: devPort,
      proxy: {
        '/api': {
          target: backendUrl,
          changeOrigin: true
        }
      }
    }
  }
})
