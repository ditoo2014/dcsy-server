const STRING_POINT_NAMES = new Set([
    '热失控风险等级',
]);
const STRING_POINT_IDS = new Set([
    'alg_rekongfengxianstage',
]);
function text(value) {
    return String(value ?? '').trim();
}
function isStrGateway(value) {
    return /STR$/i.test(text(value));
}
export function isStringStreamPoint(source = {}) {
    const dataType = text(source.dataType ?? source.type);
    if (dataType === '2')
        return true;
    const dataId = text(source.dataId ?? source.dataID ?? source.DataId ?? source.id);
    if (STRING_POINT_IDS.has(dataId))
        return true;
    const name = text(source.dataName ?? source.name ?? source.pointName ?? source.label ?? source.tagName);
    if (STRING_POINT_NAMES.has(name))
        return true;
    if (isStrGateway(source.gatewayId ?? source.gatewayID ?? source.GatewayID))
        return true;
    return [
        source.tableName,
        source.table,
        source.dataTable,
        source.source,
        source.from,
    ].some(value => /\bSTR\b/i.test(text(value)));
}
export function normalizeStreamDataType(input, source = {}) {
    if (isStringStreamPoint({ ...source, dataType: input }))
        return '2';
    if (typeof input === 'string' && input.trim()) {
        const value = input.trim();
        return value === '0' ? '1' : value;
    }
    const n = Number(input);
    if (!Number.isFinite(n))
        return '1';
    if (n === 0)
        return '1';
    return String(Math.trunc(n));
}
export function formatStreamReportType(input, source = {}) {
    const dataType = normalizeStreamDataType(input, source);
    if (dataType === '2')
        return '字符串类型';
    if (dataType === '1')
        return '浮点类型';
    return '';
}
