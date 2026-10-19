<template>
  <div class="content">
    <div class="BannerCss">
      <Banner :list="bannerList" @click-button="clickBannerBtn" />
    </div>
    <div class="app-container">
      <el-scrollbar class="allheight">
        <el-table
          :key="tableKey"
          v-loading="tableLoading"
          :data="tableData"
          height="600"
          fit
          highlight-current-row
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="55" />
          <el-table-column :label="`${alarmTypeName}代码`" align="center">
            <template #default="scope"><span>{{ scope.row.code }}</span></template>
          </el-table-column>
          <el-table-column :label="`${alarmTypeName}数据`" align="center">
            <template #default="scope"><span>{{ scope.row.labelName }}</span></template>
          </el-table-column>
          <el-table-column label="触发类型" align="center">
            <template #default="scope">
              <span>{{ scope.row.triggerModeName }}</span>
              <span v-if="scope.row.triggerModeName === '周期触发'">，间隔：{{ scope.row.realtimeCycle }}{{ scope.row.cycleTypeName }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="`${alarmTypeName}条件`" align="center">
            <template #default="scope">
              <div v-for="(logicItem, index) in scope.row.logics" :key="index" style="height:30px;line-height: 30px;">
                <span>{{ logicItem.logic }} {{ logicItem.triggerValue }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" align="center" class-name="small-padding fixed-width">
            <template #default="scope">
              <el-button type="primary" size="small" :disabled="operationLoading || tableLoading" @click="modify(scope.row)">修改</el-button>
              <el-button type="danger" size="small" :disabled="operationLoading || tableLoading" @click="remove(scope.row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 显式引用现有抽屉封装，保留原版 400 像素宽度和底部操作区。 -->
        <ElDrawerWrap v-model="showDrawer" :title="textMap[dialogStatus]" :mask-closable="false" width="400">
          <el-form ref="dataForm" :rules="rules" :model="form" size="small" label-width="100px">
            <el-row>
              <el-col :span="24">
                <el-form-item :label="`${alarmTypeName}代码`" prop="code">
                  <div style="width:250px"><el-input v-model="form.code" /></div>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item :label="`${alarmTypeName}等级`">
                  <el-select v-model="form.alarmUser.alarmStage" class="filter-item" placeholder="请选择" filterable>
                    <el-option v-for="item in alarmStageList" :key="item.key" :label="item.name" :value="item.key" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="数据" prop="labelId">
                  <el-select v-model="form.labelId" class="filter-item" placeholder="请选择" filterable @change="labelChange">
                    <el-option v-for="item in labelList" :key="item.key" :label="item.name" :value="item.key" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="是否进行统计">
                  <el-radio-group v-model="form.statisticsEnabled" @change="handleStatisticsEnabledChange">
                    <el-radio :value="true">是</el-radio><el-radio :value="false">否</el-radio>
                  </el-radio-group>
                </el-form-item>
              </el-col>
              <el-col v-if="form.triggerType === 1 || form.triggerType === 2" :span="24">
                <el-form-item label="统计类型">
                  <el-select v-model="form.statisticConfig.statisticType" class="filter-item" placeholder="请选择" filterable>
                    <el-option v-for="item in displayStatisticTypeList" :key="item.key" :label="item.name" :value="item.key" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col v-if="form.triggerType === 1 || form.triggerType === 2" :span="24">
                <el-form-item label="使用统计值">
                  <el-radio-group v-model="form.statisticConfig.useStatisticValue">
                    <el-radio :value="true">是</el-radio><el-radio :value="false">否</el-radio>
                  </el-radio-group>
                </el-form-item>
              </el-col>
              <el-col v-if="(form.triggerType === 1 || form.triggerType === 2) && form.statisticConfig.useStatisticValue" :span="24">
                <el-form-item label="统计值" prop="statisticConfig.statisticValue"><el-input v-model="form.statisticConfig.statisticValue" /></el-form-item>
              </el-col>
              <el-col v-if="form.triggerType === 1 || form.triggerType === 2" :span="24">
                <el-form-item label="统计值类型">
                  <el-select v-model="form.statisticConfig.statisticValueType" class="filter-item" placeholder="请选择" filterable>
                    <el-option v-for="item in statisticValueTypeList" :key="item.key" :label="item.name" :value="item.key" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="触发方式">
                  <el-select v-model="form.triggerMode" class="filter-item" placeholder="请选择" filterable @change="handleTriggerModeChange">
                    <el-option :key="1" label="单次触发" :value="1" /><el-option :key="2" label="周期触发" :value="2" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col v-if="form.triggerMode === 2" :span="24">
                <el-form-item label="触发间隔" prop="realtimeCycle">
                  <div style="float:left;margin-right:10px;width:100px;"><el-input v-model="form.realtimeCycle" type="number" :min="1" /></div>
                  <div style="float: left;width: 100px;">
                    <el-select v-model="form.cycleType" class="filter-item" placeholder="请选择" filterable @change="cycleChange">
                      <el-option v-for="item in realtimeCycleList" :key="item.key" :label="item.name" :value="item.key" />
                    </el-select>
                  </div>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="逻辑" prop="logics">
                  <div v-for="(logic, index) in form.logics" :key="index" style="height:40px">
                    <div style="float:left;width: 100px;">
                      <el-select v-model="logic.logic" class="filter-item" placeholder="请选择" filterable>
                        <el-option v-for="item in judgeList" :key="item.key" :label="item.name" :value="item.key" />
                      </el-select>
                    </div>
                    <div style="float:left;margin-left:10px;width: 100px;"><el-input v-model="logic.triggerValue" type="number" /></div>
                    <div style="float:left;margin-left:10px;width:40px"><el-button style="float:right" type="danger" :icon="$legacyIcon('el-icon-close')" circle @click="removelogics(index)" /></div>
                  </div>
                  <el-button type="primary" :icon="$legacyIcon('el-icon-plus')" circle @click="addlogics" />
                </el-form-item>
              </el-col>
            </el-row>
          </el-form>
          <div class="fixedBottomCss">
            <el-button :disabled="operationLoading" @click="showDrawer = false">取消</el-button>
            <el-button type="primary" :loading="operationLoading" @click="setData">保存</el-button>
          </div>
        </ElDrawerWrap>
      </el-scrollbar>
    </div>

    <el-dialog v-model="copyOtherEquipAlarmDialog" :title="`复制其他设备${alarmTypeName}`" width="30vw">
      <el-form size="small" label-width="80px">
        <el-form-item :label="`复制${alarmTypeName}`">
          <el-select v-model="copyEquipId" class="filter-item" placeholder="请选择" filterable :disabled="operationLoading" @change="copyEquipAlarmChange">
            <el-option v-for="item in equipList" :key="item.key" :label="item.name" :value="item.key" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button :disabled="operationLoading" @click="copyOtherEquipAlarmDialog = false">取消</el-button>
          <el-button type="primary" :loading="operationLoading || copyLoading" @click="sureCopy">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { Message } from '@/utils/elementShim'
import Banner from '@/components/ccpcComp/Banner.vue'
import ElDrawerWrap from '@/components/elDrawer/index.vue'
import { meConfirm } from '@/utils/utils.js'
import { getUserId } from '@/utils/auth.js'
import { judgeList, alarmStageList } from '@/utils/ccpcData.js'
import { findDataDetails, findEquipmentListExactMatch } from '@/api/equipment.js'
import { gettrigger, createtrigger, deletetrigger, updatetrigger } from '@/api/trigger.js'

const isFiniteNumber = value => value !== null && value !== undefined && String(value).trim() !== '' && Number.isFinite(Number(value))
const sameId = (left, right) => String(left) === String(right)
const getList = response => {
  // 旧接口查询失败可能返回 code=200、return=null，不能当作成功的空列表。
  if (!response.return || !Object.prototype.hasOwnProperty.call(response.return, 'data')) {
    throw new Error('未能读取设备配置，请稍后重试')
  }
  const list = response.return.data
  if (list === null) return []
  if (!Array.isArray(list)) throw new Error('接口返回数据格式错误，请稍后重试')
  return list
}
const getSuffix = value => {
  const text = String(value || '')
  const index = text.indexOf('-')
  return index >= 0 ? text.slice(index + 1) : text
}

export default {
  components: { Banner, ElDrawerWrap },
  props: {
    formData: { type: Object, default: () => ({}) },
    dataChangeKey: { type: Number, default: 0 },
    alarmTypeName: { type: String, default: '告警' },
    realTimeData: { type: Object, default: () => ({}) }
  },
  emits: ['data-change'],
  data() {
    return {
      tableLoading: false,
      operationLoading: false,
      copyLoading: false,
      // 只允许已成功加载的当前设备进入编辑，旧请求不得回填切换后的页面。
      loadedEquipmentId: null,
      labelsReady: false,
      triggersReady: false,
      requestSequence: 0,
      labelSequence: 0,
      copySequence: 0,
      copyLabelsReady: false,
      editingEquipmentId: null,
      bannerList: [
        { style: 'infoNum', title: `${this.alarmTypeName}器数`, info: '0', float: 'left' },
        { style: 'infoNum', title: '设备名称', info: ' ', float: 'left' },
        { style: 'button', buttonName: `添加${this.alarmTypeName}`, type: 'primary', icon: 'el-icon-circle-plus', float: 'right' },
        { style: 'button', buttonName: '批量删除', type: 'danger', float: 'right' },
        { style: 'button', buttonName: `复制其他设备${this.alarmTypeName}`, float: 'right' }
      ],
      tableKey: 1,
      total: 0,
      showDrawer: false,
      textMap: { add: `添加${this.alarmTypeName}`, modify: `修改${this.alarmTypeName}` },
      dialogStatus: '',
      rules: {
        code: [{ required: true, whitespace: true, trigger: 'blur', message: `${this.alarmTypeName}代码不能为空` }],
        labelId: [{ required: true, trigger: 'change', message: '请选择数据' }],
        realtimeCycle: [{ trigger: 'blur', validator: (rule, value, callback) => {
          if (this.form.triggerMode !== 2 || (Number.isSafeInteger(Number(value)) && Number(value) > 0)) callback()
          else callback(new Error('请输入大于0的正整数'))
        } }],
        logics: [{ trigger: 'blur', validator: (rule, value, callback) => {
          if (!Array.isArray(value) || value.length === 0) return callback(new Error('请至少添加一条逻辑'))
          if (value.some(item => !judgeList.some(judge => judge.key === item.logic) || !isFiniteNumber(item.triggerValue))) {
            return callback(new Error('请选择逻辑并填写有效数字'))
          }
          callback()
        } }],
        'statisticConfig.statisticValue': [{ trigger: 'blur', validator: (rule, value, callback) => {
          if (!this.form.statisticsEnabled || !this.form.statisticConfig.useStatisticValue || isFiniteNumber(value)) callback()
          else callback(new Error('统计值请填写有效数字'))
        } }]
      },
      tableData: [],
      form: { alarmUser: {}, statisticConfig: {}, logics: [] },
      statisticValueTypeList: [{ key: 0, name: '累计值' }, { key: 1, name: '增量值' }],
      realtimeCycleList: [{ key: 1, name: '秒' }, { key: 2, name: '分钟' }, { key: 3, name: '小时' }],
      alarmStageList,
      labelList: [],
      judgeList,
      selectAlarmList: [],
      copyOtherEquipAlarmDialog: false,
      copyEquipId: '',
      copyEquipLabelList: [],
      equipList: []
    }
  },
  computed: {
    displayStatisticTypeList() {
      return [{ key: '0', name: '通用统计' }, { key: '1', name: `${this.alarmTypeName}条数统计` }]
    }
  },
  watch: {
    dataChangeKey() { this.refreshLabelList() },
    'formData.id'() {
      this.showDrawer = false
      this.copyOtherEquipAlarmDialog = false
      this.init()
    },
    'formData.equipmentName'(name) { this.bannerList[1].info = name || ' ' }
  },
  mounted() {
    this.bannerList[1].info = this.formData.equipmentName || ' '
    this.init()
  },
  beforeUnmount() {
    this.requestSequence++
    this.labelSequence++
    this.copySequence++
  },
  methods: {
    getAlarmTypeCode() { return this.alarmTypeName === '预警' ? '1' : '0' },
    getSiteId() { return Number(this.formData.belongToSiteId) || 0 },
    matchAlarmType(value, type) {
      // 按逗号分隔的完整类型匹配，防止类型 10 被误认为预警 1。
      return String(value == null ? '' : value).split(',').map(item => item.trim()).includes(type)
    },
    belongsToEquipment(row, equipmentId = this.formData.id) {
      return row && sameId(row.equipmentId, equipmentId) && this.matchAlarmType(row.alarmUser && row.alarmUser.alarmType, this.getAlarmTypeCode())
    },
    buildDefaultAlarmUser(alarmType) {
      const userId = Number(getUserId())
      return {
        alarmCode: '', alarmType, alarmStage: this.alarmStageList[0].key,
        // 新建规则默认由创建人接收短信，不再使用原版固定用户 1、2 或微信渠道。
        alarmChannel: '0', alarmUsers: Number.isSafeInteger(userId) && userId > 0 ? String(userId) : ''
      }
    },
    normalizeAlarmUser(alarmUser, alarmType, preserveType = true) {
      const source = alarmUser || {}
      const channels = String(source.alarmChannel == null ? '0' : source.alarmChannel).split(',').map(item => item.trim())
      return {
        ...this.buildDefaultAlarmUser(alarmType), ...source,
        alarmType: preserveType && this.matchAlarmType(source.alarmType, alarmType) ? String(source.alarmType) : alarmType,
        // 已关闭短信的规则保持关闭，已配置的收件人原样保留，剔除不再支持的渠道。
        alarmChannel: channels.includes('0') ? '0' : '',
        alarmUsers: source.alarmUsers == null ? '' : String(source.alarmUsers)
      }
    },
    defaultStatisticConfig(source = {}) {
      return {
        useStatisticValue: false, statisticCode: '', statisticValue: 1,
        ...source,
        statisticType: String(source.statisticType == null ? '1' : source.statisticType),
        statisticValueType: Number(source.statisticValueType == null ? 1 : source.statisticValueType)
      }
    },
    canOperate() {
      if (this.operationLoading || this.tableLoading) return false
      if (!this.labelsReady || !this.triggersReady || !sameId(this.loadedEquipmentId, this.formData.id)) {
        Message.error('设备配置尚未加载完成，请刷新后重试')
        return false
      }
      return true
    },
    async init() {
      this.loadedEquipmentId = null
      this.labelsReady = false
      this.triggersReady = false
      this.labelList = []
      this.tableData = []
      this.selectAlarmList = []
      this.bannerList[0].info = '0'
      const id = Number(this.formData.id)
      if (!Number.isSafeInteger(id) || id <= 0) return
      this.loadedEquipmentId = id
      await Promise.all([this.refreshLabelList(), this.getDataList()])
    },
    async refreshLabelList() {
      const id = this.formData.id
      const sequence = ++this.labelSequence
      this.labelsReady = false
      try {
        const labels = await this.getLabelList(id)
        if (sequence !== this.labelSequence || !sameId(id, this.formData.id)) return
        this.labelList = labels
        this.labelsReady = true
      } catch (error) {
        if (sequence !== this.labelSequence || !sameId(id, this.formData.id)) return
        this.labelList = []
        this.handlePersistError(error)
      }
    },
    async getLabelList(id = this.formData.id) {
      const equipmentId = Number(id)
      if (!Number.isSafeInteger(equipmentId) || equipmentId <= 0) throw new Error('设备参数无效，请从设备列表重新进入')
      const list = getList(await findDataDetails({ equipmentId }))
      const seen = new Set()
      return list.filter(item => {
        if (!item.labelId || seen.has(String(item.labelId))) return false
        if (item.equipmentId != null && !sameId(item.equipmentId, equipmentId)) return false
        seen.add(String(item.labelId))
        return true
      }).map(item => ({ key: item.labelId, name: item.name || item.labelName, labelName: item.labelName || item.name }))
    },
    async getDataList() {
      const id = this.formData.id
      const sequence = ++this.requestSequence
      this.tableLoading = true
      this.triggersReady = false
      try {
        const list = getList(await gettrigger({ equipmentId: Number(id) }))
        if (sequence !== this.requestSequence || !sameId(id, this.formData.id)) return
        this.tableData = list.filter(item => this.belongsToEquipment(item, id)).map(item => this.setGettriggerDeatilMsg(item))
        this.total = this.tableData.length
        this.bannerList[0].info = String(this.total)
        this.selectAlarmList = []
        this.tableKey++
        this.triggersReady = true
      } catch (error) {
        if (sequence !== this.requestSequence || !sameId(id, this.formData.id)) return
        this.tableData = []
        this.total = 0
        this.bannerList[0].info = '0'
        this.handlePersistError(error)
      } finally {
        if (sequence === this.requestSequence) this.tableLoading = false
      }
    },
    setGettriggerDeatilMsg(source) {
      const cycle = Number(source.cycle) || 0
      const cycleType = cycle && cycle % 3600 === 0 ? 3 : (cycle && cycle % 60 === 0 ? 2 : 1)
      const triggerType = Number(source.triggerType) || 0
      return {
        ...source, cycle, cycleType, triggerType,
        triggerMode: cycle ? 2 : 1, triggerModeName: cycle ? '周期触发' : '单次触发',
        realtimeCycle: cycle / (cycleType === 3 ? 3600 : (cycleType === 2 ? 60 : 1)),
        cycleTypeName: this.realtimeCycleList.find(item => item.key === cycleType).name,
        logics: (source.logics || []).map(item => ({ ...item })),
        // 原 API 使用大写 ReplaceCodeFlag；后端查询返回小写，兼容后再提交。
        strMatchs: (source.strMatchs || []).map(item => ({ ...item, ReplaceCodeFlag: item.ReplaceCodeFlag ?? item.replaceCodeFlag ?? false })),
        triggerTypeArr: triggerType === 1 ? ['统计'] : (triggerType === 2 ? [this.alarmTypeName, '统计'] : [this.alarmTypeName])
      }
    },
    clickBannerBtn(item) {
      if (!this.canOperate()) return
      if (item.buttonName === `添加${this.alarmTypeName}`) this.add()
      else if (item.buttonName === `复制其他设备${this.alarmTypeName}`) this.copyOtherEquipAlarm()
      else if (item.buttonName === '批量删除') this.batchRemove()
    },
    add() {
      if (!this.canOperate()) return
      this.dialogStatus = 'add'
      if (!this.resetForm()) return
      this.showDrawer = true
    },
    modify(row) {
      if (!this.canOperate() || !this.belongsToEquipment(row)) return
      this.dialogStatus = 'modify'
      this.resetForm(row)
      this.showDrawer = true
    },
    resetForm(row) {
      const alarmType = this.getAlarmTypeCode()
      if (!row && this.labelList.length === 0) {
        Message.error('暂无数据，请到数据表添加数据')
        return false
      }
      this.editingEquipmentId = this.formData.id
      this.form = row ? {
        ...this.setGettriggerDeatilMsg(row),
        apiKey: row.apiKey || this.formData.apiKey,
        siteId: row.siteId || this.getSiteId(), equipmentId: Number(this.formData.id),
        statisticsEnabled: Number(row.triggerType) === 1 || Number(row.triggerType) === 2,
        formula: row.formula || 'x', marktrigger: row.marktrigger || '',
        alarmUser: this.normalizeAlarmUser(row.alarmUser, alarmType),
        statisticConfig: this.defaultStatisticConfig(row.statisticConfig || {})
      } : {
        apiKey: this.formData.apiKey, siteId: this.getSiteId(), equipmentId: Number(this.formData.id),
        code: '', labelId: this.labelList[0].key, labelName: this.labelList[0].labelName,
        triggerMode: 1, cycleType: 1, realtimeCycle: 0, cycle: 0, marktrigger: '',
        triggerType: 2, triggerTypeArr: [this.alarmTypeName, '统计'], statisticsEnabled: true, formula: 'x',
        // 与原版一致，由用户明确添加触发条件，不能默认创建大于 0 的规则。
        logics: [], strMatchs: [],
        alarmUser: this.buildDefaultAlarmUser(alarmType), statisticConfig: this.defaultStatisticConfig()
      }
      this.$nextTick(() => this.$refs.dataForm && this.$refs.dataForm.clearValidate())
      return true
    },
    handleStatisticsEnabledChange() {
      this.form.triggerType = this.form.statisticsEnabled ? 2 : 0
      this.form.triggerTypeArr = this.form.statisticsEnabled ? [this.alarmTypeName, '统计'] : [this.alarmTypeName]
    },
    labelChange() {
      const label = this.labelList.find(item => sameId(item.key, this.form.labelId))
      this.form.labelName = label ? label.labelName : ''
    },
    handleTriggerModeChange() {
      this.form.realtimeCycle = this.form.triggerMode === 2 ? 1 : 0
      if (this.form.triggerMode === 1) this.form.cycleType = 1
      this.cycleChange()
    },
    cycleChange() {
      const factor = this.form.cycleType === 3 ? 3600 : (this.form.cycleType === 2 ? 60 : 1)
      this.form.cycle = this.form.triggerMode === 1 ? 0 : Number(this.form.realtimeCycle) * factor
    },
    addlogics() { this.form.logics.push({ logic: '>', triggerValue: 0 }) },
    removelogics(index) { this.form.logics.splice(index, 1) },
    handlePersistError(error) { Message.error((error && error.message) || '操作失败，请稍后重试') },
    async afterPersist(title) {
      Message.success(title)
      this.showDrawer = false
      this.$emit('data-change')
      await this.getDataList()
    },
    async setData() {
      if (!this.canOperate() || !sameId(this.editingEquipmentId, this.formData.id)) return
      const formRef = this.$refs.dataForm
      if (!formRef) return Message.error('表单未就绪，请稍后重试')
      const valid = await formRef.validate().catch(() => false)
      if (!valid) return
      if (!this.canOperate() || !sameId(this.editingEquipmentId, this.formData.id)) return
      if (!this.form.apiKey) return Message.error('设备 apiKey 缺失，请从设备列表重新进入管理页')
      const label = this.labelList.find(item => sameId(item.key, this.form.labelId))
      if (!label) return Message.error('所选数据已不存在，请重新选择数据')
      if (this.dialogStatus === 'add' && !this.form.alarmUser.alarmUsers) return Message.error('当前账号信息未加载，请重新登录后添加告警配置')
      if (this.dialogStatus === 'modify' && !this.tableData.some(row => sameId(row.id, this.form.id) && this.belongsToEquipment(row))) {
        return Message.error('设备配置已变化，请刷新后重新修改')
      }
      this.cycleChange()
      if (!Number.isSafeInteger(this.form.cycle) || this.form.cycle < 0) return Message.error('触发间隔过大或格式无效')
      this.operationLoading = true
      try {
        const payload = {
          ...this.form, code: this.form.code.trim(), labelName: label.labelName,
          equipmentId: Number(this.formData.id),
          alarmUser: { ...this.form.alarmUser }, statisticConfig: { ...this.form.statisticConfig },
          logics: this.form.logics.map(item => ({ ...item })), strMatchs: this.form.strMatchs.map(item => ({ ...item }))
        }
        if (this.dialogStatus === 'add') await createtrigger(payload)
        else await updatetrigger(payload)
        await this.afterPersist(this.dialogStatus === 'add' ? '添加成功' : '修改成功')
      } catch (error) { this.handlePersistError(error) } finally { this.operationLoading = false }
    },
    async remove(row) {
      if (!this.canOperate() || !this.belongsToEquipment(row)) return
      const id = this.formData.id
      const combined = this.matchAlarmType(row.alarmUser.alarmType, '0') && this.matchAlarmType(row.alarmUser.alarmType, '1')
      const sure = await meConfirm(this, {
        title: '确认删除',
        content: `此操作将永久删除${row.code}，${combined ? '该配置同时包含告警和预警，删除后均会移除，' : ''}是否继续？`
      })
      if (!sure || !sameId(id, this.formData.id) || !this.canOperate()) return
      this.operationLoading = true
      try {
        await deletetrigger({ id: row.id })
        await this.afterPersist(`${row.code}删除成功`)
      } catch (error) { this.handlePersistError(error) } finally { this.operationLoading = false }
    },
    async batchRemove() {
      if (!this.canOperate()) return
      const rows = this.selectAlarmList.filter(row => this.belongsToEquipment(row))
      if (!rows.length) return Message.warning('请先选择要删除的记录')
      const id = this.formData.id
      const sure = await meConfirm(this, { title: '确认删除', content: `此操作将批量删除 ${rows.length} 条记录，是否继续？` })
      if (!sure || !sameId(id, this.formData.id) || !this.canOperate()) return
      this.operationLoading = true
      let deleted = 0
      try {
        for (const row of rows) {
          if (!sameId(id, this.formData.id)) throw new Error('设备已切换，已停止删除')
          await deletetrigger({ id: row.id })
          deleted++
        }
        await this.afterPersist(`成功删除${deleted}条记录`)
      } catch (error) {
        this.handlePersistError(error)
        if (deleted) {
          Message.warning(`已删除${deleted}条记录，其余未删除，请刷新后核对`)
          this.$emit('data-change')
          await this.getDataList()
        }
      } finally { this.operationLoading = false }
    },
    handleSelectionChange(rows) { this.selectAlarmList = rows },
    async copyOtherEquipAlarm() {
      if (!this.canOperate()) return
      const id = this.formData.id
      try {
        // 明确请求完整设备列表，避免原版默认分页仅能复制第一页设备。
        const list = getList(await findEquipmentListExactMatch({ pageNum: 1, pageSize: 1000000 }))
        if (!sameId(id, this.formData.id)) return
        this.equipList = list.filter(item => !sameId(item.id, id)).map(item => ({ key: item.id, name: item.equipmentName || item.equipmentCode || String(item.id) }))
        if (!this.equipList.length) return Message.warning('不存在其他设备')
        this.copyEquipId = this.equipList[0].key
        this.copyOtherEquipAlarmDialog = true
        await this.copyEquipAlarmChange()
      } catch (error) { this.handlePersistError(error) }
    },
    async copyEquipAlarmChange() {
      const id = this.copyEquipId
      const sequence = ++this.copySequence
      this.copyLoading = true
      this.copyLabelsReady = false
      this.copyEquipLabelList = []
      try {
        const labels = await this.getLabelList(id)
        if (sequence !== this.copySequence || !sameId(id, this.copyEquipId)) return
        this.copyEquipLabelList = labels
        this.copyLabelsReady = true
      } catch (error) {
        if (sequence === this.copySequence) this.handlePersistError(error)
      } finally { if (sequence === this.copySequence) this.copyLoading = false }
    },
    async sureCopy() {
      if (!this.canOperate() || this.copyLoading) return
      if (!this.copyLabelsReady) return Message.error('来源设备的数据未加载成功，请重新选择设备')
      const sourceId = this.copyEquipId
      const targetId = this.formData.id
      if (sameId(sourceId, targetId) || !this.equipList.some(item => sameId(item.key, sourceId))) return Message.error('请选择其他设备')
      if (!this.formData.apiKey) return Message.error('设备 apiKey 缺失，请从设备列表重新进入管理页')
      this.operationLoading = true
      let copied = 0
      let skipped = 0
      try {
        const rules = getList(await gettrigger({ equipmentId: Number(sourceId) }))
        if (!sameId(targetId, this.formData.id) || !sameId(sourceId, this.copyEquipId)) throw new Error('设备已切换，请重新复制')
        const existing = new Set(this.tableData.map(row => `${row.triggerType}:${row.code}`))
        for (const item of rules.filter(row => this.belongsToEquipment(row, sourceId))) {
          // 先按实际数据点名称匹配，再按各自设备前缀后的名称匹配；有歧义时不复制。
          const sourceLabel = this.copyEquipLabelList.find(label => sameId(label.key, item.labelId))
          if (!sourceLabel) { skipped++; continue }
          const labelName = sourceLabel.labelName
          let matches = this.labelList.filter(label => label.labelName === labelName)
          if (!matches.length) matches = this.labelList.filter(label => getSuffix(label.labelName) === getSuffix(labelName))
          if (matches.length !== 1) { skipped++; continue }
          const row = this.setGettriggerDeatilMsg(item)
          const prefix = this.formData.equipmentCode || ''
          const code = prefix ? `${prefix}-${getSuffix(row.code)}` : getSuffix(row.code)
          const identity = `${row.triggerType}:${code}`
          if (!code || existing.has(identity)) { skipped++; continue }
          if (!sameId(targetId, this.formData.id)) throw new Error('设备已切换，已停止复制')
          const alarmUser = this.normalizeAlarmUser(row.alarmUser, this.getAlarmTypeCode(), false)
          // 复制仅保留业务配置，不携带来源设备的主键或关联记录主键。
          delete alarmUser.id
          delete alarmUser.triggerId
          await createtrigger({
            apiKey: this.formData.apiKey, siteId: this.getSiteId(), equipmentId: Number(targetId),
            code, labelId: matches[0].key, labelName: matches[0].labelName,
            cycle: row.cycle, triggerType: row.triggerType, formula: row.formula || 'x',
            logics: row.logics.map(logic => ({ logic: logic.logic, triggerValue: logic.triggerValue })),
            strMatchs: row.strMatchs.map(match => ({ matchStr: match.matchStr, ReplaceCodeFlag: match.ReplaceCodeFlag, replaceCode: match.replaceCode })),
            alarmUser, statisticConfig: this.defaultStatisticConfig(row.statisticConfig || {})
          })
          existing.add(identity)
          copied++
        }
        if (copied) {
          this.copyOtherEquipAlarmDialog = false
          await this.afterPersist(`成功复制${copied}条${this.alarmTypeName}配置${skipped ? `，跳过${skipped}条重复或无法匹配的配置` : ''}`)
        } else Message.warning('未找到可匹配的新配置，请检查数据项名称及重复配置')
      } catch (error) {
        this.handlePersistError(error)
        if (copied) {
          Message.warning(`已复制${copied}条配置，其余未完成，请刷新后核对`)
          this.$emit('data-change')
          await this.getDataList()
        }
      } finally { this.operationLoading = false }
    }
  }
}
</script>

<style lang="stylus" scoped>
.content {
  padding-top 20px
  min-height 720px
  .BannerCss {
    margin 0 2%
  }
  .app-container {
    padding 0 2%
  }
}
</style>
