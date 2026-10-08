import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { build } from 'vite'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const projectRoot = path.resolve(scriptDir, '..')
const sourceRoot = path.join(projectRoot, 'src')
const normalize = value => path.normalize(value.split('?')[0])
const usedFiles = new Set()

// 通过一次不落盘的 DCSY 生产构建收集真实模块，避免只靠文本搜索误删动态依赖。
await build({
  root: projectRoot,
  configFile: path.join(projectRoot, 'vite.config.mjs'),
  mode: 'dcsyPro',
  logLevel: 'warn',
  build: { write: false },
  plugins: [{
    name: 'audit-dcsy-source-modules',
    enforce: 'post',
    load(id) {
      const fileName = normalize(id)
      if (fileName.startsWith(sourceRoot + path.sep)) usedFiles.add(fileName)
      return null
    },
    transform(_code, id) {
      const fileName = normalize(id)
      if (fileName.startsWith(sourceRoot + path.sep)) usedFiles.add(fileName)
      return null
    }
  }]
})

// Stylus/CSS 预处理器会自行读取 @import 文件，这些文件不会再次进入 Vite 插件钩子。
// 递归补记相对样式依赖，防止把实际生效的主题样式误报为未使用。
const pendingStyles = [...usedFiles].filter(fileName => /\.(?:css|less|sass|scss|styl)$/.test(fileName))
const checkedStyles = new Set()
while (pendingStyles.length > 0) {
  const fileName = pendingStyles.pop()
  if (checkedStyles.has(fileName) || !fs.existsSync(fileName)) continue
  checkedStyles.add(fileName)
  const content = fs.readFileSync(fileName, 'utf8')
  for (const match of content.matchAll(/@import\s+['"]([^'"]+)['"]/g)) {
    if (!match[1].startsWith('.')) continue
    const imported = normalize(path.resolve(path.dirname(fileName), match[1]))
    const candidates = path.extname(imported)
      ? [imported]
      : ['.styl', '.scss', '.sass', '.less', '.css'].map(extension => imported + extension)
    const existing = candidates.find(candidate => fs.existsSync(candidate))
    if (existing && !usedFiles.has(existing)) {
      usedFiles.add(existing)
      pendingStyles.push(existing)
    }
  }
}

function collectCodeFiles(directory, result = []) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const absolute = path.join(directory, entry.name)
    if (entry.isDirectory()) {
      collectCodeFiles(absolute, result)
    } else if (/\.(?:css|js|less|mjs|sass|scss|styl|ts|vue)$/.test(entry.name)) {
      result.push(absolute)
    }
  }
  return result
}

const allFiles = collectCodeFiles(sourceRoot)
const unusedFiles = allFiles
  .filter(fileName => !usedFiles.has(normalize(fileName)))
  .map(fileName => path.relative(projectRoot, fileName).replaceAll('\\', '/'))
  .sort()

console.log(`DCSY 构建使用 ${usedFiles.size} 个源码模块，未进入构建的代码文件 ${unusedFiles.length} 个：`)
for (const fileName of unusedFiles) console.log(fileName)
