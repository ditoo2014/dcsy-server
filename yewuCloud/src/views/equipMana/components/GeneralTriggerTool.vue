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
          <el-table-column label="触发器名称" align="center" min-width="120">
            <template #default="scope">
              <span>{{ scope.row.triggerName || scope.row.code }}</span>
            </template>
          </el-table-column>
          <el-table-column label="触发代码" align="center" min-width="100">
            <template #default="scope">
              <span>{{ scope.row.code }}</span>
            </template>
          </el-table-column>
          <el-table-column label="报警类型" align="center" width="90">
            <template #default="scope">
              <span>{{ parseAlarmCategory(scope.row.alarmUser && scope.row.alarmUser.alarmType) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="匹配数据点" align="center" min-width="120">
            <template #default="scope">
              <span>{{ scope.row.labelName }}</span>
            </template>
          </el-table-column>
          <el-table-column label="触发类型" align="center" min-width="140">
            <template #default="scope">
              <span>{{ scope.row.triggerModeName }}</span>
              <span v-if="scope.row.triggerModeName === '周期触发'">，间隔：{{ scope.row.realtimeCycle }}{{ scope.row.cycleTypeName }}</span>
            </template>
          </el-table-column>
          <el-table-column label="触发条件" align="center" min-width="120">
            <template #default="scope">
              <template v-if="scope.row.ignoreLogic">
                <span>忽略逻辑</span>
              </template>
              <template v-else-if="scope.row.triggerModel === 1">
                <div v-for="(item, index) in (scope.row.strMatchs || [])" :key="index" style="height:30px;line-height:30px;">
                  <span>{{ item.matchStr }}</span>
                </div>
              </template>
              <template v-else>
                <div v-for="(logicItem, index) in (scope.row.logics || [])" :key="index" style="height:30px;line-height:30px;">
                  <span>{{ logicItem.logic }} {{ logicItem.triggerValue }}</span>
                </div>
              </template>
            </template>
          </el-table-column>
          <el-table-column label="影响设备数" align="center" width="100">
            <template #default="scope">
              <span>{{ getAffectedDeviceCount(scope.row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" align="center" class-name="small-padding fixed-width" width="160">
            <template #default="scope">
              <el-button type="primary" size="small" @click="modify(scope.row)">修改</el-button>
              <el-button type="danger" size="small" @click="remove(scope.row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>

        <elDrawer v-model="showDrawer" :title="textMap[dialogStatus]" :mask-closable="false" width="440">
          <el-form ref="dataForm" :rules="rules" :model="form" size="small" label-width="110px">
            <el-row>
              <el-col :span="24">
                <el-form-item label="触发器名称" prop="triggerName">
                  <el-input v-model="form.triggerName" placeholder="用于展示区分" />
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="触发代码" prop="code">
                  <el-input v-model="form.code" :disabled="dialogStatus === 'modify'" />
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="报警类型">
                  <el-radio-group v-model="form.alarmCategory" @change="alarmCategoryChange">
                    <el-radio label="告警">告警</el-radio>
                    <el-radio label="预警">预警</el-radio>
                  </el-radio-group>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="告警等级">
                  <el-select v-model="form.alarmUser.alarmStage" class="filter-item" placeholder="请选择" filterable>
                    <el-option v-for="item in alarmStageList" :key="item.key" :label="item.name" :value="item.key" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="匹配数据点" prop="labelName">
                  <el-select
                    v-model="form.labelName"
                    class="filter-item"
                    filterable
                    allow-create
                    default-first-option
                    clearable
                    placeholder="请输入或选择匹配数据点名称"
                    style="width:100%"
                  >
                    <el-option v-for="item in labelList" :key="item.key" :label="item.name" :value="item.key" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="处理模式">
                  <el-radio-group v-model="form.triggerModel" :disabled="form.ignoreLogic">
                    <el-radio :label="0">浮点逻辑</el-radio>
                    <el-radio :label="1">字符串匹配</el-radio>
                  </el-radio-group>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="忽略逻辑">
                  <el-switch v-model="form.ignoreLogic" @change="ignoreLogicChange" />
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="是否进行统计">
                  <el-radio-group v-model="form.statisticsEnabled" @change="handleStatisticsEnabledChange">
                    <el-radio :label="true">是</el-radio>
                    <el-radio :label="false">否</el-radio>
                  </el-radio-group>
                </el-form-item>
              </el-col>
              <el-col v-if="form.triggerType === 1 || form.triggerType === 2" :span="24">
                <el-form-item label="统计类型">
                  <el-select v-model="form.statisticConfig.statisticType" class="filter-item" placeholder="请选择" filterable>
                    <el-option v-for="item in statisticTypeList" :key="item.key" :label="item.name" :value="item.key" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col v-if="form.triggerType === 1 || form.triggerType === 2" :span="24">
                <el-form-item label="使用手动统计值">
                  <el-radio-group v-model="form.statisticConfig.useStatisticValue">
                    <el-radio :label="true">是</el-radio>
                    <el-radio :label="false">否</el-radio>
                  </el-radio-group>
                </el-form-item>
              </el-col>
              <el-col v-if="(form.triggerType === 1 || form.triggerType === 2) && form.statisticConfig.useStatisticValue" :span="24">
                <el-form-item label="手动统计值">
                  <el-input v-model="form.statisticConfig.statisticValue" />
                </el-form-item>
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
                    <el-option :key="1" label="单次触发" :value="1" />
                    <el-option :key="2" label="周期触发" :value="2" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col v-if="form.triggerMode === 2" :span="24">
                <el-form-item label="触发间隔" prop="realtimeCycle">
                  <div style="float:left;margin-right:10px;width:100px;">
                    <el-input v-model="form.realtimeCycle" type="number" :min="1" />
                  </div>
                  <div style="float:left;width:100px;">
                    <el-select v-model="form.cycleType" class="filter-item" placeholder="请选择" filterable @change="cycleChange">
                      <el-option v-for="item in realtimeCycleList" :key="item.key" :label="item.name" :value="item.key" />
                    </el-select>
                  </div>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="持续时间(秒)">
                  <el-input v-model="form.sustainTime" type="number" :min="0" />
                </el-form-item>
              </el-col>
              <el-col v-if="!form.ignoreLogic && form.triggerModel === 0" :span="24">
                <el-form-item label="逻辑" prop="logics">
                  <div v-for="(logic, index) in form.logics" :key="index" style="height:40px">
                    <div style="float:left;width:100px;">
                      <el-select v-model="logic.logic" class="filter-item" placeholder="请选择" filterable>
                        <el-option v-for="item in judgeList" :key="item.key" :label="item.name" :value="item.key" />
                      </el-select>
                    </div>
                    <div style="float:left;margin-left:10px;width:100px;">
                      <el-input v-model="logic.triggerValue" type="number" />
                    </div>
                    <div style="float:left;margin-left:10px;width:40px">
                      <el-button style="float:right" type="danger" :icon="$legacyIcon('el-icon-close')" circle @click="removelogics(index)" />
                    </div>
                  </div>
                  <el-button type="primary" :icon="$legacyIcon('el-icon-plus')" circle @click="addlogics" />
                </el-form-item>
              </el-col>
              <el-col v-if="!form.ignoreLogic && form.triggerModel === 1" :span="24">
                <el-form-item label="字符串匹配" prop="strMatchs">
                  <div v-for="(match, index) in form.strMatchs" :key="index" style="margin-bottom:8px;">
                    <el-input v-model="match.matchStr" placeholder="匹配字符串" style="width:200px;margin-right:8px;" />
                    <el-button type="danger" :icon="$legacyIcon('el-icon-close')" circle @click="removeStrMatch(index)" />
                  </div>
                  <el-button type="primary" :icon="$legacyIcon('el-icon-plus')" circle @click="addStrMatch" />
                </el-form-item>
              </el-col>
              <el-col v-if="!globalMode" :span="24">
                <el-form-item label="仅当前设备">
                  <el-switch v-model="form.limitToCurrentDevice" @change="limitDeviceChange" />
                </el-form-item>
              </el-col>
              <el-col v-if="globalMode || !form.limitToCurrentDevice" :span="24">
                <el-form-item label="设备范围">
                  <el-select
                    v-model="form.equipmentIds"
                    class="filter-item"
                    multiple
                    filterable
                    placeholder="不选则匹配全部设备的同名数据点"
                    style="width:100%"
                    @change="onEquipmentIdsChange"
                  >
                    <el-option v-for="item in equipList" :key="item.key" :label="item.name" :value="item.key" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col v-if="form.marktriggers && form.marktriggers.length" :span="24">
                <el-form-item label="已匹配设备">
                  <span>{{ form.marktriggers.length }} 台</span>
                </el-form-item>
              </el-col>
            </el-row>
          </el-form>
          <div class="fixedBottomCss">
            <el-button @click="showDrawer = false">取消</el-button>
            <el-button type="primary" @click="setData">保存</el-button>
          </div>
        </elDrawer>
      </el-scrollbar>
    </div>
  </div>
</template>

<script>
import { Message } from '@/utils/element-shim'
import Banner from '@/components/ccpcComp/Banner.vue'
import { meConfirm, isPosiInt, isNumber } from '@/utils/utils.js'
import { judgeList, alarmStageList } from '@/utils/ccpcData.js'
import { findDataDetails, findEquipmentListExactMatch } from '@/api/equipMana.js'
import {
  findGeneralTrigger,
  addGeneralTrigger,
  updateGeneralTrigger,
  deleteGeneralTrigger,
  appliesToEquipment
} from '@/api/generaltrigger.js'

export default {
  components: {
    Banner
  },
  props: {
    formData: {
      type: Object,
      default: () => ({})
    },
    dataChangeKey: {
      type: Number,
      default: 0
    },
    globalMode: {
      type: Boolean,
      default: false
    }
  },
  data() {
    const validateRealtimeCycle = (rule, value, callback) => {
      if (this.form.triggerMode !== 2) {
        callback()
        return
      }
      const n = Math.floor(Number(value))
      if (isPosiInt(value) && n > 0) {
        callback()
      } else {
        callback(new Error('请输入大于0的正整数'))
      }
    }
    const validateLogics = (rule, value, callback) => {
      if (this.form.ignoreLogic || this.form.triggerModel !== 0) {
        callback()
        return
      }
      if (!value || value.length === 0) {
        callback(new Error('请至少添加一条逻辑'))
        return
      }
      let isAllNumber = true
      value.forEach(item => {
        if (!isNumber(item.triggerValue)) {
          isAllNumber = false
        }
      })
      if (isAllNumber) {
        callback()
      } else {
        callback(new Error('逻辑输入框请填数字'))
      }
    }
    const validateStrMatchs = (rule, value, callback) => {
      if (this.form.ignoreLogic || this.form.triggerModel !== 1) {
        callback()
        return
      }
      if (!value || value.length === 0 || !value.some(item => item.matchStr)) {
        callback(new Error('请至少添加一条字符串匹配'))
        return
      }
      callback()
    }
    const bannerList = [
      { style: 'infoNum', title: '通用触发器数', info: '0', float: 'left' },
      { style: 'button', buttonName: '添加通用触发器', type: 'primary', icon: 'el-icon-circle-plus', float: 'right' },
      { style: 'button', buttonName: '批量删除', type: 'danger', float: 'right' }
    ]
    if (!this.globalMode) {
      bannerList.splice(1, 0, { style: 'infoNum', title: '设备名称', info: ' ', float: 'left' })
    }
    return {
      tableLoading: false,
      bannerList,
      tableKey: 1,
      showDrawer: false,
      textMap: {
        add: '添加通用触发器',
        modify: '修改通用触发器'
      },
      dialogStatus: '',
      rules: {
        triggerName: [{ required: true, trigger: 'blur', message: '触发器名称不能为空' }],
        code: [{ required: true, trigger: 'blur', message: '触发代码不能为空' }],
        labelName: [{ required: true, trigger: ['blur', 'change'], message: '请输入或选择匹配数据点' }],
        realtimeCycle: [{ validator: validateRealtimeCycle, trigger: 'blur' }],
        logics: [{ validator: validateLogics, trigger: 'blur' }],
        strMatchs: [{ validator: validateStrMatchs, trigger: 'blur' }]
      },
      tableData: [],
      form: {},
      statisticValueTypeList: [
        { key: 0, name: '累计值' },
        { key: 1, name: '增量值' }
      ],
      realtimeCycleList: [
        { key: 1, name: '秒' },
        { key: 2, name: '分钟' },
        { key: 3, name: '小时' }
      ],
      alarmStageList,
      labelList: [],
      judgeList,
      selectList: [],
      equipList: []
    }
  },
  computed: {
    statisticTypeList() {
      const countLabel = this.form.alarmCategory === '预警' ? '预警条数统计' : '告警条数统计'
      return [
        { key: '0', name: '通用统计' },
        { key: '1', name: countLabel }
      ]
    }
  },
  watch: {
    dataChangeKey() {
      this.refreshLabelList()
    },
    'formData.id'(id) {
      if (!this.globalMode && id) {
        this.init()
      }
    },
    'formData.equipmentName'(name) {
      if (!this.globalMode && name && this.bannerList[1]) {
        this.bannerList[1].info = name
      }
    }
  },
  mounted() {
    if (this.globalMode) {
      this.init()
      return
    }
    if (this.formData.equipmentName && this.bannerList[1]) {
      this.bannerList[1].info = this.formData.equipmentName
    }
    if (this.formData.id) {
      this.init()
    }
  },
  methods: {
    alarmCategoryToType(category) {
      return category === '预警' ? '1' : '0'
    },
    parseAlarmCategory(alarmType) {
      if (alarmType === null || alarmType === undefined || alarmType === '') {
        return '告警'
      }
      const typeStr = String(alarmType)
      if (typeStr === '1' || typeStr.indexOf('1') !== -1 || typeStr.indexOf('3') !== -1) {
        return '预警'
      }
      return '告警'
    },
    buildDefaultAlarmUser(alarmType) {
      return {
        alarmCode: '',
        alarmType: alarmType || '0',
        alarmStage: this.alarmStageList[0].key,
        alarmChannel: '0',
        alarmUsers: ''
      }
    },
    normalizeAlarmUser(alarmUser, alarmType) {
      const resolvedType = alarmType !== undefined ? alarmType : (alarmUser && alarmUser.alarmType)
      return {
        ...this.buildDefaultAlarmUser(resolvedType || '0'),
        ...(alarmUser || {}),
        alarmType: alarmType !== undefined ? alarmType : ((alarmUser && alarmUser.alarmType) || '0')
      }
    },
    alarmCategoryChange(category) {
      if (!this.form.alarmUser) {
        this.form.alarmUser = this.buildDefaultAlarmUser(this.alarmCategoryToType(category))
        return
      }
      this.form.alarmUser.alarmType = this.alarmCategoryToType(category)
    },
    emitDataChange() {
      this.$emit('data-change')
    },
    getAffectedDeviceCount(row) {
      const marks = row.marktriggers || []
      if (marks.length) {
        return marks.length
      }
      const ids = row.equipmentIds || []
      return ids.length || '全部匹配'
    },
    async refreshLabelList() {
      const firstEquipId = this.form && this.form.equipmentIds && this.form.equipmentIds[0]
      this.labelList = await this.getLabelNameList(firstEquipId)
    },
    async onEquipmentIdsChange() {
      if (this.globalMode) {
        await this.refreshLabelList()
      }
    },
    async loadEquipList() {
      const equipList = []
      try {
        const res = await findEquipmentListExactMatch({})
        const list = res.return && res.return.data
        if (list) {
          list.forEach(item => {
            equipList.push({
              key: Number(item.id),
              name: item.equipmentName || item.equipmentCode || String(item.id)
            })
          })
        }
      } catch {
        // 设备列表加载失败时不阻断表单
      }
      this.equipList = equipList
    },
    async init() {
      if (!this.globalMode && !this.formData.id) {
        return
      }
      try {
        await Promise.all([
          this.refreshLabelList(),
          this.loadEquipList()
        ])
        await this.getDataList()
      } catch (error) {
        this.handlePersistError(error)
      }
    },
    async getLabelNameList(equipmentId) {
      const id = equipmentId || this.formData.id
      const labelList = []
      if (!id) {
        return labelList
      }
      const res = await findDataDetails({ equipmentId: id })
      const list = res.return && res.return.data
      const seen = new Set()
      if (list) {
        list.forEach(item => {
          const name = item.labelName || item.name
          if (name && !seen.has(name)) {
            seen.add(name)
            labelList.push({ key: name, name })
          }
        })
      }
      return labelList
    },
    async getDataList() {
      this.tableLoading = true
      const tableData = []
      try {
        const res = await findGeneralTrigger()
        const list = (res.return && res.return.data) || []
        list.forEach(item => {
          if (this.globalMode || appliesToEquipment(item, this.formData.id)) {
            tableData.push(this.normalizeRow(item))
          }
        })
        this.tableData = tableData
        this.bannerList[0].info = String(tableData.length)
      } catch (error) {
        this.tableData = []
        this.bannerList[0].info = '0'
        this.handlePersistError(error)
      } finally {
        this.tableLoading = false
      }
    },
    normalizeRow(item) {
      const row = { ...item }
      const cycle = Number(row.cycle) || 0
      row.cycle = cycle
      row.triggerMode = 1
      row.triggerModeName = '单次触发'
      row.cycleType = 1
      row.realtimeCycle = 0
      row.cycleTypeName = '秒'
      row.logics = row.logics || []
      row.strMatchs = row.strMatchs || []
      row.ignoreLogic = !!row.ignoreLogic
      row.triggerModel = row.triggerModel ?? 0
      if (cycle !== 0) {
        row.realtimeCycle = cycle
        row.triggerMode = 2
        row.triggerModeName = '周期触发'
      }
      if (cycle !== 0 && cycle % 3600 === 0) {
        row.cycleType = 3
        row.realtimeCycle = cycle / 3600
      } else if (cycle !== 0 && cycle % 60 === 0) {
        row.cycleType = 2
        row.realtimeCycle = cycle / 60
      }
      this.realtimeCycleList.forEach(cycleItem => {
        if (row.cycleType === cycleItem.key) {
          row.cycleTypeName = cycleItem.name
        }
      })
      switch (row.triggerType) {
        case 0: row.triggerTypeArr = ['告警']; break
        case 1: row.triggerTypeArr = ['统计']; break
        case 2: row.triggerTypeArr = ['告警', '统计']; break
        default: row.triggerTypeArr = ['告警']
      }
      return row
    },
    async clickBannerBtn(item) {
      if (item.buttonName === '添加通用触发器') {
        this.add()
      } else if (item.buttonName === '批量删除') {
        this.batchRemove()
      }
    },
    add() {
      this.dialogStatus = 'add'
      if (!this.resetForm()) {
        return
      }
      this.showDrawer = true
    },
    modify(row) {
      this.dialogStatus = 'modify'
      this.resetForm(row)
      this.showDrawer = true
    },
    resetForm(row) {
      if (this.dialogStatus === 'add') {
        this.form = {
          triggerName: '',
          code: '',
          labelName: '',
          alarmCategory: '告警',
          triggerModel: 0,
          ignoreLogic: false,
          triggerMode: 1,
          cycleType: 1,
          realtimeCycle: 0,
          cycle: 0,
          sustainTime: 0,
          triggerType: 2,
          statisticsEnabled: true,
          formula: 'x',
          logics: [],
          strMatchs: [],
          limitToCurrentDevice: !this.globalMode,
          equipmentIds: this.globalMode ? [] : [Number(this.formData.id)],
          marktriggers: [],
          alarmUser: this.buildDefaultAlarmUser('0'),
          statisticConfig: {
            useStatisticValue: false,
            statisticCode: '',
            statisticType: '1',
            statisticValue: 1,
            statisticValueType: 1,
            dataSourceType: 0
          }
        }
        this.addlogics()
      } else if (this.dialogStatus === 'modify') {
        const statisticConfig = row.statisticConfig || {}
        const equipmentIds = (row.equipmentIds || []).map(id => Number(id))
        const currentId = Number(this.formData.id)
        const limitToCurrentDevice = !this.globalMode &&
          equipmentIds.length === 1 && equipmentIds[0] === currentId
        const alarmCategory = this.parseAlarmCategory(row.alarmUser && row.alarmUser.alarmType)
        this.form = {
          id: row.id,
          triggerName: row.triggerName || '',
          code: row.code,
          labelName: row.labelName,
          alarmCategory,
          triggerModel: row.triggerModel ?? 0,
          ignoreLogic: !!row.ignoreLogic,
          triggerMode: row.triggerMode,
          cycleType: row.cycleType,
          realtimeCycle: row.realtimeCycle,
          cycle: row.cycle,
          sustainTime: row.sustainTime || 0,
          triggerType: row.triggerType,
          formula: row.formula || 'x',
          logics: (row.logics || []).map(item => ({ ...item })),
          strMatchs: (row.strMatchs || []).map(item => ({ ...item })),
          limitToCurrentDevice,
          equipmentIds: limitToCurrentDevice ? [currentId] : equipmentIds,
          marktriggers: row.marktriggers || [],
          alarmUser: this.normalizeAlarmUser(row.alarmUser, this.alarmCategoryToType(alarmCategory)),
          statisticConfig: {
            useStatisticValue: false,
            statisticCode: '',
            statisticType: '1',
            statisticValue: 1,
            ...statisticConfig,
            statisticValueType: statisticConfig.statisticValueType ?? 1
          }
        }
        this.form.statisticsEnabled = this.form.triggerType === 1 || this.form.triggerType === 2
        if (!this.form.ignoreLogic && this.form.triggerModel === 0 && this.form.logics.length === 0) {
          this.addlogics()
        }
        if (!this.form.ignoreLogic && this.form.triggerModel === 1 && this.form.strMatchs.length === 0) {
          this.addStrMatch()
        }
        if (this.globalMode) {
          this.refreshLabelList()
        }
      }
      return true
    },
    ignoreLogicChange() {
      if (this.form.ignoreLogic) {
        this.form.logics = []
        this.form.strMatchs = []
      } else if (this.form.triggerModel === 0 && this.form.logics.length === 0) {
        this.addlogics()
      } else if (this.form.triggerModel === 1 && this.form.strMatchs.length === 0) {
        this.addStrMatch()
      }
    },
    handleStatisticsEnabledChange() {
      if (this.form.statisticsEnabled) {
        this.form.triggerType = 2
      } else {
        this.form.triggerType = 0
      }
    },
    handleTriggerModeChange() {
      switch (this.form.triggerMode) {
        case 1: this.form.realtimeCycle = 0; this.form.cycleType = 1; break
        case 2: this.form.realtimeCycle = 1; break
      }
      this.cycleChange()
    },
    cycleChange() {
      switch (this.form.cycleType) {
        case 1: this.form.cycle = this.form.realtimeCycle; break
        case 2: this.form.cycle = this.form.realtimeCycle * 60; break
        case 3: this.form.cycle = this.form.realtimeCycle * 3600; break
      }
    },
    addlogics() {
      this.form.logics.push({ logic: '>', triggerValue: 0 })
    },
    removelogics(index) {
      this.form.logics = this.form.logics.filter((_, i) => i !== index)
    },
    addStrMatch() {
      this.form.strMatchs.push({ matchStr: '', replaceCodeFlag: false, replaceCode: '' })
    },
    removeStrMatch(index) {
      this.form.strMatchs = this.form.strMatchs.filter((_, i) => i !== index)
    },
    async limitDeviceChange(limit) {
      if (limit) {
        this.form.equipmentIds = [Number(this.formData.id)]
        return
      }
      const sure = await meConfirm(this, {
        title: '确认范围变更',
        content: '关闭后可为多台设备或全部匹配同名数据点的设备创建通用触发器，是否继续？'
      })
      if (!sure) {
        this.form.limitToCurrentDevice = true
        this.form.equipmentIds = [Number(this.formData.id)]
      } else {
        this.form.equipmentIds = []
      }
    },
    buildSubmitForm() {
      this.cycleChange()
      const payload = { ...this.form }
      if (!this.globalMode && payload.limitToCurrentDevice) {
        payload.equipmentIds = [Number(this.formData.id)]
      }
      if (!payload.alarmUser) {
        payload.alarmUser = this.buildDefaultAlarmUser(this.alarmCategoryToType(payload.alarmCategory))
      } else {
        payload.alarmUser.alarmType = this.alarmCategoryToType(payload.alarmCategory)
      }
      delete payload.alarmCategory
      delete payload.limitToCurrentDevice
      delete payload.marktriggers
      delete payload.triggerMode
      delete payload.triggerModeName
      delete payload.cycleType
      delete payload.cycleTypeName
      delete payload.realtimeCycle
      delete payload.statisticsEnabled
      delete payload.triggerTypeArr
      payload.sustainTime = Number(payload.sustainTime) || 0
      return payload
    },
    handlePersistSuccess(title, message) {
      this.$notify({ title, message, type: 'success', duration: 3000 })
      this.getDataList()
      this.showDrawer = false
      this.emitDataChange()
    },
    handlePersistError(error) {
      const errMsg = (error && error.message) || error || '操作失败，请稍后重试'
      Message.error(typeof errMsg === 'string' ? errMsg : '操作失败，请稍后重试')
    },
    setData() {
      const formRef = this.$refs.dataForm
      if (!formRef) {
        Message.error('表单未就绪，请稍后重试')
        return
      }
      formRef.validate(valid => {
        if (!valid) {
          return
        }
        const payload = this.buildSubmitForm()
        if (this.dialogStatus === 'add') {
          addGeneralTrigger(payload).then(() => {
            this.handlePersistSuccess('添加成功', '成功添加通用触发器')
          }).catch(this.handlePersistError)
        } else if (this.dialogStatus === 'modify') {
          updateGeneralTrigger(payload).then(() => {
            this.handlePersistSuccess('修改成功', '成功修改通用触发器')
          }).catch(this.handlePersistError)
        }
      })
    },
    async remove(row) {
      const sure = await meConfirm(this, {
        title: '确认删除',
        content: `此操作将永久删除 ${row.code}，是否继续？`
      })
      if (!sure) {
        return
      }
      try {
        await deleteGeneralTrigger({ id: row.id })
        Message.success(`${row.code} 删除成功`)
        this.getDataList()
        this.emitDataChange()
      } catch (error) {
        this.handlePersistError(error)
      }
    },
    async batchRemove() {
      if (this.selectList.length === 0) {
        Message.warning('请先选择要删除的记录')
        return
      }
      const sure = await meConfirm(this, {
        title: '确认删除',
        content: `此操作将批量删除 ${this.selectList.length} 条记录，是否继续？`
      })
      if (!sure) {
        return
      }
      try {
        for (let i = 0; i < this.selectList.length; i++) {
          await deleteGeneralTrigger({ id: this.selectList[i].id })
        }
        Message.success(`成功删除 ${this.selectList.length} 条记录`)
        this.selectList = []
        this.getDataList()
        this.emitDataChange()
      } catch (error) {
        this.handlePersistError(error)
      }
    },
    handleSelectionChange(val) {
      this.selectList = val
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
