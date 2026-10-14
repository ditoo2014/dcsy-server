import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { createSvgIconsPlugin } from 'vite-plugin-svg-icons'

const projectRoot = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, projectRoot, '')
  const protocol = env.VITE_API_PROTOCOL || 'https'
  const host = env.VITE_API_HOST || 'localhost'
  const port = env.VITE_API_PORT ? `:${env.VITE_API_PORT}` : ''
  const apiProxy = { target: `${protocol}://${host}${port}`, changeOrigin: true, secure: false }

  return {
    plugins: [
      vue(),
      createSvgIconsPlugin({
        iconDirs: [path.resolve(projectRoot, 'src/icons/svg'), path.resolve(projectRoot, 'src/icons/mySvg')],
        symbolId: 'icon-[name]'
      })
    ],
    resolve: {
      alias: { '@': path.resolve(projectRoot, 'src') },
      extensions: ['.mjs', '.js', '.json', '.vue']
    },
    server: {
      host: '0.0.0.0',
      port: Number(env.VITE_DEV_PORT || 19530),
      // 开发环境与生产反向代理保持相同的接口前缀。
      proxy: {
        '/api': { ...apiProxy },
        '/auth': { ...apiProxy },
        '/business': { ...apiProxy },
        '/go/images': { ...apiProxy },
        '/tolink': { ...apiProxy }
      }
    },
    build: {
      outDir: 'dist',
      assetsDir: 'static',
      sourcemap: false,
      // 当前项目包含地图、图表和采集云页面，允许较大的按页面资源包。
      chunkSizeWarningLimit: 2500,
      rollupOptions: {
        onwarn(warning, defaultHandler) {
          const id = String(warning.id || '').replace(/\\/g, '/')
          // 只隐藏 VueUse 依赖的 PURE 注释位置警告，其余警告仍按默认方式输出。
          if (warning.code === 'INVALID_ANNOTATION' &&
              id.includes('/node_modules/@vueuse/core/') &&
              warning.message?.includes('#__PURE__')) {
            return
          }
          defaultHandler(warning)
        }
      }
    }
  }
})
