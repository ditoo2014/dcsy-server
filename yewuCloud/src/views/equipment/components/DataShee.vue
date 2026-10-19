<template>
  <div v-loading="loading" class="content gray">
    <div class="BannerCss">
      <Banner :list="bannerList" @click-button="clickBannerBtn" />
    </div>
    <div class="app-container">
      <div class="filter-container">
        <el-input
          v-model="searchMsg"
          placeholder="请输入数据名称"
          style="width: 200px;"
          class="filter-item"
          @keyup.enter="search"
        />
        <el-button
          class="filter-item"
          style="margin-left:10px"
          size="small"
          type="primary"
          :icon="$legacyIcon('el-icon-search')"
          @click="search"
        >查询</el-button>
      </div>
      <el-scrollbar class="allheight">
        <!-- 保留原版列顺序、宽度和颜色提示，拖拽仅作用于本组件的表格。 -->
        <el-table
          ref="dataTable"
          :key="tableKey"
          :data="tableData"
          row-key="id"
          fit
          highlight-current-row
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="55" />
          <el-table-column label="序号" prop="sortNum" align="center" width="80">
            <template #default="scope">
              <span>{{ scope.row.sortNum }}</span>
            </template>
          </el-table-column>
          <el-table-column label="数据名称" align="center">
            <template #default="scope">
              <span :style="{color:scope.row.hasLabel?'':'red'}">{{ scope.row.name }}</span>
            </template>
          </el-table-column>
          <el-table-column label="数据标签" align="center">
            <template #default="scope">
              <span :style="{color:scope.row.repeatLabel?'orange':''}">{{ scope.row.labelName }}</span>
            </template>
          </el-table-column>
          <el-table-column label="限值" align="center">
            <template #default="scope"><span>{{ scope.row.limitValue }}</span></template>
          </el-table-column>
          <el-table-column label="保留小数点位数" align="center">
            <template #default="scope"><span>{{ scope.row.decimalPlaces }}</span></template>
          </el-table-column>
          <el-table-column label="曲线颜色" align="center" width="80">
            <template #default="scope">
              <div style="margin-left: 15px">
                <VueColor :color="scope.row.color" :disabled="true" :color-select-ed="false" />
              </div>
            </template>
          </el-table-column>
          <el-table-column label="单位" align="center">
            <template #default="scope"><span>{{ scope.row.labelUnit }}</span></template>
          </el-table-column>
          <el-table-column label="备注" align="center">
            <template #default="scope"><span>{{ scope.row.remark }}</span></template>
          </el-table-column>
          <el-table-column label="状态灯" align="center" width="140">
            <template #default="scope">
              <div v-for="(statusItem, index) in scope.row.statusLightArr" :key="index" style="height:40px;line-height: 30px;">
                <span style="float: left">{{ statusItem.name }}{{ statusItem.value }}时显示</span>
                <span v-if="statusItem.isMsg">{{ statusItem.color }}</span>
                <span v-else><VueColor :color="statusItem.color" :disabled="true" :color-select-ed="false" /></span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" align="center" class-name="small-padding fixed-width" width="210">
            <template #default="scope">
              <el-button size="small" @click="setMsg(scope.row)">设置</el-button>
              <el-button type="primary" size="small" @click="modify(scope.row)">修改</el-button>
              <el-button type="danger" size="small" @click="remove(scope.row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <!-- 右侧弹窗沿用业务云抽屉封装，保持原版 600 像素宽度。 -->
        <ElDrawerWrap v-model="showDrawer" :title="textMap[dialogStatus]" :mask-closable="false" width="600">
          <el-form ref="dataForm" :rules="rules" :model="form" size="small" label-width="120px">
            <el-row v-if="dialogStatus === 'add' || dialogStatus === 'modify'">
              <el-col v-if="labelSelectType === '1' && form.labelId !== '全部'" :span="24">
                <el-form-item label="序号" prop="sortNum">
                  <div style="width:80%"><el-input v-model="form.sortNum" type="number" /></div>
                </el-form-item>
              </el-col>
              <el-col v-if="labelSelectType === '1'" :span="24">
                <el-form-item label="数据名称" prop="name">
                  <div style="width:80%"><el-input v-model="form.name" :disabled="form.labelId === '全部'" /></div>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="数据名称是否去掉'-'之前的内容">
                  <el-radio v-model="isRem" value="1">去掉</el-radio>
                  <el-radio v-model="isRem" value="2">不去掉</el-radio>
                </el-form-item>
              </el-col>
              <el-col v-if="dialogStatus === 'add'" :span="24">
                <el-form-item label="数据池选择方式">
                  <el-radio v-model="labelSelectType" value="1">单选</el-radio>
                  <el-radio v-model="labelSelectType" value="2">多选</el-radio>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="数据池">
                  <div style="width:80%">
                    <el-select v-show="dialogStatus === 'add' && labelSelectType === '2'" v-model="form.labelIds" filterable multiple @change="labelChange">
                      <el-option v-for="item in labelList" :key="item.key" :label="item.name" :value="item.key" />
                    </el-select>
                    <el-select v-show="labelSelectType === '1'" v-model="form.labelId" filterable @change="labelChange">
                      <el-option v-for="item in labelList" :key="item.key" :label="item.name" :value="item.key" />
                    </el-select>
                  </div>
                </el-form-item>
              </el-col>
            </el-row>
            <el-row v-if="dialogStatus === 'setMsg'">
              <el-col :span="24">
                <el-form-item label="复制属性">
                  <div style="width:80%">
                    <el-select v-model="form.copyDataPointId" filterable @change="copyDataPointChange">
                      <el-option v-for="item in dataPointList" :key="item.key" :label="item.name" :value="item.key" />
                    </el-select>
                  </div>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="保留小数点位数" prop="decimalPlaces">
                  <div style="width:50%"><el-input v-model="form.decimalPlaces" type="number" :min="0" /></div>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                曲线画面设置：
                <el-form-item label="限值">
                  <div style="width:50%"><el-input v-model="form.limitValue" type="number" :min="0" placeholder="用于曲线显示的比例限值" /></div>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="曲线颜色">
                  <div style="float:left;width:22%"><el-input v-model="form.color" disabled /></div>
                  <div style="float:left;margin-left:3%"><VueColor :color="form.color" @color-select="selectColor" /></div>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                数据设置：
                <el-form-item label="单位">
                  <div style="width:50%"><el-input v-model="form.labelUnit" /></div>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="备注">
                  <div style="width:50%"><el-input v-model="form.remark" /></div>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="状态灯" prop="statusLightArr">
                  <div v-for="(statusItem, index) in form.statusLightArr" :key="index" style="height:40px">
                    <div style="float:left;width:25%">
                      <el-select v-model="statusItem.key" class="filter-item" placeholder="请选择" filterable>
                        <el-option v-for="item in judgeList" :key="item.key" :label="item.name" :value="item.key" />
                      </el-select>
                    </div>
                    <div style="float:left;margin-left:10px;width:20%"><el-input v-model="statusItem.value" /></div>
                    <div style="float:left;margin-left:10px;width:20%"><el-input v-model="statusItem.color" placeholder="颜色Hex" /></div>
                    <div
                      style="float:left;margin-left:10px;padding: 1px;border:2px solid"
                      :style="{'border-color': selectStatusLightIndex === index ? '#1890FF' : '#fff'}"
                      @click="statusLightSelColor(index)"
                    >
                      <VueColor :color="statusItem.color" :disabled="true" :color-select-ed="false" />
                    </div>
                    <div style="float:left;margin-left:10px;width:40px">
                      <el-button style="float:right" type="danger" :icon="$legacyIcon('el-icon-close')" circle @click="removeStatusLight(index)" />
                    </div>
                  </div>
                  <el-button type="primary" :icon="$legacyIcon('el-icon-plus')" circle @click="addStatusLight" />
                  <div style="margin-top: 10px">
                    <div style="float:left;width:20%;margin-right:3%">选择颜色Hex:</div>
                    <div style="float:left"><VueColor @color-select="sureStatusLightSelColor" /></div>
                  </div>
                </el-form-item>
              </el-col>
            </el-row>
          </el-form>
          <div class="fixedBottomCss">
            <el-button :disabled="saving" @click="showDrawer = false">取消</el-button>
            <el-button type="primary" :loading="saving" @click="setData">确定</el-button>
          </div>
        </ElDrawerWrap>
      </el-scrollbar>
    </div>
  </div>
</template>

<script>
import { markRaw } from 'vue'
import Sortable from 'sortablejs'
import { Message } from '@/utils/elementShim'
import Banner from '@/components/ccpcComp/Banner.vue'
import VueColor from '@/components/sharedComponents/vueColor/VueColor.vue'
import ElDrawerWrap from '@/components/elDrawer/index.vue'
import { meAlert, checkIsColor } from '@/utils/utils.js'
import { findDataDetails, saveDataDetails, deleteDataDetails, modifyDataDetails, getlabel } from '@/api/equipment.js'
import { judgeList } from '@/utils/ccpcData.js'

const DEFAULT_COLOR = '#7AF7BE'
const clone = value => JSON.parse(JSON.stringify(value))

export default {
  name: 'DataShee',
  components: { Banner, VueColor, ElDrawerWrap },
  props: {
    formData: { type: Object, default: () => ({}) },
    dataChangeKey: { type: Number, default: 0 }
  },
  data() {
    const validateSortNum = (rule, value, callback) => {
      const number = Number(value)
      if (value === '' || value == null || !Number.isSafeInteger(number) || number < 1) {
        callback(new Error('序号必须为正整数'))
      } else if (this.allTableData.some(item => Number(item.sortNum) === number && item.id !== this.form.id)) {
        // 修改时排除当前数据点，避免未改变序号却不能提交。
        callback(new Error('序号已存在'))
      } else {
        callback()
      }
    }
    const validateDecimalPlaces = (rule, value, callback) => {
      const number = Number(value)
      if (value === '' || value == null || !Number.isSafeInteger(number) || number < 0) {
        callback(new Error('保留小数点位数必须为非负整数'))
      } else {
        callback()
      }
    }
    return {
      pendingRequests: 0,
      saving: false,
      disposed: false,
      requestVersion: 0,
      sortable: null,
      bannerList: [
        { style: 'infoNum', title: '数据总量(个)', info: ' ', float: 'left' },
        { style: 'infoNum', title: '设备名称：', info: ' ', float: 'left' },
        { style: 'button', buttonName: '批量删除', type: 'danger', icon: 'el-icon-delete', float: 'right' },
        { style: 'button', buttonName: '添加数据', type: 'primary', icon: 'el-icon-circle-plus', float: 'right' }
      ],
      tableKey: 1,
      searchMsg: '',
      showDrawer: false,
      textMap: { add: '添加数据', modify: '修改数据', setMsg: '属性设置' },
      dialogStatus: '',
      rules: {
        sortNum: [{ required: true, trigger: 'blur', validator: validateSortNum }],
        decimalPlaces: [{ required: true, trigger: 'blur', validator: validateDecimalPlaces }]
      },
      tableData: [],
      allTableData: [], // 完整列表用于序号校验和批量去重，避免搜索后遗漏已有数据点。
      dataLoaded: false, // 只有确认列表查询成功后才能操作，查询失败不能当成空表。
      selectedTableData: [],
      form: {},
      isRem: '2', // 原版名称处理：1 去掉 '-' 之前的内容，2 保留完整名称。
      labelList: [],
      lastAddLabelId: '',
      labelSelectType: '1',
      contrLabelList: [],
      labelsLoaded: false,
      dataPointList: [],
      judgeList,
      selectStatusLightIndex: ''
    }
  },
  computed: {
    loading() { return this.pendingRequests > 0 }
  },
  watch: {
    dataChangeKey() { this.getContrLabelList() },
    'formData.id'() {
      // 切换设备立即使旧查询和编辑状态失效，避免使用旧设备列表操作新设备。
      this.dataLoaded = false
      this.requestVersion++
      this.showDrawer = false
      this.destroySortable()
      this.getContrLabelList()
    },
    'formData.apiKey'() { this.getContrLabelList() },
    isRem() { this.labelChange() }
  },
  mounted() {
    this.getContrLabelList()
  },
  beforeUnmount() {
    // 关闭标签时释放拖拽监听，并忽略已经发出的旧查询结果。
    this.disposed = true
    this.dataLoaded = false
    this.requestVersion++
    this.destroySortable()
  },
  methods: {
    showError(error, fallback) {
      if (!this.disposed) Message.error((error && error.message) || fallback)
    },
    requireDataLoaded() {
      if (this.disposed) return false
      if (this.dataLoaded) return true
      Message.warning('数据表尚未成功加载，请先重新查询后操作')
      return false
    },
    destroySortable() {
      if (this.sortable) {
        this.sortable.destroy()
        this.sortable = null
      }
    },
    async rowDrop() {
      await this.$nextTick()
      this.destroySortable()
      if (this.disposed || !this.dataLoaded) return
      const tbody = this.$refs.dataTable?.$el.querySelector('.el-table__body-wrapper tbody')
      if (!tbody) return
      // 第三方实例不能被 Vue 的响应式代理包装，否则会影响内部 DOM 操作。
      this.sortable = markRaw(Sortable.create(tbody, {
        animation: 150,
        filter: 'button, input, .el-button, .el-checkbox',
        preventOnFilter: false,
        onEnd: event => { this.persistRowOrder(event) }
      }))
    },
    async persistRowOrder({ newIndex, oldIndex, item, from }) {
      if (newIndex == null || oldIndex == null || newIndex === oldIndex) return
      // Sortable 已调整真实 DOM，先恢复旧 DOM 顺序，再交给 Vue 渲染新数据。
      if (item && from) {
        const restoreIndex = oldIndex + (newIndex < oldIndex ? 1 : 0)
        from.insertBefore(item, from.children[restoreIndex] || null)
      }
      if (!this.requireDataLoaded()) return
      const reordered = [...this.tableData]
      const [moved] = reordered.splice(oldIndex, 1)
      if (!moved) return
      reordered.splice(newIndex, 0, moved)
      const oldNumbers = this.tableData.map(row => Number(row.sortNum))
      const validNumbers = oldNumbers.every(number => Number.isSafeInteger(number) && number > 0) && new Set(oldNumbers).size === oldNumbers.length
      const changes = reordered.map((row, index) => ({ ...row, sortNum: validNumbers ? oldNumbers[index] : index + 1 }))
        .filter((row, index) => row.sortNum !== Number(reordered[index].sortNum))
      this.pendingRequests++
      try {
        // 后端不自动移动其他行，必须保存所有受影响的数据点以避免重复序号。
        for (const row of changes) await modifyDataDetails(row)
        Message.success('成功重新排序')
        this.$emit('data-change')
      } catch (error) {
        this.showError(error, '重新排序失败')
      } finally {
        await this.getDataList()
        this.pendingRequests--
      }
    },
    async getContrLabelList() {
      if (!this.formData.id || this.disposed) return
      this.dataLoaded = false
      const equipmentId = this.formData.id
      this.bannerList[1].info = this.formData.equipmentName || ''
      this.pendingRequests++
      try {
        if (this.formData.apiKey) {
          const res = await getlabel({ ApiKey: this.formData.apiKey })
          if (this.disposed || equipmentId !== this.formData.id) return
          this.contrLabelList = Array.isArray(res.return?.list) ? res.return.list : []
          this.labelsLoaded = true
        } else {
          this.contrLabelList = []
          this.labelsLoaded = false
        }
      } catch (error) {
        this.labelsLoaded = false
        this.showError(error, '读取数据池标签失败')
      } finally {
        // 数据池暂不可用也要显示已有业务数据，避免整张数据表空白。
        if (!this.disposed && equipmentId === this.formData.id) await this.getDataList()
        this.pendingRequests--
      }
    },
    async getDataList() {
      if (!this.formData.id || this.disposed) return
      const version = ++this.requestVersion
      this.dataLoaded = false
      this.destroySortable()
      this.pendingRequests++
      try {
        const res = await findDataDetails({ equipmentId: this.formData.id })
        if (this.disposed || version !== this.requestVersion) return
        // 后端查询错误时可能仍返回 code=200、return=null，必须显式核验列表。
        if (!Array.isArray(res?.return?.data)) throw new Error('加载设备数据表失败：接口未返回有效的数据列表，请重新查询')
        const list = res.return.data
        const labelIds = new Set(this.contrLabelList.map(item => item.labelId))
        const labelCounts = new Map()
        list.forEach(item => labelCounts.set(item.labelId, (labelCounts.get(item.labelId) || 0) + 1))
        this.allTableData = list.map(item => {
          const content = Array.isArray(item.contentDataDetails) ? clone(item.contentDataDetails) : []
          const settings = content[0] || {}
          return {
            ...item,
            contentDataDetails: content,
            limitValue: settings.limitValue,
            unit: settings.unit,
            decimalPlaces: settings.decimalPlaces ?? 2, // 0 位小数是合法设置，不能用真假值回退。
            color: settings.color || DEFAULT_COLOR,
            labelUnit: settings.labelUnit,
            remark: settings.remark,
            statusLightArr: (Array.isArray(settings.statusLightArr) ? settings.statusLightArr : []).map(status => ({
              ...status,
              name: this.judgeList.find(judge => judge.key === status.key)?.name || status.key,
              isMsg: !checkIsColor(status.color)
            })),
            hasLabel: !this.labelsLoaded || labelIds.has(item.labelId),
            repeatLabel: labelCounts.get(item.labelId) > 1
          }
        }).sort((a, b) => Number(a.sortNum) - Number(b.sortNum))
        // 原查询接口按名称精确匹配，保留显示行为，同时保存未筛选的序号集合。
        this.tableData = this.searchMsg ? this.allTableData.filter(item => item.name === this.searchMsg) : [...this.allTableData]
        this.dataPointList = this.allTableData.map(item => ({ key: item.id, name: item.name }))
        this.selectedTableData = []
        this.bannerList[0].info = String(this.tableData.length)
        await this.initSortNum()
        if (this.disposed || version !== this.requestVersion) return
        this.dataLoaded = true
        await this.rowDrop()
      } catch (error) {
        if (!this.disposed && version === this.requestVersion) {
          this.dataLoaded = false
          this.destroySortable()
          this.showError(error, '读取设备数据表失败')
        }
      } finally {
        this.pendingRequests--
      }
    },
    async initSortNum() {
      // 与原版兼容：仅全部没有序号时初始化，不在搜索结果上重排原有数据。
      if (!this.allTableData.length || this.allTableData.some(row => Number(row.sortNum) > 0)) return
      for (const [index, row] of this.allTableData.entries()) {
        row.sortNum = index + 1
        await modifyDataDetails(row)
      }
    },
    handleSelectionChange(rows) { this.selectedTableData = rows },
    clickBannerBtn(item) {
      if (this.loading) return
      if (item.buttonName === '添加数据') this.add()
      else if (item.buttonName === '批量删除') this.batchRemove()
    },
    add() { return this.openForm('add') },
    modify(row) { return this.openForm('modify', row) },
    setMsg(row) { return this.openForm('setMsg', row) },
    async openForm(status, row) {
      if (this.saving || this.loading) return
      if (!this.requireDataLoaded()) return
      this.dialogStatus = status
      this.selectStatusLightIndex = ''
      this.pendingRequests++
      try {
        await this.resetForm(row)
        this.showDrawer = true
        await this.$nextTick()
        this.$refs.dataForm?.clearValidate()
      } catch (error) {
        this.showError(error, '读取数据点信息失败')
      } finally {
        this.pendingRequests--
      }
    },
    nextSortNum() {
      return Math.max(0, ...this.allTableData.map(row => Number(row.sortNum) || 0)) + 1
    },
    dataName(name) {
      const value = String(name || '')
      return this.isRem === '1' && value.includes('-') ? value.slice(value.indexOf('-') + 1) : value
    },
    async resetForm(row) {
      if (this.dialogStatus !== 'setMsg') {
        try {
          await this.getlabelList()
        } catch (error) {
          // 修改已有点无需依赖数据池在线，保留当前标签仍能编辑业务名称。
          if (this.dialogStatus !== 'modify') throw error
          this.labelList = []
          this.showError(error, '读取数据池标签失败')
        }
      }
      if (this.dialogStatus === 'add') {
        this.labelSelectType = '1'
        const labels = this.labelList.filter(item => item.key !== '全部')
        if (!labels.length) Message.warning('对应的数据流尚未添加数据')
        const selected = labels.find(item => item.key === this.lastAddLabelId) || this.labelList[0]
        this.form = {
          sortNum: this.nextSortNum(), name: this.dataName(selected?.name),
          equipmentId: this.formData.id, equipmentName: this.formData.equipmentName,
          labelId: selected?.key || '', labelName: selected?.name || '', labelIds: [], contentDataDetails: []
        }
      } else {
        this.labelSelectType = '1'
        this.form = {
          id: row.id, sortNum: row.sortNum, name: row.name,
          equipmentId: this.formData.id, equipmentName: this.formData.equipmentName,
          labelId: row.labelId, labelName: row.labelName,
          contentDataDetails: clone(row.contentDataDetails || [])
        }
        if (this.dialogStatus === 'modify') {
          this.labelList = this.labelList.filter(item => item.key !== '全部')
          if (!this.labelList.some(item => item.key === row.labelId)) this.labelList.push({ key: row.labelId, name: row.labelName })
        } else {
          Object.assign(this.form, {
            copyDataPointId: '', limitValue: row.limitValue ?? 1, unit: row.unit,
            decimalPlaces: row.decimalPlaces ?? 2, color: row.color || DEFAULT_COLOR,
            labelUnit: row.labelUnit || '', remark: row.remark || '', statusLightArr: clone(row.statusLightArr || [])
          })
        }
      }
    },
    async getlabelList() {
      if (!this.formData.apiKey) throw new Error('设备尚未绑定数据流，请先配置数据接入')
      const res = await getlabel({ ApiKey: this.formData.apiKey })
      const list = Array.isArray(res.return?.list) ? res.return.list : []
      this.labelList = [{ key: '全部', name: '全部' }, ...list.map(item => ({ ...item, key: item.labelId }))]
    },
    labelChange() {
      if (this.dialogStatus === 'setMsg' || this.labelSelectType !== '1') return
      const selected = this.labelList.find(item => item.key === this.form.labelId)
      if (selected) this.form.name = this.dataName(selected.name)
    },
    selectColor(event) { this.form.color = event.hex },
    addStatusLight() { this.form.statusLightArr.push({ key: '>', value: 0, color: '#000000' }) },
    statusLightSelColor(index) { this.selectStatusLightIndex = index },
    sureStatusLightSelColor(event) {
      const item = this.form.statusLightArr[this.selectStatusLightIndex]
      if (item) item.color = event.hex
    },
    removeStatusLight(index) {
      this.form.statusLightArr.splice(index, 1)
      if (this.selectStatusLightIndex === index) this.selectStatusLightIndex = ''
      else if (this.selectStatusLightIndex > index) this.selectStatusLightIndex--
    },
    copyDataPointChange() {
      const row = this.allTableData.find(item => item.id === this.form.copyDataPointId)
      if (!row) return
      // 深复制属性，取消抽屉时不会改变源数据点或当前表格中的状态灯。
      Object.assign(this.form, {
        contentDataDetails: clone(row.contentDataDetails || []), limitValue: row.limitValue ?? 1,
        unit: row.unit, decimalPlaces: row.decimalPlaces ?? 2, color: row.color || DEFAULT_COLOR,
        labelUnit: row.labelUnit || '', remark: row.remark || '', statusLightArr: clone(row.statusLightArr || [])
      })
    },
    async setData() {
      if (this.saving) return
      if (!this.requireDataLoaded()) return
      if (!await this.$refs.dataForm?.validate().catch(() => false)) return
      if (!this.requireDataLoaded()) return
      this.saving = true
      this.pendingRequests++
      let changed = false
      try {
        const payload = clone(this.form)
        payload.sortNum = Number(payload.sortNum)
        const selected = this.labelList.find(item => item.key === payload.labelId)
        if (selected) payload.labelName = selected.name
        if (this.dialogStatus === 'add') {
          const multiple = this.labelSelectType === '2' || payload.labelId === '全部'
          if (multiple) {
            const ids = this.labelSelectType === '2' ? payload.labelIds : ['全部']
            if (!ids.length) throw new Error('请选择数据池中的数据点')
            const existedIds = new Set(this.allTableData.map(item => item.labelId))
            const existedNames = new Set(this.allTableData.map(item => item.labelName))
            const labels = this.labelList.filter(item => item.key !== '全部' &&
              (ids.includes('全部') || ids.includes(item.key)) &&
              !existedIds.has(item.key) && !existedNames.has(item.name))
            if (!labels.length) {
              Message.warning('已全部添加，无需重复添加')
              return
            }
            let sortNum = this.nextSortNum()
            for (const label of labels) {
              await saveDataDetails({ ...payload, sortNum: sortNum++, labelId: label.key, labelName: label.name, name: this.dataName(label.name) })
              changed = true
            }
          } else {
            if (!payload.labelId || !payload.labelName) throw new Error('请选择数据池中的数据点')
            await saveDataDetails(payload)
            changed = true
            this.lastAddLabelId = payload.labelId
          }
          Message.success('成功添加')
        } else {
          if (this.dialogStatus === 'setMsg') {
            const content = Array.isArray(payload.contentDataDetails) ? payload.contentDataDetails : []
            content[0] = {
              ...(content[0] || {}), // 保留原有属性中本页面未编辑的字段。
              limitValue: payload.limitValue === '' ? '' : Number(payload.limitValue), unit: payload.unit,
              decimalPlaces: Number(payload.decimalPlaces), color: payload.color,
              labelUnit: payload.labelUnit || '', remark: payload.remark,
              statusLightArr: payload.statusLightArr.map(({ name, isMsg, ...status }) => status)
            }
            payload.contentDataDetails = content
          }
          await modifyDataDetails(payload)
          changed = true
          Message.success(this.dialogStatus === 'setMsg' ? '成功设置' : '成功修改')
        }
        this.showDrawer = false
      } catch (error) {
        this.showError(error, '保存设备数据点失败')
      } finally {
        // 批量操作中途失败也刷新已经完成的数据，避免重试时重复添加。
        if (changed) {
          this.$emit('data-change')
          await this.getDataList()
        }
        this.pendingRequests--
        this.saving = false
      }
    },
    async batchRemove() {
      if (!this.requireDataLoaded()) return
      if (!this.selectedTableData.length) {
        Message.warning('请选择表格中数据点')
        return
      }
      if (this.loading) return
      const rows = [...this.selectedTableData]
      this.pendingRequests++
      let changed = false
      try {
        for (const row of rows) {
          await deleteDataDetails({ id: row.id })
          changed = true
        }
        Message.success('删除成功')
      } catch (error) {
        this.showError(error, '批量删除失败')
      } finally {
        if (changed) {
          this.$emit('data-change')
          await this.getDataList()
        }
        this.pendingRequests--
      }
    },
    async remove(row) {
      if (this.loading) return
      if (!this.requireDataLoaded()) return
      const sure = await meAlert(this, { title: '确认删除', content: `此操作将永久删除${row.name}，是否继续？` })
      if (!sure || !this.requireDataLoaded()) return
      this.pendingRequests++
      try {
        // 通用请求层已验证 code=200，不能沿用原版的 code=0 判断。
        await deleteDataDetails({ id: row.id })
        Message.success(`${row.name}删除成功`)
        this.$emit('data-change')
        await this.getDataList()
      } catch (error) {
        this.showError(error, '删除数据点失败')
      } finally {
        this.pendingRequests--
      }
    },
    search() { this.getDataList() }
  }
}
</script>

<style lang="stylus" scoped>
.content {
  padding-top 20px
  .BannerCss {
    margin 0 2%
  }
}
</style>
