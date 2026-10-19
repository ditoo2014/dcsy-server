import { defineStore } from 'pinia'
import { ref } from 'vue'

const DEFAULT_PAGE_TITLE = '页面'
const GATEWAY_MANAGE_TITLE = '网关管理'
const GATEWAY_DETAIL_TITLE = '网关详情'
const STREAM_MANAGE_TITLE = '数据池管理'
const STREAM_DETAIL_TITLE = '数据池详情'

export const useTagsViewStore = defineStore('collectionTagsView', () => {
  const visitedViews = ref([])

  function resolveTabKey(route) {
    if (route.name === 'collectionHome') {
      return route.query.module === 'stream' ? 'home:stream' : 'home:gateway'
    }
    if (route.name === 'gatewayDetail') {
      return `detail:${String(route.params.modelSlug || '')}:${String(route.params.gatewayId || '')}`
    }
    if (route.name === 'gatewayDetailLegacy') {
      return `detail-legacy:${String(route.params.gatewayId || '')}`
    }
    if (route.name === 'streamDetail') {
      return `stream-detail:${String(route.params.streamCode || '')}`
    }
    return `path:${route.fullPath}`
  }

  function resolveTitle(route) {
    if (route.name === 'collectionHome') {
      return route.query.module === 'stream' ? STREAM_MANAGE_TITLE : GATEWAY_MANAGE_TITLE
    }
    if (route.name === 'gatewayDetail' || route.name === 'gatewayDetailLegacy') {
      const gatewayId = String(route.params.gatewayId || '')
      return gatewayId ? `${GATEWAY_DETAIL_TITLE} - ${gatewayId}` : GATEWAY_DETAIL_TITLE
    }
    if (route.name === 'streamDetail') {
      const streamCode = String(route.params.streamCode || '')
      const streamName = String(route.query.name || '').trim()
      return `${STREAM_DETAIL_TITLE}${streamName || streamCode ? ` - ${streamName || streamCode}` : ''}`
    }
    return typeof route.meta?.title === 'string' ? route.meta.title : DEFAULT_PAGE_TITLE
  }

  function addVisitedView(route) {
    if (!route.matched.some(item => item.meta?.collectionCloud)) return
    // 网关详情保持原采集云行为，不加入顶部标签栏。
    if (route.name === 'gatewayDetail' || route.name === 'gatewayDetailLegacy') return

    const tabKey = resolveTabKey(route)
    const current = visitedViews.value.find(item => item.tabKey === tabKey)
    if (current) {
      current.title = resolveTitle(route)
      current.fullPath = route.fullPath
      current.path = route.path
      return
    }
    visitedViews.value.push({
      tabKey,
      fullPath: route.fullPath,
      path: route.path,
      title: resolveTitle(route)
    })
  }

  function delView(tabKey) {
    const index = visitedViews.value.findIndex(item => item.tabKey === tabKey)
    if (index >= 0) visitedViews.value.splice(index, 1)
  }

  function clearVisitedViews() {
    visitedViews.value = []
  }

  function isTabActive(route, item) {
    return resolveTabKey(route) === item.tabKey
  }

  return {
    visitedViews,
    addVisitedView,
    delView,
    clearVisitedViews,
    resolveTitle,
    isTabActive
  }
})
