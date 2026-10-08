export function getTime(timestamp = '') { // 获取时间或将时间戳转化为时间格式（日期格式）后返回,timestamp为时间戳
  let date
  if (timestamp === '') {
    date = new Date()
  } else {
    timestamp = parseInt(timestamp)
    if (timestamp.toString().length === 10) { // timestamp 为10位时，需要*1000
      date = new Date(timestamp * 1000)
    } else {
      date = new Date(timestamp)
    }
  }
  const year = date.getFullYear()
  let month = date.getMonth() + 1
  month = dateZeroFill(month)
  let day = date.getDate()
  day = dateZeroFill(day)
  let hour = date.getHours()
  hour = dateZeroFill(hour)
  let minute = date.getMinutes()
  minute = dateZeroFill(minute)
  let second = date.getSeconds()
  second = dateZeroFill(second)
  const busUpdateTime = year + '-' + month + '-' + day + ' ' + hour + ':' + minute + ':' + second
  return busUpdateTime
}

export function formatDate(date, monthAdd = true) { // 将ISO8601‌国际标准日期转化为时间对象格式； monthAdd 为true是代表转换后需要月份+1
  date = new Date(date)
  const year = date.getFullYear()
  let month = date.getMonth()
  if (monthAdd) {
    month = date.getMonth() + 1
  }
  month = dateZeroFill(month)
  let day = date.getDate()
  day = dateZeroFill(day)
  let hour = date.getHours()
  hour = dateZeroFill(hour)
  let minute = date.getMinutes()
  minute = dateZeroFill(minute)
  let second = date.getSeconds()
  second = dateZeroFill(second)
  date = {
    year,
    month,
    day,
    hour,
    minute,
    second
  }
  return date
}

export function timeStmapReset(timestamp) { // 将时间戳转化时间对象
  timestamp = parseInt(timestamp)
  let date = new Date(timestamp)
  if (timestamp.toString().length === 10) { // 时间戳为10位需*1000，时间戳为13位的话不需乘1000
    date = new Date(timestamp * 1000)
  }
  const year = date.getFullYear()
  let month = date.getMonth() + 1 < 10 ? '0' + (date.getMonth() + 1) : date.getMonth() + 1
  month = dateZeroFill(month)
  let day = date.getDate()
  day = dateZeroFill(day)
  let hour = date.getHours()
  hour = dateZeroFill(hour)
  let minute = date.getMinutes()
  minute = dateZeroFill(minute)
  let second = date.getSeconds()
  second = dateZeroFill(second)
  date = {
    year,
    month,
    day,
    hour,
    minute,
    second
  }
  return date
}

export function formatDateString(date, type, monthAdd) { // 将ISO8601‌国际标准日期或时间戳转化为日期字符串 返回格式为:'2018-05-28 23:24:12'
  let dateObj = null
  if (type === 'formatDate') { // ISO8601‌国际标准日期
    dateObj = formatDate(date, monthAdd)
  } else if (type === 'timeStmapReset') { // 时间戳
    dateObj = timeStmapReset(date)
  } else {
    console.log('时间转化方法缺少type参数')
  }
  if (dateObj) {
    return dateObj.year + '-' + dateObj.month + '-' + dateObj.day + ' ' + dateObj.hour + ':' + dateObj.minute + ':' + dateObj.second
  }
}

export function format0800CSTDateString(dateString) { // 将 尾部 '+0800 CST' 转化为日期字符串 返回格式为:'2018-05-28 23:24:12'
  const regex = /(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})/ // 使用正则表达式提取日期和时间部分
  const match = dateString.match(regex)
  if (match) {
    return match[1] // 提取的日期时间部分
  } else {
    return '日期时间格式不匹配'
  }
}

function dateZeroFill(time) { // time为月或日、时、分、秒，若time只有一位则在前面补零
  return time.toString().length === 1 ? '0' + time.toString() : time
}

export function dateConvStamp(dateTime) { // 将日期转化为时间戳 dateTime格式：'2018-05-28 23:24:12'
  // 手机系统不支持2019-04-29这种格式因此需要.replace(/-/g, '/')来转化为2019/04/29这种格式
  return Date.parse(new Date(dateTime.replace(/-/g, '/')))
}

export function dateResetObj(dateTime) { // 将日期转化为时间对象 dateTime格式：'2018-05-28 23:24:12'
  return timeStmapReset(dateConvStamp(dateTime))
}

export function setGlwzTime(time) { // 将格林威治时间转化为日期字符串
  var date = time.substr(0, 10) // 年月日
  var hours = time.substring(11, 13)
  var minutes = time.substring(14, 16)
  var seconds = time.substring(17, 19)
  var timeFlag = date + ' ' + hours + ':' + minutes + ':' + seconds
  timeFlag = timeFlag.replace(/-/g, '/')
  timeFlag = new Date(timeFlag)
  timeFlag = new Date(timeFlag.getTime() + 8 * 3600 * 1000)
  timeFlag = timeFlag.getFullYear() +
    '-' +
    (timeFlag.getMonth() + 1 < 10 ? '0' + (timeFlag.getMonth() + 1) : timeFlag.getMonth() + 1) +
    '-' +
    (timeFlag.getDate() < 10 ? '0' + timeFlag.getDate() : timeFlag.getDate()) +
    ' ' +
    (timeFlag.getHours() < 10 ? '0' + timeFlag.getHours() : timeFlag.getHours()) +
    ':' +
    (timeFlag.getMinutes() < 10 ? '0' + timeFlag.getMinutes() : timeFlag.getMinutes()) +
    ':' +
    (timeFlag.getSeconds() < 10 ? '0' + timeFlag.getSeconds() : timeFlag.getSeconds())
  return timeFlag
}

export function convertToCST(dateTime) { // 将日期字符串转为ISO8601‌国际标准日期，字符串 dateTime格式：'2018-05-28 23:24:12'
  const date = new Date(dateTime)
  return date.toISOString()
}

export function dateAddOrSubtract(addOrSubtract, dateTime, level, type = 'obj') { // 日期时间后退或前进
  /*
    addOrSubtract === -1时，代表日期时间后退。addOrSubtract === 1时，代表日期时间前进。
    dateTime 为日期，格式为：'2018-05-28 23:24:12'
    level 为日期后退或前进的级别，可取 date 、 month 、 year 分别代表日、月、年
    type 为返回的时间类型，可取 date 、 obj 、 timeStmap ，分别代表日期，对象，时间戳
  */
  const dateObj = dateResetObj(dateTime) // 日期转化为对象
  const isLeapYear = dateIsLeapYear(dateObj.year)
  if (addOrSubtract === -1) { // 时间后退
    if (level === 'date') { // 日统计
      dateObj.day--
      if (dateObj.day === 0) {
        dateObj.month--
        if (dateObj.month === 0 ||
          dateObj.month === 1 ||
          dateObj.month === 3 ||
          dateObj.month === 5 ||
          dateObj.month === 7 ||
          dateObj.month === 8 ||
          dateObj.month === 10) { // 大月
          dateObj.day = 31
        } else if (dateObj.month === 2) { // 2月
          if (isLeapYear) { // 是闰年
            dateObj.day = 29
          } else {
            dateObj.day = 28
          }
        } else { // 小月
          dateObj.day = 30
        }
        if (dateObj.month === 0) {
          dateObj.year--
          dateObj.month = 12
        }
      }
    } else if (level === 'month') {
      dateObj.month--
      if (dateObj.month === 0) {
        dateObj.year--
        dateObj.month = 12
      }
    } else if (level === 'year') {
      dateObj.year--
    }
  } else if (addOrSubtract === 1) { // 时间前进
    if (level === 'date') { // 日统计
      dateObj.day++
      if (dateObj.month === 1 ||
        dateObj.month === 3 ||
        dateObj.month === 5 ||
        dateObj.month === 7 ||
        dateObj.month === 8 ||
        dateObj.month === 10 ||
        dateObj.month === 12) { // 大月
        if (dateObj.day > 31) {
          dateObj.month++
          dateObj.day = 1
        }
      } else if (dateObj.month === 2) { // 2月
        if (isLeapYear && dateObj.day > 29) { // 是闰年
          dateObj.month++
          dateObj.day = 1
        } else if (isLeapYear && dateObj.day > 28) {
          dateObj.month++
          dateObj.day = 1
        }
      } else { // 小月
        if (dateObj.day > 30) {
          dateObj.month++
          dateObj.day = 1
        }
      }
      if (dateObj.month > 12) {
        dateObj.year++
        dateObj.month = 1
      }
    } else if (level === 'month') {
      dateObj.month++
      if (dateObj.month > 12) {
        dateObj.year++
        dateObj.month = 1
      }
    } else if (level === 'year') {
      dateObj.year++
    }
  }
  dateObj.day = dateZeroFill(dateObj.day)
  dateObj.month = dateZeroFill(dateObj.month)
  if (type === 'obj') {
    dateTime = dateObj
  } else if (type === 'date') {
    dateTime = dateObj.year + '-' + dateObj.month + '-' + dateObj.day + ' ' + dateObj.hour + ':' + dateObj.minute + ':' + dateObj.second
  } else if (type === 'timeStmap') {
    dateTime = dateConvStamp(dateObj.year + '-' + dateObj.month + '-' + dateObj.day + ' ' + dateObj.hour + ':' + dateObj.minute + ':' + dateObj.second)
  }
  return dateTime
}

export function getRecentlyDate(number) { // 获取最近 number 天日期列表
  const nowDate = getTime() // 当前时间
  const dateList = [nowDate]
  for (let i = 0; i < number - 1; i++) {
    const nowEndDate = dateList[0] // 最后一个日期时间
    const endDate = dateAddOrSubtract(-1, nowEndDate, 'date', 'date') // 时间后退一天
    dateList.splice(0, 0, endDate) // 在数组开头插入元素
  }
  return dateList
}
function dateIsLeapYear(year) { // 判断 year 是否是闰年
  if (year % 4 === 0 && year % 100 !== 0 || year % 400 === 0) {
    return true
  } else {
    return false
  }
}

export function getMonthLength(year, month) { // 判断某年某月有多少天
  return new Date(year, month, 0).getDate()
}
export function getYearLength(year) { // 判断某年有多少天
  let leapYear = false; let sum_day = 0; const month_arr = [4, 6, 9, 11]
  if (year % 100 === 0) { // 年份是整百
    leapYear = year % 400 === 0
  } else {
    leapYear = year % 4 === 0
  }
  // 下面计算每个月的天数
  for (let i = 1; i < 13; i++) {
    if (i === 2) {
      sum_day += leapYear ? 29 : 28
    } else if (month_arr.includes(i)) {
      sum_day += 30
    } else {
      sum_day += 31
    }
  }
  return sum_day
}

export function setNumerOrStrNum(params) { // 设置数字型字符串或数字数据
  let value = params.value
  const fixed = params.fixed || params.fixed === 0 ? params.fixed : 2 // 保留位数
  let isNumber = false // vakue 是否为数字类型或字符串型数字
  if (typeof value === 'number') {
    isNumber = true
  } else if (typeof value === 'string' && /^-?\d+(\.\d+)?$/.test(value)) { // 字符串（不包含科学计数法）
    value = parseFloat(value)
    isNumber = true
  }
  if (isNumber) {
    value = parseFloat(value.toFixed(fixed))
    return value
  } else {
    return false
  }
}

export function meAlert(This, alert) { // 参数This为调用该方法页面的指针,alert 包含弹框标题和内容
  return new Promise((resolve) => {
    const title = alert.title ? alert.title : '提示'
    This.$alert(alert.content, title, {
      confirmButtonText: '确定', // (无法设置取消键)
      callback: action => {
        resolve(action === 'confirm')
      }
    }).catch(() => {
      resolve(false)
    })
  })
}

export function meConfirm(This, confirm) { // 参数This为调用该方法页面的指针,confirm 包含弹框标题和内容
  return new Promise((resolve) => {
    const title = confirm.title ? confirm.title : '提示'
    This.$confirm(confirm.content, title, {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }).then(() => {
      resolve(true)
    }).catch(() => {
      resolve(false)
    })
  })
}

export function strToBinary(str) { // 将字符串转化为二进制的数据
  const result = []
  const list = str.split('')
  for (let i = 0; i < list.length; i++) {
    if (i !== 0) {
      result.push(' ') // 加空格，分割二进制
    }
    var item = list[i]
    var binaryStr = item.charCodeAt().toString(2) // 将字符串转化为二进制数据
    result.push(binaryStr)
  }
  return result.join('')
}

export function binaryToStr(str) { // 二进制转为字符串
  const result = []
  const list = str.split(' ') // 通过空格来分开二进制的字符
  for (let i = 0; i < list.length; i++) {
    const item = list[i]
    const asciiCode = parseInt(item, 2) // 转为asciicode 码
    const charValue = String.fromCharCode(asciiCode) // 转为文字
    result.push(charValue) // 添加到集合中
  }
  return result.join('') // 返回结果
}

export function strSetArr(str, spacer = '-') { // 将包含spacer的'值'的字符串,转化为以spacer的'值'为间隔的数组
  const num = (str.split(spacer)).length - 1
  const arr = []
  if (num === 0) {
    arr[0] = str
  } else {
    for (let i = 0; i < num; i++) {
      arr[i] = str.substring(0, str.indexOf(spacer))
      str = str.substring(arr[i].length + 1)
    }
    arr.push(str)
  }
  return arr
}

export function ipToArray(ip) { // ip字符串以.为间隔分割成数组，例如ip:10.10.1.1,分割后得到[10,10,1,1]
  let array = []
  if (ip !== '') {
    array = ip.split('.').map(item => parseInt(item))
    if (array.length !== 4) {
      array = ['', '', '', '']
    }
  }
  return array
}

export function goBack(This) { // 返回上一级页面并关闭当前页 This为页面的指针this
  This.$router.go(-1)// 返回
}

export function sleep(delay) { // sleep睡眠延时,delay单位为毫秒
  delay = parseInt(delay)
  const start = (new Date()).getTime()
  while ((new Date()).getTime() - start < delay) {
    continue
  }
}

export function getRSApassword(password, This) { // RSA 加密  参考地址：https://www.cnblogs.com/mmzuo-798/p/9872419.html
  // 公钥随部署密钥对一起注入，避免轮换后仍把旧公钥固化在前端源码中。
  const publicKey = String(import.meta.env.VITE_RSA_PUBLIC_KEY || '').trim()
  if (!publicKey) {
    throw new Error('未配置 VITE_RSA_PUBLIC_KEY')
  }
  const jse = new This.JsEncrypt()
  jse.setPublicKey(publicKey)
  return jse.encrypt(password)
}

export function isPosiInt(value) { // 判断是否是正整数
  const n = Math.floor(Number(value))
  return n !== Infinity && n >= 0
}

export function isNumber(val) { // 判断 val 是否是字符串数字 或 数字
  var regPos = /^\d+(\.\d+)?$/ // 非负浮点数
  var regNeg = /^(-(([0-9]+\.[0-9]*[1-9][0-9]*)|([0-9]*[1-9][0-9]*\.[0-9]+)|([0-9]*[1-9][0-9]*)))$/ // 负浮点数
  return regPos.test(val) || regNeg.test(val) || val === 0
}

export function listdupliRemoval(list, keyString) { // 对象数组去重,去重属性名为keyString的值
  const arr = list.reduce((prev, element) => {
    if (!prev.find(el => el[keyString] === element[keyString])) {
      prev.push(element)
    }
    return prev
  }, [])
  return arr
}

// listdupliRemoval 和 unique 是同样的方法
export function unique(list, key) { // 对象数组去重,去重属性名为key的值
  const res = new Map()
  return list.filter(item => !res.has(item[key]) && res.set(item[key], 1))
}

export function strReplaceAll(str, find, replace) { // 将字符串str 中所有的 find 字符 替换为 replace 字符
  const reg = new RegExp(find.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g')
  return str.replace(reg, replace)
}

export function randomString(e, type = 'all') { // 生成 e 长度的随机字符串, type为随机的类型
  e = e || 32
  let t = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz123456789'
  if (type === 'number') { // 数字
    t = '123456789'
  } else if (type === 'letters') { // 字母
    t = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'
  } else if (type === 'capitalLetters') { // 大写字母
    t = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'
  } else if (type === 'lowercaseLetters') { // 小写字母
    t = 'abcdefghijklmnopqrstuvwxyz'
  }

  const a = t.length
  let n = ''
  for (let i = 0; i < e; i++) n += t.charAt(Math.floor(Math.random() * a))
  return n
}

export function setScreen(id, fullscreen) { // 设置后改变 id 对应的容器全屏 fullscreen 的状态
  const element = document.getElementById(id)
  if (!fullscreen) {
    if (document.exitFullscreen) {
      document.exitFullscreen()
    } else if (document.webkitCancelFullScreen) {
      document.webkitCancelFullScreen()
    } else if (document.mozCancelFullScreen) {
      document.mozCancelFullScreen()
    } else if (document.msExitFullscreen) {
      document.msExitFullscreen()
    }
  } else {
    if (element.requestFullscreen) {
      element.requestFullscreen()
    } else if (element.webkitRequestFullScreen) {
      element.webkitRequestFullScreen()
    } else if (element.mozRequestFullScreen) {
      element.mozRequestFullScreen()
    } else if (element.msRequestFullscreen) {
      element.msRequestFullscreen()
    }
  }
  fullscreen = !fullscreen
  return fullscreen
}

export function checkIsColor(color) { // 校验 bgVal 是否是颜色的值
  const re1 = /^#([0-9a-f]{6}|[0-9a-f]{3})$/i
  const re2 = /^rgb\(([0-9]|[0-9][0-9]|25[0-5]|2[0-4][0-9]|[0-1][0-9][0-9])\,([0-9]|[0-9][0-9]|25[0-5]|2[0-4][0-9]|[0-1][0-9][0-9])\,([0-9]|[0-9][0-9]|25[0-5]|2[0-4][0-9]|[0-1][0-9][0-9])\)$/i
  const re3 = /^rgba\(([0-9]|[0-9][0-9]|25[0-5]|2[0-4][0-9]|[0-1][0-9][0-9])\,([0-9]|[0-9][0-9]|25[0-5]|2[0-4][0-9]|[0-1][0-9][0-9])\,([0-9]|[0-9][0-9]|25[0-5]|2[0-4][0-9]|[0-1][0-9][0-9])\,(1|1.0|0.[0-9])\)$/i
  return re2.test(color) || re1.test(color) || re3.test(color)
}

// 中文转英文方法 start
import PinYin from './ChineseHelperStr.js' // 引入字符编码（JSON）
export const loadPinYinInit = {} // 定义方法
loadPinYinInit.GetPinyinStr = () => { // 获取所有编码
  return PinYin
}
loadPinYinInit.ConvertPinyin = (l1) => { // 汉字转拼音
  var l2 = l1.length
  var I1 = ''
  var reg = new RegExp('[a-zA-Z0-9]')
  for (var i = 0; i < l2; i++) {
    var val = l1.substr(i, 1)
    var name = loadPinYinInit.arraySearch(val, PinYin)
    if (reg.test(val)) {
      I1 += val
    } else if (name !== false) {
      I1 += name
    }
  }
  I1 = I1.replace(/ /g, ' ')
  while (I1.indexOf('--') > 0) {
    I1 = I1.replace('--', '-')
  }
  return I1
}
loadPinYinInit.arraySearch = (l1) => { // 在对象中搜索
  let needName = ''
  for (var name in PinYin) {
    if (PinYin[name].indexOf(l1) !== -1) {
      needName = loadPinYinInit.ucfirst(name); break
    }
  }
  return needName
}
loadPinYinInit.ucfirst = (l1) => { // 首字母大写
  if (l1.length > 0) {
    var first = l1.substr(0, 1).toUpperCase()
    var spare = l1.substr(1, l1.length)
    return first + spare
  }
}
// 中文转英文方法 end

export function ceilNum(d, prex = 2) { // 任意位正小数，向上进位（有小数就进一位） d 为被除数，prex 为除数
  // 参考文档: https://blog.csdn.net/zjscy666/article/details/106240798
  if (parseInt(d) === d) return d
  let len = 1
  let i = 0
  while (parseInt(d) !== d) {
    d = d * 10
    len *= 10
    i++
  }
  const last = Math.abs(i <= prex ? 0 : d % (Math.pow(10, (i - prex))))
  let d1 = d - last
  if (last > 0) {
    d1 = d1 + Math.pow(10, (i - prex))
  }
  d1 = d1 / len
  return d1
}

// 前端无bug加减乘除计算 start
export function floatAdd(arg1, arg2) { // 加
  var r1, r2, m
  try { r1 = arg1.toString().split('.')[1].length } catch (e) { r1 = 0 }
  try { r2 = arg2.toString().split('.')[1].length } catch (e) { r2 = 0 }
  m = Math.pow(10, Math.max(r1, r2))
  return (arg1 * m + arg2 * m) / m
}

export function floatSub(arg1, arg2) { // 减
  var r1, r2, m, n
  try { r1 = arg1.toString().split('.')[1].length } catch (e) { r1 = 0 }
  try { r2 = arg2.toString().split('.')[1].length } catch (e) { r2 = 0 }
  m = Math.pow(10, Math.max(r1, r2))
  // 动态控制精度长度
  n = (r1 >= r2) ? r1 : r2
  return ((arg1 * m - arg2 * m) / m).toFixed(n)
}

export function floatMul(arg1, arg2) { // 乘
  if (!(arg1 && arg2)) {
    return 0
  }
  var m = 0; var s1 = arg1.toString(); var s2 = arg2.toString()
  try { m += s1.split('.')[1].length } catch (e) { e }
  try { m += s2.split('.')[1].length } catch (e) { e }
  return Number(s1.replace('.', '')) * Number(s2.replace('.', '')) / Math.pow(10, m)
}

export function floatDiv(arg1, arg2) { // 除
  if (!(arg1 && arg2)) {
    return 0
  }
  var t1 = 0; var t2 = 0; var r1; var r2
  try { t1 = arg1.toString().split('.')[1].length } catch (e) { e }
  try { t2 = arg2.toString().split('.')[1].length } catch (e) { e }

  r1 = Number(arg1.toString().replace('.', ''))

  r2 = Number(arg2.toString().replace('.', ''))
  return (r1 / r2) * Math.pow(10, t2 - t1)
}
// 前端无bug加减乘除计算 end

export function getDayOfWeek(dateString) { // 获取日期是周几, dateString: '2018-05-28 23:24:12'
  const days = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  const date = new Date(dateString)
  return days[date.getDay()]
}

export function isJson(str) { // 判断字符串是否属于JSON字符串
  let isJson = false
  try {
    JSON.stringify(JSON.parse(str), null, 2)
    isJson = true
  } catch (e) {
    e
  }
  return isJson
}

export function objectSort(arr = [], attributeName, order = true) { // 根据属性排序，arr为预排序的数组，attributeName为进行排序属性名称，order为true是顺序排序
  if (arr.length < 2) return arr
  const temp = [...arr]
  for (let i = 0; i < temp.length - 1; i++) {
    let [v, pos] = [temp[i][attributeName], i]
    for (let j = i + 1; j < temp.length; j++) {
      if (Number(v) > Number(temp[j][attributeName])) {
        v = temp[j][attributeName]
        pos = j
      }
    }
    [temp[i], temp[pos]] = [temp[pos], temp[i]]
  }
  return order ? temp : temp.reverse()
}

export function getExt(filename) { // 获取文件扩展名
  if (!filename || typeof filename !== 'string') return ''
  const idx = filename.lastIndexOf('.')
  return idx === -1 ? '' : filename.slice(idx).toLowerCase()
}

export function strRemoveNum(str) { // 去掉字符串中的数字
  return str.replace(/[0-9]+/g, '')
}

export function getRandomColor(excludeColor) { // 生成随机颜色
  const letters = '0123456789ABCDEF'
  let color = '#'
  if (excludeColor) {
    do {
      color = '#'
      for (let i = 0; i < 6; i++) {
        color += letters[Math.floor(Math.random() * 16)]
      }
    } while (color === excludeColor || isColorSimilar(color, excludeColor))
  } else {
    for (let i = 0; i < 6; i++) {
      color += letters[Math.floor(Math.random() * 16)]
    }
  }
  return color
}

function isColorSimilar(hex1, hex2) { // 比较两个颜色的相似度
  const diff = (hex1[1] !== hex2[1]) + (hex1[2] !== hex2[2]) + (hex1[3] !== hex2[3])
  return diff < 2 // 假设如果两个颜色在RGB中的不同位数少于2，则认为它们是相似的
}

export function completionPointList(params) { // 补全数据点列表
  const { initValue, xAxisData, pointList } = params
  const seriesdata = [] // 补全后的数据
  xAxisData.forEach(item => { // 横轴数据
    let hasDate = false // pointList 是否存在 xAxisData中的时间
    pointList.forEach(valueItem => { // 数据点列表
      if (item.timestamp === valueItem.timestamp) {
        hasDate = true
        seriesdata.push(valueItem)
      }
    })
    if (!hasDate) {
      let value = initValue // 补全数据的初始值
      if (seriesdata.length !== 0) {
        value = seriesdata[seriesdata.length - 1].value
      }
      seriesdata.push({
        timestamp: item.timestamp,
        date: item.date,
        value
      })
    }
  })
  return seriesdata
}

export function fileIsImg(file) { // 判断上传的文件是否是图片
  if ((file.type).indexOf('image/') === -1) {
    return {
      sureIsImg: false,
      msg: `请上传图片!`
    }
  } else {
    const suffixIndex = file.name.lastIndexOf('.')
    const suffix = file.name.substring(suffixIndex + 1).toUpperCase()
    if (suffix !== 'BMP' && suffix !== 'JPG' && suffix !== 'JPEG' && suffix !== 'PNG' && suffix !== 'GIF') {
      return {
        sureIsImg: false,
        msg: `请上传图片（格式BMP、JPG、JPEG、PNG、GIF等）!`
      }
    }
  }
  return {
    sureIsImg: true,
    msg: `该文件是图片`
  }
}

export function exportBlobFile(params) { // 导出二进制文件
  const contentType = params.data?.type || params.headers?.['content-type'] || 'application/octet-stream'
  const blob = params.data instanceof Blob ? params.data : new Blob([params.data], { type: contentType })
  const contentDisposition = params.headers?.['content-disposition'] || ''
  let fileName = 'exported-file.xlsx' // 默认文件名
  const encodedFileName = contentDisposition.match(/filename\*\s*=\s*UTF-8''([^;]+)/i)
  const fallbackFileName = contentDisposition.match(/filename\s*=\s*"?([^";]+)"?/i)
  if (encodedFileName) {
    fileName = decodeURIComponent(encodedFileName[1].trim())
  } else if (fallbackFileName) {
    fileName = fallbackFileName[1].trim()
  }

  // 创建下载链接
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.setAttribute('download', fileName)

  // 触发点击事件下载文件
  document.body.appendChild(link)
  link.click()

  // 清理 URL 对象
  window.URL.revokeObjectURL(url)
  document.body.removeChild(link)
}

export function downloadBinaryFile(binaryData, fileName, mimeType = 'application/octet-stream') { // 导出二进制数据流
  // 1. 创建 Blob
  const blob = new Blob([binaryData], { type: mimeType })

  // 2. 创建临时 URL
  const url = URL.createObjectURL(blob)

  // 3. 创建 <a> 标签并触发下载
  const a = document.createElement('a')
  a.href = url
  a.download = fileName || 'download.bin'
  document.body.appendChild(a)
  a.click()

  // 4. 清理资源
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
