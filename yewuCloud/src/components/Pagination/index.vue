<template>
  <div :class="{'hidden':hidden}" class="pagination-container" :style="{padding}">
    <!--
      small: 是否使用小型分页样式
      current-page:当前页数,page-size:每页显示条目个数,layout:组件布局，子组件名用逗号分隔
      page-sizes:每页显示个数选择器的选项设置,total:总条目数,v-bind="$attrs":用于组件间通讯
      size-change: pageSize 改变时会触发,current-change: currentPage 改变时会触发
    -->
    <el-pagination
      v-model:current-page="currentPage"
      v-model:page-size="pageSize"
      :small="small"
      :background="background"
      :layout="layout"
      :page-sizes="pageSizes"
      :total="total"
      v-bind="$attrs"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
    />
  </div>
</template>

<script>
import { scrollTo } from '@/utils/scrollTo'

export default {
  name: 'Pagination',
  props: {
    padding: {
      type: String,
      default: '32px 16px'
    },
    small: {
      type: Boolean,
      default: false
    },
    total: {
      required: true,
      type: Number
    },
    page: {
      type: Number,
      default: 1
    },
    limit: {
      type: Number,
      default: 10
    },
    pageSizes: {
      type: Array,
      default() {
        return [10, 20, 30, 50, 100]
      }
    },
    layout: {
      type: String,
      default: 'total, sizes, prev, pager, next, jumper'
    },
    background: {
      type: Boolean,
      default: true
    },
    autoScroll: {
      type: Boolean,
      default: true
    },
    hidden: {
      type: Boolean,
      default: false
    }
  },
  computed: {
    currentPage: {
      get() {
        return this.page
      },
      set(val) {
        this.$emit('update:page', val)
      }
    },
    pageSize: {
      get() {
        return this.limit
      },
      set(val) {
        this.$emit('update:limit', val)
      }
    }
  },
  methods: {
    handleSizeChange(val) {
      this.$emit('pagination', { page: this.currentPage, limit: val })
      if (this.autoScroll) {
        scrollTo(0, 800)
      }
    },
    handleCurrentChange(val) {
      this.$emit('pagination', { page: val, limit: this.pageSize })
      if (this.autoScroll) {
        scrollTo(0, 800)
      }
    }
  }
}
</script>

<style scoped>
.pagination-container {
  background: #fff;
  margin-top: 0;
  /* 分页栏属于可收缩内容区，不让固定宽度的按钮撑大整个页面。 */
  min-width: 0;
  max-width: 100%;
  overflow-x: auto;
}
.pagination-container.hidden {
  display: none;
}
</style>
