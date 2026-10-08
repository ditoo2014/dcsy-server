import { Fragment, h } from 'vue'
import SvgIcon from '@/components/SvgIcon'

export default {
  name: 'MenuItem',
  props: {
    icon: {
      type: String,
      default: ''
    },
    title: {
      type: String,
      default: ''
    }
  },
  setup(props) {
    // 使用 Vue 渲染函数，避免普通 JSX 被构建工具按 React 语法转换。
    return () => h(Fragment, null, [
      props.icon ? h(SvgIcon, { iconClass: props.icon }) : null,
      props.title
        ? h('span', { class: 'menu-item-title menu-item-title--inline' }, props.title)
        : null
    ])
  }
}
