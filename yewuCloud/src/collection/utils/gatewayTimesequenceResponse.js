/** 解析 /tolink/timesequence/getdata 的常见返回结构。 */
export function normalizeRealtimeTs(ts) {
    if (ts == null || ts === '')
        return '-';
    const n = Number(ts);
    if (Number.isFinite(n)) {
        const ms = n < 10_000_000_000 ? n * 1000 : n;
        return new Date(ms).toLocaleString();
    }
    const s = String(ts).trim();
    return s || '-';
}
export function parseRealtimeTsMs(ts) {
    if (ts == null || ts === '')
        return null;
    const n = Number(ts);
    if (Number.isFinite(n))
        return n < 10_000_000_000 ? n * 1000 : n;
    const t = Date.parse(String(ts));
    return Number.isFinite(t) ? t : null;
}
export function normalizeRealtimeValue(v) {
    if (v == null)
        return '-';
    const s = String(v).trim();
    return s || '-';
}
export function extractRealtimeItems(raw) {
    const response = raw;
    const points = response?.return?.points;
    const times = response?.return?.times;
    const values = response?.return?.values;
    if (Array.isArray(points) && Array.isArray(values)) {
        return points
            .map((point, index) => ({
            dataId: String(point ?? '').trim(),
            value: values[index],
            ts: Array.isArray(times) ? times[index] : undefined,
            status: undefined,
        }))
            .filter(item => !!item.dataId);
    }
    const list = response?.return?.data ?? response?.data ?? response;
    if (Array.isArray(list)) {
        return list
            .map((item) => ({
            dataId: String(item?.dataId ?? item?.DataID ?? item?.dataID ?? item?.DataId ?? item?.id ?? '').trim(),
            value: item?.value ?? item?.Value ?? item?.val ?? item?.data,
            ts: item?.ts ?? item?.time ?? item?.Time ?? item?.timestamp,
            status: item?.status ?? item?.Status,
        }))
            .filter((item) => !!item.dataId);
    }
    if (list && typeof list === 'object') {
        return Object.entries(list)
            .map(([key, value]) => ({
            dataId: String(key).trim(),
            value: value?.value ?? value,
            ts: value?.ts ?? value?.time,
            status: value?.status ?? value?.Status,
        }))
            .filter(item => !!item.dataId);
    }
    return [];
}
