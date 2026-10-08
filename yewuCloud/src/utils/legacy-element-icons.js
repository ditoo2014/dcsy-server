/**
 * Element UI 字符串图标名 → Element Plus @element-plus/icons-vue 组件
 */
import {
  ArrowDown,
  ArrowRight,
  Back,
  ChatDotRound,
  CirclePlus,
  Close,
  Delete,
  Document,
  DocumentCopy,
  Download,
  Edit,
  EditPen,
  Lock,
  More,
  Plus,
  QuestionFilled,
  Refresh,
  RefreshRight,
  Remove,
  Search,
  Select,
  User,
  VideoCamera
} from '@element-plus/icons-vue'

export const LEGACY_ICON_MAP = {
  'el-icon-user': User,
  'el-icon-lock': Lock,
  'el-icon-download': Download,
  'el-icon-plus': Plus,
  'el-icon-search': Search,
  'el-icon-circle-plus': CirclePlus,
  'el-icon-circle-plus-outline': CirclePlus,
  'el-icon-back': Back,
  'el-icon-close': Close,
  'el-icon-delete': Delete,
  'el-icon-edit': Edit,
  'el-icon-edit-outline': EditPen,
  'el-icon-refresh': Refresh,
  'el-icon-refresh-right': RefreshRight,
  'el-icon-document': Document,
  'el-icon-document-copy': DocumentCopy,
  'el-icon-s-claim': Select,
  'el-icon-s-comment': ChatDotRound,
  'el-icon-more': More,
  'el-icon-arrow-right': ArrowRight,
  'el-icon-arrow-down': ArrowDown,
  'el-icon-video-camera-solid': VideoCamera,
  'el-icon-question': QuestionFilled,
  'el-icon-remove-outline': Remove
}

/** 解析 Element UI 图标名；已是组件则原样返回 */
export function resolveLegacyIcon(name) {
  if (!name) return undefined
  if (typeof name === 'object' || typeof name === 'function') return name
  return LEGACY_ICON_MAP[name] || undefined
}
