<template>
  <div class="bannerCss">
    <div
      v-for="(item, index) in list"
      :key="index"
      :class="`bannerItemCss--${item.style}`"
      :style="{'float': item.float ? item.float : 'left'}"
    >
      <div v-if="item.style === 'infoNum'" class="infoNumCss marginCss">
        <div v-if="item.icon" class="infoNumIconCss">
          <img :src="item.icon">
        </div>
        <div
          v-if="item.title"
          class="infoNumTitleCss"
          :style="{'font-size': item.titleFontSize ? item.titleFontSize : '0.72916vw'}"
        >{{ item.title }}</div>
        <div v-if="item.num || item.num === 0" class="numUnitAndInfoCss">
          <div
            class="numCss"
            :style="{'font-size': item.numFontSize ? item.numFontSize : '1.25vw'}"
          >{{ item.num }}</div>
          <div
            v-if="item.unit"
            class="unitCss"
            :style="{'font-size': item.unitFontSize ? item.unitFontSize : '0.72916vw'}"
          > {{ item.unit }} </div>
        </div>
        <div v-if="item.info" class="numUnitAndInfoCss">
          <div
            :id="item.info"
            class="infoCss"
            :data-clipboard-text="item.info"
            :style="{color: item.color}"
            @contextmenu.prevent="rightMenu(item)"
          >
            {{ item.info }}
          </div>
          <input v-if="item.allowCopy" :id="item.info + 'input'" type="text" value="" style="position: absolute;top: 0;left: 0;opacity: 0;z-index: -10;">
        </div>
      </div>
      <div
        v-else-if="item.style === 'input'"
        class="marginCss"
        :style="{width:item.width?item.width:''}"
      >
        <el-input
          v-if="item.onKeypress === '正整数'"
          v-model="item.input"
          :size="item.size"
          :placeholder="item.placeholder"
          type="number"
          :min="item.min"
          :max="item.max"
          @keypress="onPositiveIntegerKeypress"
          @change="inputChange(item)"
          @keyup.enter="inputKeyupEnter(item)"
        />
        <el-input
          v-else
          v-model="item.input"
          :size="item.size"
          :placeholder="item.placeholder"
          :type="item.inputType"
          :min="item.min"
          :max="item.max"
          @change="inputChange(item)"
          @keyup.enter="inputKeyupEnter(item)"
        />
      </div>
      <div
        v-else-if="item.style === 'select'"
        class="marginCss"
        :style="{width:item.width?item.width:'8vw'}"
      >
        <el-select
          v-model="item.value"
          :size="item.size"
          :filterable="item.filterable !== false"
          :teleported="true"
          :placeholder="item.placeholder"
          @change="selectChange(item)"
        >
          <el-option
            v-for="data in item.list"
            :key="String(data.key)"
            :label="data.name"
            :value="data.key"
          />
        </el-select>
      </div>
      <div v-else-if="item.style === 'multipleSelect'" class="marginCss">
        <el-select
          v-model="item.valueArr"
          multiple
          filterable
          allow-create
          collapse-tags
          default-first-option
          clearable
          :reserve-keyword="true"
          :placeholder="item.placeholder"
          @change="selectChange(item)"
        >
          <el-option
            v-for="data in item.list"
            :key="data.key"
            :label="data.name"
            :value="data.key"
          />
        </el-select>
      </div>
      <div v-else-if="item.style === 'dropdown'" class="marginCss">
        <el-dropdown @command="dropdownClick">
          <el-button :type="item.type?item.type:'default'" :size="item.size">
            {{ item.buttonName }}
            <el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-for="(data, i) in item.list" :key="i" :command="data.key">{{ data.name }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
      <div v-else-if="item.style === 'button'" class="marginCss">
        <el-button
          :type="resolveButtonType(item)"
          :icon="resolveIcon(item.icon)"
          :size="item.size"
          @click="clickButton(item)"
        >
          {{ item.buttonName }}
        </el-button>
      </div>
      <div v-else-if="item.style === 'selectTimeSlot'" class="marginCss">
        <el-date-picker
          v-model="item.timeDateArr"
          :style="{width: item.width ? item.width : '20vw'}"
          :type="item.datePickerType ? item.datePickerType : 'datetimerange'"
          range-separator="至"
          :start-placeholder="item.startPlaceholder ? item.startPlaceholder: '开始日期'"
          :end-placeholder="item.endPlaceholder ? item.endPlaceholder: '结束日期'"
          :size="item.size ? item.size : 'small'"
          @change="timeChange(item)"
        />
      </div>
      <div v-else-if="item.style === 'selectMonthDate'" class="marginCss">
        <el-date-picker
          v-model="item.value"
          :type="item.datePickerType ? item.datePickerType : 'datetime'"
          :placeholder="item.placeholder"
          @change="timeChange(item)"
        />
      </div>
      <div v-else-if="item.style === 'stateSwitchBut'" class="switchContentCss marginCss">
        <div class="switchTitleCss">{{ item.title }}</div>
        <div v-if="item.state" class="stateImgCss">
          <img v-if="item.state === 'close'" :src="stateIcons.close">
          <img v-if="item.state === 'open'" :src="stateIcons.open">
          <img v-if="item.state === 'wait'" :src="stateIcons.wait">
        </div>
        <el-switch
          v-model="item.value"
          @change="switchChange($event, item)"
        />
      </div>
      <div v-else-if="item.style === 'downloadUrl'" class="downloadUrlCss cursorPointer">
        <div @click="downloadFile(item)">{{ item.downloadName }}</div>
      </div>
      <div v-else-if="item.style === 'cascader'" class="marginCss">
        <el-cascader
          v-model="item.valueArr"
          :options="item.options"
          :props="item.props ? item.props : { multiple: true }"
          :placeholder="item.placeholder"
          filterable
          collapse-tags
          clearable
          @change="cascaderChange(item)"
          @expand-change="cascaderExpandChange($event, item)"
        />
      </div>
    </div>
    <br style="clear: both">
  </div>
</template>

<script>
import { ArrowDown } from '@element-plus/icons-vue'
import closeSvg from './close.svg'
import openSvg from './open.svg'
import waitSvg from './wait.svg'
import { formatDateString, getTime, dateConvStamp } from '@/utils/utils.js'
import { resolveLegacyIcon } from '@/utils/legacy-element-icons'

export default {
  components: {
    ArrowDown
  },
  props: {
    list: {
      type: Array,
      default: () => {
        return [
          { style: 'infoNum', title: '标签数', num: '', unit: '个', titleFontSize: '0.72916vw', numFontSize: '1.25vw', unitFontSize: '0.72916vw' },
          { style: 'infoNum', title: '流名称', info: '', color: '#D757AC', allowCopy: false }, // allowCopy 是否可以复制
          { style: 'input', placeholder: '请输入站址名称', input: '' },
          { style: 'select', placeholder: '', value: '', list: [
            { key: '1', name: 'A' },
            { key: '2', name: 'B' }
          ] },
          { style: 'multipleSelect', placeholder: '', valueArr: [], list: [
            { key: '1', name: 'A' },
            { key: '2', name: 'B' },
            { key: '3', name: 'C' }
          ] },
          { style: 'cascader', placeholder: '', valueArr: [],
            props: { multiple: true, checkStrictly: true },
            options: [{ value: 'zhinan', label: '指南',
              children: [{ value: 'shejiyuanze', label: '设计原则',
                children: [{ value: 'yizhi', label: '一致' }, { value: 'fankui', label: '反馈' }]
              }, { value: 'daohang', label: '导航',
                children: [{ value: 'cexiangdaohang', label: '侧向导航' }, { value: 'dingbudaohang', label: '顶部导航' }]
              }]
            }, { value: 'zujian', label: '组件',
              children: [{ value: 'basic', label: 'Basic',
                children: [{ value: 'layout', label: 'Layout 布局' }, { value: 'color', label: 'Color 色彩' }]
              }, { value: 'form', label: 'Form',
                children: [{ value: 'radio', label: 'Radio 单选框' }, { value: 'checkbox', label: 'Checkbox 多选框' }]
              }]
            }]
          },
          { style: 'dropdown', buttonName: '更多操作', type: 'default', list: [
            { key: '客户地图', name: '客户地图' },
            { key: 'Excel导入', name: 'Excel导入' },
            { key: 'Excel导出', name: 'Excel导出' },
            { key: '批量删除', name: '批量删除' }
          ], float: 'right' },
          { style: 'stateSwitchBut', title: '选择', state: false, value: true },
          { style: 'button', buttonName: '添加站址', type: 'primary', icon: 'el-icon-plus', float: 'right' },
          { style: 'selectTimeSlot', name: '选择时间段', timeDateArr: [],
            startPlaceholder: '开始日期', endPlaceholder: '结束日期', datePickerType: 'datetimerange'
          }, // 可取值: datetimerange/daterange/monthrange
          { style: 'selectMonthDate', placeholder: '选择时间', value: '', datePickerType: 'datetime' }, // datePickerType 可取值: datetime/date/month/year
          { style: 'downloadUrl', downloadName: '北片区动态成本填报模板下载', float: 'right' }
        ]
      }
    }
  },
  emits: [
    'clickButton',
    'inputChange',
    'inputKeyupEnter',
    'selectChange',
    'dropdownClick',
    'timeChange',
    'cascaderChange',
    'cascaderExpandChange',
    'switchChange',
    'downloadFile'
  ],
  data() {
    return {
      stateIcons: {
        close: closeSvg,
        open: openSvg,
        wait: waitSvg
      }
    }
  },
  methods: {
    resolveIcon(icon) {
      return resolveLegacyIcon(icon)
    },
    resolveButtonType(item) {
      const name = String(item.buttonName || item.name || '')
      if (name.includes('删除')) {
        return 'danger'
      }
      return item.type ? item.type : 'default'
    },
    onPositiveIntegerKeypress(event) {
      const charCode = event.keyCode || event.which
      if (charCode === 8) return
      if (!/[\d]/.test(String.fromCharCode(charCode))) {
        event.preventDefault()
      }
    },
    rightMenu(item) {
      this.copy(item)
    },
    clickButton(item) { // 点击按钮
      this.$emit('clickButton', item)
    },
    inputChange(item) { // 输入框内容改变时
      this.$emit('inputChange', item)
    },
    inputKeyupEnter(item) {
      this.$emit('inputKeyupEnter', item)
    },
    selectChange(item) { // 选择选项（nextTick 避免与下拉关闭时的布局观测冲突）
      this.$nextTick(() => {
        this.$emit('selectChange', item)
      })
    },
    dropdownClick(e) {
      this.$emit('dropdownClick', { dropdownName: e })
    },
    timeChange(e) {
      const nowDate = getTime()
      const nowTimeStamp = dateConvStamp(nowDate) / 1000 // 当前时间的时间戳
      if (e.style === 'selectTimeSlot') { // 选择时间段
        if (e.timeDateArr && e.timeDateArr.length !== 0) {
          let startDate = formatDateString(e.timeDateArr[0], 'formatDate')
          const startDateStamp = dateConvStamp(startDate) / 1000 // 开始时间的时间戳
          if (nowTimeStamp < startDateStamp) { // 如果当前时间小于开始时间，则将开始时间设置为当前时间
            startDate = nowDate.slice(0, 10) + ' 00:00:00'
          }
          e.timeDateArr[0] = startDate
          let endDate = formatDateString(e.timeDateArr[1], 'formatDate')
          if (endDate.slice(11) === '00:00:00') {
            endDate = endDate.slice(0, 11) + '23:59:59'
          }
          const endTimeStamp = dateConvStamp(endDate) / 1000 // 结束时间的时间戳
          if (nowTimeStamp < endTimeStamp) { // 如果当前时间小于结束时间，则将结束时间设置为当前时间
            endDate = nowDate
          }
          e.timeDateArr[1] = endDate
        } else {
          e.timeDateArr = []
        }
      } else if (e.style === 'selectMonthDate') { // 选择时间
        if (e.value && e.value !== '') {
          let selectDate = formatDateString(e.value, 'formatDate')
          const selectTimeStamp = dateConvStamp(selectDate) / 1000 // 选择时间的时间戳
          if (nowTimeStamp < selectTimeStamp) { // 如果当前时间小于选择时间，则将选择时间设置为当前时间
            selectDate = nowDate
          }
          e.value = selectDate
        } else {
          e.value = ''
        }
      }
      this.$emit('timeChange', e)
    },
    cascaderChange(item) { // 级联选择器改变时
      this.$emit('cascaderChange', item)
    },
    cascaderExpandChange(e, item) { // 级联选择器展开改变时
      this.$emit('cascaderExpandChange', {
        value: e,
        item
      })
    },
    switchChange(e, item) {
      this.$emit('switchChange', {
        state: e,
        item
      })
    },
    copy(item) {
      if (item.allowCopy) {
        const clickContent = document.getElementById(item.info).innerText // 获取点击的值
        if (navigator.clipboard && window.isSecureContext) {
          const inputElement = document.getElementById(item.info + 'input') // 获取要赋值的input的元素
          inputElement.value = clickContent // 给input框赋值
          inputElement.select() // 选中input框的内容
          navigator.clipboard.writeText(clickContent) // 执行浏览器复制命令
          this.$notify({ // 提示已复制
            title: clickContent,
            message: `已复制`,
            type: 'success',
            duration: 3 * 1000
          })
        } else {
          // 创建text area
          const textArea = document.createElement('textarea')
          textArea.value = clickContent
          // 使text area不在viewport，同时设置不可见
          document.body.appendChild(textArea)
          textArea.focus()
          textArea.select()
          this.$notify({ // 提示已复制
            title: clickContent,
            message: `已复制`,
            type: 'success',
            duration: 3 * 1000
          })
        }
      }
    },
    downloadFile(item) { // 下载文件
      this.$emit('downloadFile', item)
    }
  }
}
</script>

<style lang="stylus" scoped>
searchContentHeight = 50px
infoNumHeight = 38px

.bannerCss {
  // 隔离筛选项浮动，窗口变化时多行筛选不会挤压下方内容。
  display flow-root
  padding 0 7px 0 3px
  width 100%
  line-height searchContentHeight
  background-color #fff
  .infoNumCss {
    padding-top 6px
    height searchContentHeight
    .infoNumIconCss {
      float left
      width infoNumHeight
      height infoNumHeight
      img {
        width 100%
        height 100%
      }
    }
    .infoNumTitleCss {
      float left
      width 4.583vw
      line-height infoNumHeight
      text-align center
    }
    .numUnitAndInfoCss {
      float left
      line-height infoNumHeight
      padding-right 0.72916vw
      border-right 1px solid #EAEDF7
      .numCss {
        float left
        font-weight bold
        color #369AFE
      }
      .unitCss {
        float left
        margin 0 0.7vw 0 0.78125vw
      }
      .infoCss {
        float left
        font-size 1vw
        font-weight bold
        color #369AFE
      }
    }
  }
  .switchContentCss {
    float left
    height searchContentHeight
    padding-right 0.72916vw
    border-right 1px solid #EAEDF7
    .switchTitleCss {
      float left
      width 4.583vw
      text-align center
      font-size 0.72916vw
    }
    .stateImgCss {
      float left
      margin-right 5px
      width 40px
      height 40px
      img {
        width 100%
        height 100%
      }
    }
  }
  .downloadUrlCss {
    padding 0 0.5vw
    text-decoration underline
    color rgba(0,142,253, 1)
  }
  .downloadUrlCss:hover {
    color rgba(0,142,253, 0.5)
  }
  .marginCss {
    margin 0 0.1vw
  }
  > div {
    max-width 100%
  }
}

// 仅在可用宽度较小时设置控件的可读下限，仍按原来的浮动顺序换行。
@media (max-width: 1280px) {
  .bannerCss {
    .bannerItemCss--input, .bannerItemCss--select, .bannerItemCss--multipleSelect, .bannerItemCss--cascader {
      min-width unquote('min(144px, 100%)')
      .marginCss {
        min-width 100%
      }
    }
    .bannerItemCss--selectTimeSlot, .bannerItemCss--selectMonthDate {
      min-width unquote('min(320px, 100%)')
      .marginCss {
        min-width 100%
      }
    }
    .marginCss {
      max-width 100%
    }
    :deep(.el-input), :deep(.el-select), :deep(.el-cascader) {
      min-width unquote('min(144px, 100%)')
      max-width 100%
    }
    :deep(.el-date-editor) {
      min-width unquote('min(320px, 100%)')
      max-width 100%
    }
    .infoNumCss {
      .infoNumTitleCss {
        width auto
        padding 0 8px
        font-size 13px !important
      }
      .numUnitAndInfoCss {
        .numCss {
          font-size 18px !important
        }
        .unitCss, .infoCss {
          font-size 13px !important
        }
      }
    }
    .switchContentCss .switchTitleCss {
      width auto
      padding 0 8px
      font-size 13px
    }
  }
}
</style>
