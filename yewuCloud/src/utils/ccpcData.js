export const processDrawList = [ // 工艺图模板列表
  { key: '0', name: '默认画面' }
]

export const draw3DList = [ // 3D图模板列表
  { key: '1', name: '3D测试图', url: 'https://www.thingjs.com/pp/4f66efec2bdb3aad33d8b91f' }
]

export const judgeList = [ // 判断列表
  { key: '>', name: '大于' },
  { key: '=', name: '等于' },
  { key: '<', name: '小于' },
  { key: '>=', name: '大于等于' },
  { key: '<=', name: '小于等于' }
]

export const trigTypeList = [ // 触发器类型列表
  { key: 'realtime', name: '实时触发器' },
  { key: 'timing', name: '定时触发器' }
]

import { isNumber, formatDateString } from '@/utils/utils.js'

export function setChartValue(params) { // 设置曲线图要显示的数据值
  const { times, values, status, item, index, decimalPlaces } = params
  let chartPointData = {} // 曲线图点数据
  let value = item
  if (status[index] < 0) {
    if (index - 1 >= 0) {
      value = values[index - 1]
    }
    chartPointData = {
      time: times[index],
      date: formatDateString(times[index], 'timeStmapReset'),
      value,
      itemStyle: {
        color: 'red' // 异常数据的点颜色
      }
    }
  } else {
    if (isNumber(item)) {
      value = parseFloat(item).toFixed(decimalPlaces)
    }
    chartPointData = {
      time: times[index],
      date: formatDateString(times[index], 'timeStmapReset'),
      value
    }
  }
  return chartPointData
}

export const pushWayList = [ // 推送方式
  { key: 1, name: '短信' }
]

export const alarmStageList = [ // 告警等级列表
  { key: 1, name: '一级' },
  { key: 2, name: '二级' },
  { key: 3, name: '三级' }
]
