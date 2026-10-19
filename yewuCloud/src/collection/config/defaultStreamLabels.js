export const TTDC_DEFAULT_LABEL_MAPPINGS = [
    { labelName: 'SOH', dataName: 'SOH' },
    { labelName: 'SOC', dataName: 'SOC' },
    { labelName: '温度采集数', dataName: 'N (电芯温度数量)' },
    { labelName: '1号单体SOC', dataName: '1号单体SOC' },
    { labelName: '2号单体SOC', dataName: '2号单体SOC' },
    { labelName: '3号单体SOC', dataName: '3号单体SOC' },
    { labelName: '4号单体SOC', dataName: '4号单体SOC' },
    { labelName: '5号单体SOC', dataName: '5号单体SOC' },
    { labelName: '6号单体SOC', dataName: '6号单体SOC' },
    { labelName: '7号单体SOC', dataName: '7号单体SOC' },
    { labelName: '8号单体SOC', dataName: '8号单体SOC' },
    { labelName: '9号单体SOC', dataName: '9号单体SOC' },
    { labelName: '10号单体SOC', dataName: '10号单体SOC' },
    { labelName: '11号单体SOC', dataName: '11号单体SOC' },
    { labelName: '12号单体SOC', dataName: '12号单体SOC' },
    { labelName: '13号单体SOC', dataName: '13号单体SOC' },
    { labelName: '14号单体SOC', dataName: '14号单体SOC' },
    { labelName: '15号单体SOC', dataName: '15号单体SOC' },
    { labelName: '16号单体SOC', dataName: '16号单体SOC' },
    { labelName: '压差', dataName: '压差' },
    { labelName: '温差', dataName: '温差' },
    { labelName: 'MOS温度', dataName: 'MOS 温度数据' },
    { labelName: '热失控风险等级', dataName: '热失控风险等级' },
    { labelName: '单体电池支数', dataName: 'M (单体电池数量)' },
    { labelName: '环境温度', dataName: '环境温度数据' },
    { labelName: '放电时长', dataName: '放电时长' },
    { labelName: '循环次数', dataName: '电池循环次数' },
    { labelName: '组电压', dataName: '电池总压数据' },
    { labelName: '组电流', dataName: '电池电流数据' },
    { labelName: '电池组SOC', dataName: '电池组荷电状态 (SOC)' },
    { labelName: '电池组SOH', dataName: '电池组健康状态 (SOH)' },
    { labelName: '最高单体电池电压', dataName: '最高单体电池电压' },
    { labelName: '最高电压单体编号', dataName: '最高电压单体编号' },
    { labelName: '最低单体电池电压', dataName: '最低单体电池电压' },
    { labelName: '最低电压单体编号', dataName: '最低电压单体编号' },
    { labelName: '最高单体电池温度', dataName: '最高单体电池温度' },
    { labelName: '最低单体电池温度', dataName: '最低单体电池温度' },
    { labelName: 'Link电压', dataName: 'Link电压' },
    { labelName: '剩余放电时长', dataName: '剩余放电时长' },
    { labelName: '剩余充电时长', dataName: '剩余充电时长' },
    { labelName: '电池额定容量', dataName: '电池额定容量' },
    { labelName: '年', dataName: '年' },
    { labelName: '月', dataName: '月' },
    { labelName: '日', dataName: '日' },
    { labelName: '时', dataName: '时' },
    { labelName: '分', dataName: '分' },
    { labelName: '秒', dataName: '秒' },
    { labelName: '累计充电量', dataName: '累计充电容量' },
    { labelName: '累计放电量', dataName: '累计放电容量' },
    { labelName: 'PTC温度', dataName: 'PTC温度' },
    { labelName: '软件版本号', dataName: '软件版本号' },
];
export function normalizeLabelKey(value) {
    return String(value ?? '')
        .trim()
        .replace(/（/g, '(')
        .replace(/）/g, ')')
        .replace(/\s+/g, '');
}
function resolveDynamicTtdcLabelName(dataName) {
    const key = normalizeLabelKey(dataName);
    const cellVoltageMatch = key.match(/^单体电池(\d+)电压$/);
    if (cellVoltageMatch?.[1])
        return `电压${Number(cellVoltageMatch[1])}`;
    const cellTemperatureMatch = key.match(/^电芯温度(\d+)数据$/);
    if (cellTemperatureMatch?.[1])
        return `温度${Number(cellTemperatureMatch[1])}`;
    return '';
}
function createMappingResolver(mappings) {
    const dict = new Map();
    for (const item of mappings) {
        dict.set(normalizeLabelKey(item.dataName), item.labelName);
    }
    return (dataName) => {
        const key = normalizeLabelKey(dataName);
        return dict.get(key) ?? resolveDynamicTtdcLabelName(dataName);
    };
}
export function createDefaultLabelNameResolver(model) {
    if (String(model || '').trim().toUpperCase() !== 'TTDC') {
        return () => '';
    }
    return createMappingResolver(TTDC_DEFAULT_LABEL_MAPPINGS);
}
