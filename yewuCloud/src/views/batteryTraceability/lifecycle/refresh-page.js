import { Message } from '@/utils/element-shim'

// 三个生命周期页面共用进入刷新规则；首次 mounted/activated 只发起一次查询。
export default {
  data() {
    return { pageActive: false, entryVersion: 0, queryVersion: 0, pageInitialization: null }
  },
  mounted() {
    this.enterPage()
  },
  activated() {
    this.enterPage()
  },
  deactivated() {
    this.leavePage()
  },
  beforeUnmount() {
    this.leavePage()
  },
  methods: {
    async enterPage() {
      if (this.pageActive) return
      this.pageActive = true
      const entry = ++this.entryVersion
      const query = this.queryVersion
      this.showLoading = true
      try {
        // 初始化只读取部门等基础资料，重新进入时保留用户的筛选和部门选择。
        if (!this.pageInitialization) {
          this.pageInitialization = Promise.resolve().then(() => this.init()).catch(error => {
            this.pageInitialization = null
            throw error
          })
        }
        await this.pageInitialization
        if (!this.pageActive || entry !== this.entryVersion) return
        if (this.getAllSiteList) await this.getAllSiteList()
        if (!this.pageActive || entry !== this.entryVersion) return
        // 初始化期间若用户已主动查询，不再追加一次进入查询。
        if (query !== this.queryVersion) return
        await this.search()
      } catch {
        if (this.pageActive && entry === this.entryVersion) {
          Message({ message: '页面数据加载失败，请稍后重试', type: 'error' })
          if (query === this.queryVersion) this.showLoading = false
        }
      }
    },
    leavePage() {
      this.pageActive = false
      this.entryVersion++
      // 请求可以正常结束，但离开页面后不能再覆盖列表。
      this.queryVersion++
      this.showLoading = false
    }
  }
}
