<template>
  <div class="app-container equip-page">
    <el-tabs v-if="showTabs" v-model="tabLabel" type="border-card">
      <el-tab-pane label="数据表" name="数据表">
        <DataShee
          v-if="tabLabel === '数据表'"
          :form-data="form"
          :data-change-key="dataChangeKey"
          @data-change="dataChangeKey++"
        />
      </el-tab-pane>
      <el-tab-pane label="告警设置" name="告警设置">
        <TriggerTool
          v-if="tabLabel === '告警设置'"
          :form-data="form"
          alarm-type-name="告警"
          :data-change-key="dataChangeKey"
          :real-time-data="realTimeData"
          @data-change="dataChangeKey++"
        />
      </el-tab-pane>
      <el-tab-pane label="预警设置" name="预警设置">
        <TriggerTool
          v-if="tabLabel === '预警设置'"
          :form-data="form"
          alarm-type-name="预警"
          :data-change-key="dataChangeKey"
          :real-time-data="realTimeData"
          @data-change="dataChangeKey++"
        />
      </el-tab-pane>
      <el-tab-pane label="通用触发器设置" name="通用触发器设置">
        <GeneralTriggerTool
          v-if="tabLabel === '通用触发器设置'"
          :form-data="form"
          :data-change-key="dataChangeKey"
          @data-change="dataChangeKey++"
        />
      </el-tab-pane>
    </el-tabs>
    <el-empty v-else-if="loadError" :description="loadError" />
  </div>
</template>

<script>
import { Message } from '@/utils/element-shim'
import DataShee from './components/DataShee.vue'
import TriggerTool from './components/TriggerTool.vue'
import GeneralTriggerTool from './components/GeneralTriggerTool.vue'
import { findEquipmentListExactMatch } from '@/api/equipMana.js'

/** 恢复原独立设备管理页，四个标签沿用原结构和样式。 */
export default {
  name: 'EquipManaPage',
  components: { DataShee, TriggerTool, GeneralTriggerTool },
  data() {
    return {
      showTabs: false,
      loadError: '',
      tabLabel: '数据表',
      dataChangeKey: 0,
      realTimeData: {},
      formData: {},
      form: {},
      pageDisposed: false
    }
  },
  mounted() {
    try {
      const row = JSON.parse(this.$route.query.row || '{}')
      const id = Number(row?.id)
      if (!row || Array.isArray(row) || !Number.isSafeInteger(id) || id <= 0) {
        this.loadError = '缺少设备信息，请从设备列表重新进入'
        return
      }
      this.formData = { ...row, id }
    } catch {
      this.loadError = '设备参数无效，请从设备列表重新进入'
      return
    }
    this.getEquipMsg()
  },
  beforeUnmount() {
    this.pageDisposed = true
  },
  methods: {
    async getEquipMsg() {
      this.loadError = ''
      try {
        // 地址参数仅确定设备 ID；重新查询当前设备，避免使用列表中的过期配置。
        const res = await findEquipmentListExactMatch({ id: this.formData.id })
        if (this.pageDisposed) return
        const equipment = res.return?.data?.find(item => Number(item.id) === this.formData.id)
        if (!equipment) {
          this.loadError = '未查询到设备信息'
          return
        }
        this.form = { ...equipment, app: this.formData.app, apiKey: equipment.apiKey || '' }
        this.showTabs = true
        this.tabLabel = '数据表'
      } catch (error) {
        if (this.pageDisposed) return
        this.loadError = error?.message || '加载设备信息失败'
        Message.error(this.loadError)
      }
    }
  }
}
</script>
