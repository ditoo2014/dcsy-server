<template>
  <div class="app-container system-log-page">
    <div class="log-filter">
      <el-input
        v-model="searchForm.username"
        class="filter-item"
        clearable
        placeholder="账号"
        @keyup.enter="handleSearch"
      />
      <el-input
        v-model="searchForm.trueName"
        class="filter-item"
        clearable
        placeholder="姓名"
        @keyup.enter="handleSearch"
      />
      <div class="filter-date-wrap">
        <el-date-picker
          v-model="searchForm.dateRange"
          class="filter-date"
          type="datetimerange"
          start-placeholder="开始时间"
          end-placeholder="结束时间"
          range-separator="至"
        />
      </div>
      <el-button type="primary" class="filter-button" @click="handleSearch">查询</el-button>
      <el-button class="filter-button" @click="resetSearch">重置</el-button>
    </div>

    <el-table
      :key="tableKey"
      v-loading="loading"
      height="620"
      :data="tableData"
      border
      fit
      highlight-current-row
      class="log-table"
      :header-cell-style="tableHeaderStyle"
      :row-class-name="tableRowClassName"
    >
      <el-table-column label="序号" align="center" width="70">
        <template #default="scope">
          <span>{{ (page - 1) * limit + scope.$index + 1 }}</span>
        </template>
      </el-table-column>
      <el-table-column label="账号" prop="username" align="center" min-width="120" show-overflow-tooltip>
        <template #default="scope">
          <span>{{ scope.row.username || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="姓名" prop="trueName" align="center" min-width="120" show-overflow-tooltip>
        <template #default="scope">
          <span>{{ scope.row.trueName || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作内容" prop="logContent" align="center" min-width="180" show-overflow-tooltip>
        <template #default="scope">
          <span>{{ scope.row.logContent || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作时间" prop="dateTime" align="center" min-width="170">
        <template #default="scope">
          <span>{{ scope.row.dateTime }}</span>
        </template>
      </el-table-column>
      <el-table-column label="登录人IP" prop="ipAddress" align="center" min-width="140" show-overflow-tooltip>
        <template #default="scope">
          <span>{{ scope.row.ipAddress || '-' }}</span>
        </template>
      </el-table-column>
    </el-table>

    <Pagination
      v-show="total>0"
      v-model:page="page"
      v-model:limit="limit"
      :total="total"
      @pagination="getMsgList"
    />
  </div>
</template>

<script>
import Pagination from '@/components/Pagination' // 分页组件
import { formatDateString } from '@/utils/utils.js'
import { syslog } from '@/api/user.js'

export default {
  components: {
    Pagination
  },
  data() {
    return {
      tableKey: 1, // 参数值变化，表格对应改变
      total: 0, // 数据总条数
      limit: 10, // 每次加载数据条数
      page: 1, // 页码
      loading: false,
      searchForm: {
        username: '',
        trueName: '',
        dateRange: []
      },
      tableData: []
    }
  },
  mounted() {
    this.getSysLogList()
  },
  methods: {
    async getSysLogList() {
      this.loading = true
      try {
        const res = await syslog({
          page: this.page,
          limit: this.limit,
          username: this.searchForm.username,
          trueName: this.searchForm.trueName,
          dateRange: this.searchForm.dateRange
        })
        const result = res.return || {}
        const data = result.data || []
        this.total = Number(result.total || 0)
        this.tableData = data.map(item => this.normalizeLog(item))
      } finally {
        this.loading = false
      }
    },
    handleSearch() {
      this.page = 1
      this.getSysLogList()
    },
    resetSearch() {
      this.searchForm = {
        username: '',
        trueName: '',
        dateRange: []
      }
      this.page = 1
      this.getSysLogList()
    },
    getMsgList(e) {
      // 页码改变时触发
      this.page = e.page
      this.limit = e.limit
      this.getSysLogList()
    },
    normalizeLog(item) {
      return {
        ...item,
        trueName: item.trueName || item.truename || '',
        logContent: item.logContent || item.content || '',
        ipAddress: item.ipAddress || item.address || '',
        dateTime: this.formatLogTime(item.logTime || item.time)
      }
    },
    formatLogTime(value) {
      if (!value) return '-'
      const text = formatDateString(value, 'timeStmapReset')
      return text || '-'
    },
    tableHeaderStyle() {
      return {
        background: '#f5f7fa',
        color: '#303133',
        fontWeight: 600
      }
    },
    tableRowClassName({ rowIndex }) {
      return rowIndex % 2 === 0 ? 'log-row-even' : ''
    }
  }
}
</script>

<style scoped>
.system-log-page {
  padding-top: 16px;
}

.log-filter {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  padding: 14px 16px;
  margin-bottom: 14px;
  background: #fff;
  border: 1px solid #ebeef5;
}

.filter-item {
  flex: 0 0 150px;
  width: 150px;
}

.filter-date-wrap {
  flex: 0 0 380px;
  width: 380px;
}

.filter-date {
  width: 100% !important;
}

:deep(.filter-date) {
  width: 100% !important;
  max-width: 100%;
  flex: 0 0 auto;
}

.filter-button {
  flex: 0 0 auto;
  min-width: 72px;
}

.log-table {
  width: 100%;
  border-radius: 4px;
}

:deep(.log-row-even) {
  background: #fafcff;
}

:deep(.el-table__cell) {
  padding: 9px 0;
}

@media (max-width: 1200px) {
  .filter-item,
  .filter-date-wrap {
    flex-basis: 100%;
    width: 100%;
  }
}
</style>
