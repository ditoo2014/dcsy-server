<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import PointHistoryChart from './PointHistoryChart.vue'

// 保留原采集云的运行时参数默认值，等待首次查询完成后再展示列表。
const props = defineProps({
  rows: { type: Array, required: true },
  loading: { type: Boolean, default: false },
  contentReady: { type: Boolean, default: true },
  lastUpdated: { type: String, default: '' },
  gatewayId: { type: String, default: '' },
  token: { type: String, default: '' },
  refreshEnabled: { type: Boolean, default: true }
})
const emit = defineEmits(['toggle-refresh'])

const keyword = ref('')
const appliedKeyword = ref('')
const page = ref(1)
const pageSize = ref(20)
const expandedRowKeys = ref([])
const isDarkTheme = ref(false)
let themeObserver = null
let themeMediaQuery = null

function detectDarkTheme() {
  if (typeof window === 'undefined') return false
  const html = document.documentElement
  const body = document.body
  const byAttr = html.getAttribute('data-theme') === 'dark' || body?.getAttribute?.('data-theme') === 'dark'
  const byClass =
    html.classList.contains('dark') ||
    body?.classList?.contains('dark') ||
    html.classList.contains('theme-dark') ||
    body?.classList?.contains('theme-dark') ||
    html.classList.contains('dark-theme') ||
    body?.classList?.contains('dark-theme')
  const bySystem = window.matchMedia?.('(prefers-color-scheme: dark)')?.matches ?? false
  return byAttr || byClass || bySystem
}

function syncThemeMode() {
  isDarkTheme.value = detectDarkTheme()
}

const filteredRows = computed(() => {
  const kw = appliedKeyword.value.trim().toLowerCase()
  return props.rows.filter(row => {
    if (!kw) return true
    return row.pointName.toLowerCase().includes(kw)
  })
})

const sortedRows = computed(() => {
  return [...filteredRows.value].sort((a, b) => {
    const ai = Number.isFinite(Number(a.orderIndex)) ? Number(a.orderIndex) : Number.MAX_SAFE_INTEGER
    const bi = Number.isFinite(Number(b.orderIndex)) ? Number(b.orderIndex) : Number.MAX_SAFE_INTEGER
    if (ai !== bi) return ai - bi
    return a.pointName.localeCompare(b.pointName, 'zh-CN')
  })
})

const pagedRows = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return sortedRows.value.slice(start, start + pageSize.value)
})

const total = computed(() => sortedRows.value.length)

function resetFilters() {
  keyword.value = ''
  appliedKeyword.value = ''
  page.value = 1
}

function queryFilters() {
  appliedKeyword.value = keyword.value
  page.value = 1
}

function onPageSizeChange(size) {
  pageSize.value = size
  page.value = 1
}

function onExpandChange(_row, expandedRows) {
  expandedRowKeys.value = expandedRows.map(r => r.pointKey)
}

function isAbnormalValue(status) {
  const s = String(status ?? '').trim()
  return s === '-1' || s === '异常'
}

function toggleRefresh() {
  emit('toggle-refresh')
}

onMounted(() => {
  syncThemeMode()
  themeObserver = new MutationObserver(() => syncThemeMode())
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class', 'data-theme']
  })
  if (document.body) {
    themeObserver.observe(document.body, {
      attributes: true,
      attributeFilter: ['class', 'data-theme']
    })
  }
  themeMediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  themeMediaQuery.addEventListener('change', syncThemeMode)
})

onUnmounted(() => {
  themeObserver?.disconnect()
  themeObserver = null
  if (themeMediaQuery) {
    themeMediaQuery.removeEventListener('change', syncThemeMode)
    themeMediaQuery = null
  }
})
</script>

<template>
  <el-card
    v-loading="loading || !contentReady"
    class="point-list-card"
    :class="{ 'is-dark': isDarkTheme }"
    shadow="never"
  >
    <template #header>
      <div class="head">
        <div class="title-wrap">
          <span class="title">网关数据列表</span>
          <span class="meta">数据点总数：{{ rows.length }}</span>
          <span v-if="lastUpdated" class="meta">更新时间：{{ lastUpdated }}</span>
        </div>
      </div>
    </template>

    <div class="list-body" :class="{ 'list-body--pending': !contentReady }">
      <template v-if="contentReady">
        <div class="filters">
          <el-input
            v-model="keyword"
            clearable
            placeholder="搜索数据点名"
            class="filter-item filter-keyword"
          />
          <el-button type="primary" @click="queryFilters">查询</el-button>
          <el-button plain @click="resetFilters">重置</el-button>
          <el-button :type="refreshEnabled ? 'warning' : 'success'" plain @click="toggleRefresh">
            {{ refreshEnabled ? '暂停刷新' : '开启刷新' }}
          </el-button>
        </div>

        <el-table
          :data="pagedRows"
          stripe
          :row-key="(row) => row.pointKey"
          :expand-row-keys="expandedRowKeys"
          @expand-change="onExpandChange"
        >
          <el-table-column type="expand" width="56">
            <template #default="{ row }">
              <PointHistoryChart
                :title="row.displayName || row.pointName"
                :gateway-id="row.realtimeGatewayId || gatewayId"
                :token="token"
                :data-id="row.dataId"
                :global-refresh-enabled="refreshEnabled"
                :dark-mode="isDarkTheme"
              />
            </template>
          </el-table-column>
          <el-table-column label="序号" width="90">
            <template #default="{ $index }">{{ (page - 1) * pageSize + $index + 1 }}</template>
          </el-table-column>
          <el-table-column prop="displayName" label="数据点名称" min-width="220">
            <template #default="{ row }">{{ row.displayName || row.pointName }}</template>
          </el-table-column>
          <el-table-column label="最新值" min-width="160">
            <template #default="{ row }">
              <span :class="{ 'value-abnormal': isAbnormalValue(row.status) }">{{ row.displayValue }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="valueTime" label="更新时间" min-width="190" />
        </el-table>

        <div class="pager">
          <el-pagination
            v-model:current-page="page"
            :page-size="pageSize"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            :total="total"
            background
            @size-change="onPageSizeChange"
          />
        </div>
      </template>
    </div>
  </el-card>
</template>

<style scoped>

.point-list-card {
  border-radius: 8px;
  border: 1px solid #f0f0f0;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.03), 0 2px 8px rgba(0, 0, 0, 0.06);
}

.point-list-card :deep(.el-card__header) {
  background: linear-gradient(180deg, #fbfdff 0%, #f7faff 100%);
  border-bottom: 1px solid #e7edf6;
}

.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.title-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.title {
  font-size: 15px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.88);
}

.meta {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.55);
}

.list-body--pending {
  min-height: 280px;
}

.filters {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  padding: 4px 2px 2px;
  flex-wrap: wrap;
}

.filter-item {
  width: 180px;
  max-width: 100%;
}

.filter-keyword {
  width: 260px;
  max-width: 100%;
}

.pager {
  margin-top: 12px;
  display: flex;
  min-width: 0;
  justify-content: flex-start;
}

.point-list-card :deep(.el-table) {
  --el-table-header-bg-color: #f8fbff;
  --el-table-row-hover-bg-color: #f5faff;
  --el-table-border-color: #ebf1f7;
  border-radius: 10px;
  overflow: hidden;
}

.point-list-card :deep(.el-table th.el-table__cell) {
  color: #5b6678;
  font-weight: 600;
  height: 44px;
}

.point-list-card :deep(.el-table td.el-table__cell) {
  height: 46px;
}

.point-list-card :deep(.el-table th.el-table__cell > .cell),
.point-list-card :deep(.el-table td.el-table__cell > .cell) {
  justify-content: center;
  text-align: center;
}

.point-list-card :deep(.el-table__expanded-cell) {
  background: #fbfdff;
  padding: 14px 18px !important;
}

.point-list-card.is-dark :deep(.el-table__expanded-cell) {
  background: linear-gradient(180deg, rgba(15, 29, 48, 0.96) 0%, rgba(12, 24, 40, 0.96) 100%) !important;
  border-top: 1px solid #2d4663 !important;
}

.point-list-card.is-dark {
  background: linear-gradient(180deg, #13253d 0%, #0f1d31 100%);
  border-color: #294362;
}

.point-list-card.is-dark :deep(.el-card__header) {
  background: rgba(15, 29, 48, 0.95);
  border-bottom-color: #294362;
}

.point-list-card.is-dark .title,
.point-list-card.is-dark .meta {
  color: #dbe9ff;
}

.point-list-card.is-dark :deep(.el-table) {
  --el-table-bg-color: #0f1d31;
  --el-table-tr-bg-color: #0f1d31;
  --el-table-header-bg-color: #13253d;
  --el-table-row-hover-bg-color: #17304f;
  --el-table-current-row-bg-color: #17304f;
  --el-table-border-color: #2d4663;
  --el-table-text-color: #d7e3f5;
  --el-table-header-text-color: #eef5ff;
}

.point-list-card.is-dark :deep(.el-table th.el-table__cell) {
  color: var(--el-table-header-text-color);
  background: var(--el-table-header-bg-color) !important;
}

.point-list-card.is-dark :deep(.el-table td.el-table__cell) {
  color: var(--el-table-text-color);
  background: var(--el-table-tr-bg-color) !important;
}

.point-list-card.is-dark :deep(.el-table--striped .el-table__body tr.el-table__row--striped td.el-table__cell) {
  background: #122742 !important;
}

.point-list-card.is-dark :deep(.el-table__expand-icon) {
  color: #9fb2ca;
}

.point-list-card.is-dark :deep(.el-input__wrapper),
.point-list-card.is-dark :deep(.el-select__wrapper) {
  background: rgba(15, 29, 48, 0.95) !important;
  box-shadow: 0 0 0 1px rgba(71, 95, 124, 0.85) inset !important;
}

.point-list-card.is-dark :deep(.el-input__inner),
.point-list-card.is-dark :deep(.el-select__selected-item),
.point-list-card.is-dark :deep(.el-select__placeholder) {
  color: #dbe9ff !important;
}

.point-list-card.is-dark .pager {
  border-top: 1px solid #294362;
  padding-top: 10px;
}

.point-list-card.is-dark .pager :deep(.btn-prev),
.point-list-card.is-dark .pager :deep(.btn-next),
.point-list-card.is-dark .pager :deep(.el-pager li) {
  background: rgba(15, 29, 48, 0.95) !important;
  border: 1px solid rgba(71, 95, 124, 0.72) !important;
  color: #cfe1f8 !important;
}

.point-list-card.is-dark .pager :deep(.btn-prev:hover),
.point-list-card.is-dark .pager :deep(.btn-next:hover),
.point-list-card.is-dark .pager :deep(.el-pager li:hover) {
  color: #ffffff !important;
  border-color: rgba(96, 165, 250, 0.75) !important;
}

.point-list-card.is-dark .pager :deep(.el-pager li.is-active) {
  background: linear-gradient(180deg, #225b8e 0%, #1a4f80 100%) !important;
  border-color: #5fa9e6 !important;
  color: #ffffff !important;
}

.point-list-card.is-dark .pager :deep(.btn-prev:disabled),
.point-list-card.is-dark .pager :deep(.btn-next:disabled),
.point-list-card.is-dark .pager :deep(button:disabled) {
  background: rgba(17, 33, 53, 0.78) !important;
  border-color: rgba(71, 95, 124, 0.55) !important;
  color: rgba(167, 191, 220, 0.45) !important;
  cursor: not-allowed !important;
  opacity: 1 !important;
}

.point-list-card :deep(.el-table__expand-icon) {
  color: #8ca3c3;
}

.point-list-card :deep(.el-input__wrapper) {
  border-radius: 8px;
}

.point-list-card :deep(.el-button) {
  border-radius: 8px;
}

.value-abnormal {
  color: #ff4d4f;
  font-weight: 600;
}
</style>
