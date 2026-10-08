/** 后端机型字段到详情页 URL 段的映射。 */
export const GATEWAY_MODEL_DETAIL_SLUG = {
    TTDC: 'ttdc',
};

// 采集云使用独立布局，内部页面统一挂在该路由前缀下。
export const COLLECTION_BASE_PATH = '/collection';
function pickGatewayModelFieldFromRecord(record) {
    const value = record.model ??
        record.Model ??
        record.modelName ??
        record.ModelName ??
        record.equipmentModel ??
        record.EquipmentModel ??
        record.gatewayModel ??
        record.GatewayModel ??
        record.gatewayType ??
        record.GatewayType ??
        record.deviceModel ??
        record.DeviceModel ??
        record.equipmentType ??
        record.EquipmentType ??
        record.type ??
        record.Type ??
        '';
    return String(value ?? '').trim();
}
export function pickGatewayModelField(obj) {
    if (!obj || typeof obj !== 'object')
        return '';
    const record = obj;
    let model = pickGatewayModelFieldFromRecord(record);
    if (model)
        return model;
    const nested = record.gateway ?? record.gatewayInfo ?? record.data;
    if (nested && typeof nested === 'object') {
        model = pickGatewayModelFieldFromRecord(nested);
    }
    return model;
}
export function normalizeGatewayModelString(model) {
    let value = String(model ?? '');
    if (value.charCodeAt(0) === 0xfeff)
        value = value.slice(1);
    return value
        .replace(/[\u00A0\u1680\u2000-\u200A\u202F\u205F\u3000]/g, ' ')
        .replace(/[\uFF0D\u2013\u2014\u2212]/g, '-')
        .trim()
        .replace(/\s+/g, ' ');
}
function modelFingerprint(model) {
    return normalizeGatewayModelString(model)
        .toLowerCase()
        .replace(/[^a-z0-9]/g, '');
}
export function modelToDetailSlug(model) {
    const key = normalizeGatewayModelString(model);
    if (!key)
        return null;
    const direct = GATEWAY_MODEL_DETAIL_SLUG[key];
    if (direct)
        return direct;
    const lower = key.toLowerCase();
    for (const modelName of Object.keys(GATEWAY_MODEL_DETAIL_SLUG)) {
        if (modelName.toLowerCase() === lower)
            return GATEWAY_MODEL_DETAIL_SLUG[modelName];
    }
    const fingerprint = modelFingerprint(key);
    if (!fingerprint)
        return null;
    for (const modelName of Object.keys(GATEWAY_MODEL_DETAIL_SLUG)) {
        if (modelFingerprint(modelName) === fingerprint)
            return GATEWAY_MODEL_DETAIL_SLUG[modelName];
    }
    return null;
}
export function gatewayDetailPath(model, gatewayId) {
    const slug = modelToDetailSlug(model);
    if (!slug)
        return null;
    return `${COLLECTION_BASE_PATH}/gateway/${slug}/${encodeURIComponent(gatewayId)}`;
}
export const GATEWAY_DETAIL_SLUGS = new Set(Object.values(GATEWAY_MODEL_DETAIL_SLUG));
export function resolveGatewayDetailSlugParam(param) {
    const raw = String(param ?? '').trim();
    if (!raw)
        return null;
    if (GATEWAY_DETAIL_SLUGS.has(raw))
        return raw;
    const lower = raw.toLowerCase();
    for (const slug of GATEWAY_DETAIL_SLUGS) {
        const value = String(slug);
        if (value.toLowerCase() === lower)
            return value;
    }
    const rawFingerprint = modelFingerprint(raw);
    if (!rawFingerprint)
        return null;
    for (const slug of GATEWAY_DETAIL_SLUGS) {
        const value = String(slug);
        if (modelFingerprint(value) === rawFingerprint)
            return value;
    }
    return null;
}
export function flattenGatewayFindoneReturn(ret) {
    if (ret == null || typeof ret !== 'object' || Array.isArray(ret))
        return {};
    const record = ret;
    const inner = record.data;
    const result = { ...record };
    if (inner != null && typeof inner === 'object' && !Array.isArray(inner)) {
        Object.assign(result, inner);
    }
    return result;
}
const GATEWAY_DATAS_KEYS = [
    'gatewayDatas',
    'GatewayDatas',
    'gateway_datas',
    'gatewayDataList',
    'GatewayDataList',
    'dataPoints',
];
export function pickGatewayDatasArray(obj, depth = 0) {
    if (depth > 5 || obj == null || typeof obj !== 'object')
        return [];
    const record = obj;
    for (const key of GATEWAY_DATAS_KEYS) {
        const value = record[key];
        if (Array.isArray(value) && value.length > 0)
            return value;
    }
    for (const nestKey of ['gateway', 'gatewayInfo']) {
        const nested = record[nestKey];
        if (nested != null && typeof nested === 'object' && !Array.isArray(nested)) {
            const got = pickGatewayDatasArray(nested, depth + 1);
            if (got.length)
                return got;
        }
    }
    return [];
}
export function pickDcFsuSecondaryGatewayIdForIntact(gateway, primaryGatewayId) {
    const primary = String(primaryGatewayId ?? '').trim();
    if (!gateway || typeof gateway !== 'object')
        return null;
    const record = gateway;
    const tryValue = (value) => {
        const text = typeof value === 'string' ? value.trim() : value != null && value !== '' ? String(value).trim() : '';
        if (!text || text === '-' || text === primary)
            return null;
        return text;
    };
    const keys = [
        'STR',
        'Str',
        'strGateway',
        'strGatewayId',
        'STRGatewayId',
        'gatewayIdStr',
        'GatewayIdStr',
        'serialStr',
        'SerialStr',
        'subGatewayId',
        'slaveGatewayId',
        'bindGatewayId',
        'batteryGatewayId',
        'equipmentGatewayId',
        'csvGatewayId',
        'fsuGatewayId',
    ];
    for (const key of keys) {
        const hit = tryValue(record[key]);
        if (hit)
            return hit;
    }
    const nests = [record.csvConfig, record.fsuConfig, record.gatewayConfig, record.config, record.dcFsuConfig].filter((value) => value != null && typeof value === 'object' && !Array.isArray(value));
    for (const nested of nests) {
        for (const key of keys) {
            const hit = tryValue(nested[key]);
            if (hit)
                return hit;
        }
    }
    const card = tryValue(record.card);
    if (card && card.length >= 8)
        return card;
    return null;
}
export function dcFsuTimesequenceGatewayIdStrBranch(primaryGatewayId) {
    const base = String(primaryGatewayId ?? '')
        .trim()
        .replace(/STR$/i, '');
    return base ? `${base}STR` : '';
}
