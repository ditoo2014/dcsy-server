export const chinaRegions = { // 省份及其下辖的市
  '北京市': { type: 'city', children: ['东城区', '西城区', '朝阳区', '丰台区', '石景山区', '海淀区', '顺义区', '通州区', '大兴区', '房山区', '门头沟区', '昌平区', '平谷区', '密云区', '怀柔区', '延庆区'] },
  '天津市': { type: 'city', children: ['和平区', '河东区', '河西区', '南开区', '河北区', '红桥区', '东丽区', '西青区', '津南区', '北辰区', '武清区', '宝坻区', '滨海新区', '宁河区', '静海区', '蓟州区'] },
  '河北省': { type: 'province', children: ['石家庄市', '唐山市', '秦皇岛市', '邯郸市', '邢台市', '保定市', '张家口市', '承德市', '沧州市', '廊坊市', '衡水市'] },
  '山西省': { type: 'province', children: ['太原市', '大同市', '阳泉市', '长治市', '晋城市', '朔州市', '晋中市', '运城市', '忻州市', '临汾市', '吕梁市'] },
  '内蒙古自治区': { type: 'province', children: ['呼和浩特市', '包头市', '乌海市', '赤峰市', '通辽市', '鄂尔多斯市', '呼伦贝尔市', '巴彦淖尔市', '乌兰察布市', '兴安盟', '锡林郭勒盟', '阿拉善盟'] },
  '辽宁省': { type: 'province', children: ['沈阳市', '大连市', '鞍山市', '抚顺市', '本溪市', '丹东市', '锦州市', '营口市', '阜新市', '辽阳市', '盘锦市', '铁岭市', '朝阳市', '葫芦岛市'] },
  '吉林省': { type: 'province', children: ['长春市', '吉林市', '四平市', '辽源市', '通化市', '白山市', '松原市', '白城市', '延边朝鲜族自治州'] },
  '黑龙江省': { type: 'province', children: ['哈尔滨市', '齐齐哈尔市', '鸡西市', '鹤岗市', '双鸭山市', '大庆市', '伊春市', '佳木斯市', '七台河市', '牡丹江市', '黑河市', '绥化市', '大兴安岭地区'] },
  '上海市': { type: 'city', children: ['黄浦区', '徐汇区', '长宁区', '静安区', '普陀区', '虹口区', '杨浦区', '闵行区', '宝山区', '嘉定区', '浦东新区', '金山区', '松江区', '青浦区', '奉贤区', '崇明区'] },
  '江苏省': { type: 'province', children: ['南京市', '无锡市', '徐州市', '常州市', '苏州市', '南通市', '连云港市', '淮安市', '盐城市', '扬州市', '镇江市', '泰州市', '宿迁市'] },
  '浙江省': { type: 'province', children: ['杭州市', '宁波市', '温州市', '嘉兴市', '湖州市', '绍兴市', '金华市', '衢州市', '舟山市', '台州市', '丽水市'] },
  '安徽省': { type: 'province', children: ['合肥市', '芜湖市', '蚌埠市', '淮南市', '马鞍山市', '淮北市', '铜陵市', '安庆市', '黄山市', '滁州市', '阜阳市', '宿州市', '六安市', '亳州市', '池州市', '宣城市'] },
  '福建省': { type: 'province', children: ['福州市', '厦门市', '莆田市', '三明市', '泉州市', '漳州市', '南平市', '龙岩市', '宁德市'] },
  '江西省': { type: 'province', children: ['南昌市', '景德镇市', '萍乡市', '九江市', '新余市', '鹰潭市', '赣州市', '吉安市', '宜春市', '抚州市', '上饶市'] },
  '山东省': { type: 'province', children: ['济南市', '青岛市', '淄博市', '枣庄市', '东营市', '烟台市', '潍坊市', '济宁市', '泰安市', '威海市', '日照市', '临沂市', '德州市', '聊城市', '滨州市', '菏泽市'] },
  '河南省': { type: 'province', children: ['郑州市', '开封市', '洛阳市', '平顶山市', '安阳市', '鹤壁市', '新乡市', '焦作市', '濮阳市', '许昌市', '漯河市', '三门峡市', '南阳市', '商丘市', '信阳市', '周口市', '驻马店市', '济源市'] },
  '湖北省': { type: 'province', children: ['武汉市', '黄石市', '十堰市', '宜昌市', '襄阳市', '鄂州市', '荆门市', '孝感市', '荆州市', '黄冈市', '咸宁市', '随州市', '恩施土家族苗族自治州', '仙桃市', '潜江市', '天门市', '神农架林区'] },
  '湖南省': { type: 'province', children: ['长沙市', '株洲市', '湘潭市', '衡阳市', '邵阳市', '岳阳市', '常德市', '张家界市', '益阳市', '郴州市', '永州市', '怀化市', '娄底市', '湘西土家族苗族自治州'] },
  '广东省': { type: 'province', children: ['广州市', '韶关市', '深圳市', '珠海市', '汕头市', '佛山市', '江门市', '湛江市', '茂名市', '肇庆市', '惠州市', '梅州市', '汕尾市', '河源市', '阳江市', '清远市', '东莞市', '中山市', '潮州市', '揭阳市', '云浮市'] },
  '广西壮族自治区': { type: 'province', children: ['南宁市', '柳州市', '桂林市', '梧州市', '北海市', '防城港市', '钦州市', '贵港市', '玉林市', '百色市', '贺州市', '河池市', '来宾市', '崇左市'] },
  '海南省': { type: 'province', children: ['海口市', '三亚市', '三沙市', '儋州市', '五指山市', '琼海市', '文昌市', '万宁市', '东方市', '定安县', '屯昌县', '澄迈县', '临高县', '白沙黎族自治县', '昌江黎族自治县', '乐东黎族自治县', '陵水黎族自治县', '保亭黎族苗族自治县', '琼中黎族苗族自治县'] },
  '重庆市': { type: 'city', children: ['万州区', '黔江区', '涪陵区', '渝中区', '大渡口区', '江北区', '沙坪坝区', '九龙坡区', '南岸区', '北碚区', '渝北区', '巴南区', '长寿区', '江津区', '合川区', '永川区', '南川区', '綦江区', '大足区', '璧山区', '铜梁区', '潼南区', '荣昌区', '开州区', '梁平区', '武隆区', '城口县', '丰都县', '垫江县', '忠县', '云阳县', '奉节县', '巫山县', '巫溪县', '石柱土家族自治县', '秀山土家族苗族自治县', '酉阳土家族苗族自治县', '彭水苗族土家族自治县'] },
  '四川省': { type: 'province', children: ['成都市', '自贡市', '攀枝花市', '泸州市', '德阳市', '绵阳市', '广元市', '遂宁市', '内江市', '乐山市', '南充市', '眉山市', '宜宾市', '广安市', '达州市', '雅安市', '巴中市', '资阳市', '阿坝藏族羌族自治州', '甘孜藏族自治州', '凉山彝族自治州'] },
  '贵州省': { type: 'province', children: ['贵阳市', '六盘水市', '遵义市', '安顺市', '毕节市', '铜仁市', '黔西南布依族苗族自治州', '黔东南苗族侗族自治州', '黔南布依族苗族自治州'] },
  '云南省': { type: 'province', children: ['昆明市', '曲靖市', '玉溪市', '保山市', '昭通市', '丽江市', '普洱市', '临沧市', '楚雄彝族自治州', '红河哈尼族彝族自治州', '文山壮族苗族自治州', '西双版纳傣族自治州', '大理白族自治州', '德宏傣族景颇族自治州', '怒江傈僳族自治州', '迪庆藏族自治州'] },
  '西藏自治区': { type: 'province', children: ['拉萨市', '日喀则市', '昌都市', '林芝市', '山南市', '那曲市', '阿里地区'] },
  '陕西省': { type: 'province', children: ['西安市', '铜川市', '宝鸡市', '咸阳市', '渭南市', '延安市', '汉中市', '榆林市', '安康市', '商洛市'] },
  '甘肃省': { type: 'province', children: ['兰州市', '嘉峪关市', '金昌市', '白银市', '天水市', '武威市', '张掖市', '平凉市', '酒泉市', '庆阳市', '定西市', '陇南市', '临夏回族自治州', '甘南藏族自治州'] },
  '青海省': { type: 'province', children: ['西宁市', '海东市', '海北藏族自治州', '黄南藏族自治州', '海南藏族自治州', '果洛藏族自治州', '玉树藏族自治州', '海西蒙古族藏族自治州'] },
  '宁夏回族自治区': { type: 'province', children: ['银川市', '石嘴山市', '吴忠市', '固原市', '中卫市'] },
  '新疆维吾尔自治区': { type: 'province', children: ['乌鲁木齐市', '克拉玛依市', '吐鲁番市', '哈密市', '昌吉回族自治州', '博尔塔拉蒙古自治州', '巴音郭楞蒙古自治州', '阿克苏地区', '克孜勒苏柯尔克孜自治州', '喀什地区', '和田地区', '伊犁哈萨克自治州', '塔城地区', '阿勒泰地区', '石河子市', '阿拉尔市', '图木舒克市', '五家渠市', '北屯市', '铁门关市', '双河市', '可克达拉市', '昆玉市', '胡杨河市', '新星市'] },
  '台湾省': { type: 'province', children: ['台北市', '高雄市', '新北市', '台中市', '台南市', '桃园市', '基隆市', '新竹市', '嘉义市', '新竹县', '苗栗县', '彰化县', '南投县', '云林县', '嘉义县', '屏东县', '宜兰县', '花莲县', '台东县', '澎湖县'] },
  '香港特别行政区': { type: 'city', children: ['香港岛', '九龙半岛', '新界'] },
  '澳门特别行政区': { type: 'city', children: ['澳门半岛', '氹仔岛', '路环岛'] }

  // 部分市级单位及其下辖的区/县
  // '广州市': { type: 'city', children: ['荔湾区', '越秀区', '海珠区', '天河区', '白云区', '黄埔区', '番禺区', '花都区', '南沙区', '从化区', '增城区'] },
  // '深圳市': { type: 'city', children: ['罗湖区', '福田区', '南山区', '宝安区', '龙岗区', '盐田区', '龙华区', '坪山区', '光明区', '大鹏新区'] },
  // '上海市': { type: 'city', children: ['黄浦区', '徐汇区', '长宁区', '静安区', '普陀区', '虹口区', '杨浦区', '闵行区', '宝山区', '嘉定区', '浦东新区', '金山区', '松江区', '青浦区', '奉贤区', '崇明区'] },
  // '北京市': { type: 'city', children: ['东城区', '西城区', '朝阳区', '丰台区', '石景山区', '海淀区', '顺义区', '通州区', '大兴区', '房山区', '门头沟区', '昌平区', '平谷区', '密云区', '怀柔区', '延庆区'] },
  // '南京市': { type: 'city', children: ['玄武区', '秦淮区', '建邺区', '鼓楼区', '浦口区', '栖霞区', '雨花台区', '江宁区', '六合区', '溧水区', '高淳区'] },
  // '杭州市': { type: 'city', children: ['上城区', '下城区', '江干区', '拱墅区', '西湖区', '滨江区', '萧山区', '余杭区', '富阳区', '临安区', '桐庐县', '淳安县', '建德市'] }
}

const REGION_COMMON_SUFFIXS = ['特别行政区', '自治区', '自治州', '地区', '盟', '省', '市', '县', '区', '州']
const COMPANY_COMMON_SUFFIXS = ['股份有限公司', '有限责任公司', '集团有限公司', '有限公司', '分公司', '公司']
const REGION_ETHNIC_TOKENS = [
  '壮族',
  '维吾尔',
  '回族',
  '藏族',
  '苗族',
  '彝族',
  '土家族',
  '哈尼族',
  '傣族',
  '侗族',
  '傈僳族',
  '白族',
  '羌族',
  '蒙古族',
  '哈萨克',
  '布依族',
  '朝鲜族',
  '拉祜族',
  '佤族',
  '仡佬族',
  '景颇族',
  '纳西族',
  '德昂族',
  '独龙族',
  '水族',
  '毛南族',
  '仫佬族',
  '普米族',
  '阿昌族',
  '怒族',
  '门巴族',
  '珞巴族',
  '俄罗斯族',
  '鄂温克族',
  '鄂伦春族',
  '达斡尔族',
  '京族',
  '塔塔尔族',
  '赫哲族',
  '高山族',
  '撒拉族',
  '塔吉克族',
  '乌孜别克族',
  '柯尔克孜族',
  '锡伯族',
  '裕固族',
  '保安族',
  '东乡族'
]

function normalizeRegionMatchText(value) {
  return String(value ?? '').trim().replace(/\s+/g, '')
}

function stripSuffixes(value, suffixList) {
  let text = normalizeRegionMatchText(value)
  let changed = true
  while (changed && text) {
    changed = false
    suffixList.forEach(suffix => {
      if (text.endsWith(suffix)) {
        text = text.slice(0, -suffix.length)
        changed = true
      }
    })
  }
  return text
}

function stripRegionEthnicTokens(value) {
  let text = normalizeRegionMatchText(value)
  REGION_ETHNIC_TOKENS.forEach(token => {
    text = text.replace(new RegExp(token, 'g'), '')
  })
  return text
}

function getRegionMatchVariants(value) {
  const compact = normalizeRegionMatchText(value)
  if (!compact) {
    return []
  }
  const companyLike = stripSuffixes(compact, COMPANY_COMMON_SUFFIXS)
  const regionLike = stripSuffixes(stripRegionEthnicTokens(compact), REGION_COMMON_SUFFIXS)
  const companyThenRegion = stripSuffixes(stripRegionEthnicTokens(companyLike), REGION_COMMON_SUFFIXS)
  const regionThenCompany = stripSuffixes(regionLike, COMPANY_COMMON_SUFFIXS)
  return [...new Set([
    compact,
    companyLike,
    regionLike,
    companyThenRegion,
    regionThenCompany
  ].filter(Boolean))]
}

export function regionNameMatchScore(leftName, rightName) {
  if (!leftName || !rightName) {
    return 0
  }

  const leftText = normalizeRegionMatchText(leftName)
  const rightText = normalizeRegionMatchText(rightName)
  if (leftText === rightText) {
    return 4
  }

  const leftCompanyName = stripSuffixes(leftText, COMPANY_COMMON_SUFFIXS)
  const rightCompanyName = stripSuffixes(rightText, COMPANY_COMMON_SUFFIXS)
  if (leftCompanyName === rightText || rightCompanyName === leftText) {
    return 3
  }

  const leftRegionName = stripSuffixes(stripRegionEthnicTokens(leftCompanyName), REGION_COMMON_SUFFIXS)
  const rightRegionName = stripSuffixes(stripRegionEthnicTokens(rightCompanyName), REGION_COMMON_SUFFIXS)
  if (leftRegionName && leftRegionName === rightRegionName) {
    return 2
  }

  const leftVariants = getRegionMatchVariants(leftName)
  const rightVariants = getRegionMatchVariants(rightName)
  const fuzzyMatch = leftVariants.some(leftVariant => {
    return rightVariants.some(rightVariant => {
      return leftVariant === rightVariant ||
        leftVariant.includes(rightVariant) ||
        rightVariant.includes(leftVariant)
    })
  })
  return fuzzyMatch ? 1 : 0
}

export function regionNameMatches(leftName, rightName) {
  return regionNameMatchScore(leftName, rightName) > 0
}

export function findBestCorpByRegionName(treeList = [], regionName, allowedCorpKeys = null) {
  if (!regionName) {
    return null
  }

  const allowedKeys = Array.isArray(allowedCorpKeys)
    ? new Set(allowedCorpKeys.map(item => String(item)))
    : null
  let matchedCorp = null
  let bestScore = 0

  treeList.forEach(item => {
    if (!item || !item.name || (allowedKeys && !allowedKeys.has(String(item.id)))) {
      return
    }
    const score = regionNameMatchScore(item.name, regionName)
    if (score > bestScore) {
      bestScore = score
      matchedCorp = item
    }
  })

  return matchedCorp
}

export function resolveSiteRegionName(siteItem = {}) {
  const corpName = normalizeRegionMatchText(siteItem.corpName)
  if (corpName) {
    return corpName
  }
  const detailMsg = siteItem.siteDetails || {}
  const cityName = normalizeRegionMatchText(detailMsg.cityName)
  if (cityName) {
    return cityName
  }
  const provinceName = normalizeRegionMatchText(detailMsg.provinceName)
  if (provinceName) {
    return provinceName
  }
  return ''
}

export function judgeRegionType(regionName) { // 判断一个地区是省还是市
  const region = chinaRegions[regionName]
  let chinaRegionsKey = null
  if (!region) { // 不属于第一级行政区划，可能是直辖市或特别行政区
    for (const key in chinaRegions) {
      if (chinaRegions[key].children.includes(regionName)) {
        chinaRegionsKey = key
      }
    }
    if (chinaRegionsKey) { // 区县等下级单位，父级为直辖市/省，当前层级视为市
      return {
        success: true,
        regionName,
        type: '市',
        children: [],
        hasCityChildren: false
      }
    }
    return {
      success: false,
      message: `未找到"${regionName}"的行政区划信息。`
    }
  }

  const children = region.children || []
  // 检查是否有下一级单位
  if (children.length === 0) {
    return {
      success: false,
      message: `未找到"${regionName}"的下一级行政单位信息。`
    }
  }

  // 判断下一级单位是否包含市
  // 简单规则：名称以"市"结尾的视为市（实际应用中可能需要更复杂的判断）
  const hasCityChildren = children.some(child => {
    return child.endsWith('市') ||
      chinaRegions[child]?.type === 'city'
  })

  // 根据规则判断当前地区类型
  const type = hasCityChildren ? '省' : '市'

  return {
    success: true,
    regionName: regionName,
    type: type,
    children: children,
    hasCityChildren: hasCityChildren
  }
}

export function lastudgeRegionType(regionName) { // 判断一个地区上一级地区是省还是市
  let parentRegion = {} // 父级地区信息
  for (const key in chinaRegions) {
    chinaRegions[key].children.forEach(item => {
      if (item === regionName) {
        parentRegion = {
          areaName: key,
          data: chinaRegions[key]
        }
      }
    })
  }
  return parentRegion
}
