<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
// 局部注册原生抽屉，避免继承业务云抽屉封装的默认行为。
import { ElDrawer, ElMessage, ElMessageBox } from 'element-plus';
import { useAuthStore } from '../stores/auth';
import { addGatewayModule, deleteGatewayModule, deleteGatewayPoint, fetchGatewayModules, fetchGatewayOne, updateGatewayModule, } from '../api/gateway-module';
import { useGatewayDetailRouteSync } from '../composables/useGatewayDetailRouteSync';
import { fetchTtdcTemplateDetail, fetchTtdcTemplateOptions } from '../api/ttdc-template';
import { formatTtdcConversionType, isTtdcDecimalConversion, TTDC_CONV_HEX_TO_BIN, } from '../config/ttdc-transport';
import { fetchGatewayRealtime } from '../api/gateway-realtime';
import RealtimePointListPanel from '../components/gateway/RealtimePointListPanel.vue';
const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const { abortIfRouteMismatch } = useGatewayDetailRouteSync();
const loading = ref(false);
const submitLoading = ref(false);
const drawerVisible = ref(false);
const drawerMode = ref('add');
const inlineAddVisible = ref(false);
const inlineEditVisible = ref(false);
const inlineDetailVisible = ref(false);
const inlineContextAdr = ref('');
const detailModule = ref(null);
const templateOptions = ref([]);
const templateLoading = ref(false);
const gateway = ref(null);
const gatewayModules = ref([]);
const pointListUpdatedAt = ref('');
const detailViewMode = ref('config');
const realtimeMap = ref(new Map());
const realtimeLoading = ref(false);
/** 网关数据列表：首次完成实时查询后再渲染表格，避免先显示占位值再刷新为真实值。 */
const realtimeListContentReady = ref(false);
const realtimeOrder = ref([]);
const globalRefreshEnabled = ref(true);
const lastRealtimeSuccessAt = ref(null);
const statusNow = ref(Date.now());
const isDarkTheme = ref(false);
let realtimeTimer = null;
let statusTimer = null;
let onlineStatusTimer = null;
let themeObserver = null;
let themeMediaQuery = null;
const gatewayId = computed(() => String(route.params.gatewayId || ''));
function detectDarkTheme() {
    if (typeof window === 'undefined')
        return false;
    const html = document.documentElement;
    const body = document.body;
    const byAttr = html.getAttribute('data-theme') === 'dark' ||
        body?.getAttribute?.('data-theme') === 'dark';
    const byClass = html.classList.contains('dark') ||
        body?.classList?.contains('dark') ||
        html.classList.contains('theme-dark') ||
        body?.classList?.contains('theme-dark') ||
        html.classList.contains('dark-theme') ||
        body?.classList?.contains('dark-theme');
    const bySystem = window.matchMedia?.('(prefers-color-scheme: dark)')?.matches ?? false;
    return byAttr || byClass || bySystem;
}
function syncThemeMode() {
    isDarkTheme.value = detectDarkTheme();
}
const ONLINE_STALE_MS = 30 * 60 * 1000;
const gatewayOnline = computed(() => {
    if (lastRealtimeSuccessAt.value == null)
        return false;
    return statusNow.value - lastRealtimeSuccessAt.value <= ONLINE_STALE_MS;
});
const gatewayOnlineText = computed(() => (gatewayOnline.value ? '在线' : '离线'));
const backButtonStyle = computed(() => {
    if (!isDarkTheme.value)
        return {};
    return {
        background: 'linear-gradient(180deg, #16314d 0%, #12263d 100%)',
        borderColor: '#4f7fab',
        color: '#eaf4ff',
        boxShadow: 'inset 0 0 0 1px rgba(143, 212, 255, 0.2), 0 2px 8px rgba(2, 6, 23, 0.28)',
    };
});
/** 数据点按 id 升序；无 id 的排在其后，相对顺序不变。 */
function sortPointsById(list) {
    if (!Array.isArray(list))
        return [];
    return [...list].sort((a, b) => {
        const ia = a?.id != null ? Number(a.id) : NaN;
        const ib = b?.id != null ? Number(b.id) : NaN;
        const hasA = Number.isFinite(ia);
        const hasB = Number.isFinite(ib);
        if (hasA && hasB)
            return ia - ib;
        if (hasA && !hasB)
            return -1;
        if (!hasA && hasB)
            return 1;
        return 0;
    });
}
const formListSorted = computed(() => sortPointsById(form.value.list));
const POINT_PAGE_SIZE = 25;
const pointPage = ref(1);
const formListPaged = computed(() => {
    const start = (pointPage.value - 1) * POINT_PAGE_SIZE;
    return formListSorted.value.slice(start, start + POINT_PAGE_SIZE);
});
const pointPaginationVisible = computed(() => formListSorted.value.length > POINT_PAGE_SIZE);
function resetPointPage() {
    pointPage.value = 1;
}
function clampPointPage() {
    const maxPage = Math.max(1, Math.ceil(formListSorted.value.length / POINT_PAGE_SIZE));
    if (pointPage.value > maxPage)
        pointPage.value = maxPage;
}
function goToLastPointPage() {
    pointPage.value = Math.max(1, Math.ceil(formListSorted.value.length / POINT_PAGE_SIZE));
}
const detailModuleListSorted = computed(() => sortPointsById(detailModule.value?.list));
const moduleGroupsByAdr = computed(() => {
    const list = Array.isArray(gatewayModules.value) ? gatewayModules.value : [];
    const groups = new Map();
    const order = [];
    for (const item of list) {
        const adr = String(item?.ttdcConfig?.moduleAdr ?? '').trim() || '-';
        if (!groups.has(adr)) {
            groups.set(adr, []);
            order.push(adr);
        }
        groups.get(adr).push(item);
    }
    return order.map(adr => ({ adr, items: groups.get(adr) ?? [] }));
});
const adrGroupExpanded = ref({});
function isAdrGroupExpanded(adr) {
    const v = adrGroupExpanded.value[adr];
    // 默认折叠：只有显式设置为 true 才展开。
    return v === true;
}
function toggleAdrGroup(adr) {
    adrGroupExpanded.value[adr] = !isAdrGroupExpanded(adr);
}
function getModuleAdr(moduleItem) {
    return String(moduleItem?.ttdcConfig?.moduleAdr ?? '').trim() || '-';
}
function openAdrGroup(adr) {
    adrGroupExpanded.value[adr] = true;
}
function getGroupNames(items) {
    const names = (Array.isArray(items) ? items : [])
        .map(x => String(x?.name ?? '').trim())
        .filter(Boolean);
    return names.length ? names : ['-'];
}
const gatewayPointRows = computed(() => {
    const modules = Array.isArray(gatewayModules.value) ? gatewayModules.value : [];
    const rows = [];
    for (const moduleItem of modules) {
        const moduleId = moduleItem?.id != null ? Number(moduleItem.id) : undefined;
        const moduleName = String(moduleItem?.name ?? '').trim() || '未命名通道';
        const ttdcAdr = String(moduleItem?.ttdcConfig?.moduleAdr ?? '').trim() || '-';
        const cid2 = formatTtdcControlCode(moduleItem?.ttdcConfig?.moduleCid2);
        const points = Array.isArray(moduleItem?.list) ? moduleItem.list : [];
        for (const pointItem of points) {
            const pointId = pointItem?.id != null ? Number(pointItem.id) : undefined;
            const dataIdRaw = pointItem?.dataId ?? pointItem?.DataID ?? pointItem?.dataID ?? pointItem?.DataId;
            const dataId = dataIdRaw != null && String(dataIdRaw).trim() ? String(dataIdRaw).trim() : (pointId != null ? String(pointId) : '');
            const pointName = String(pointItem?.name ?? '').trim() || '-';
            rows.push({
                pointKey: `${moduleId ?? 'm'}:${pointId ?? `${ttdcAdr}:${pointName}`}`,
                pointId,
                moduleId,
                moduleName,
                ttdcAdr,
                cid2,
                pointName,
                displayName: pointName,
                dataId,
                dataType: Number(pointItem?.dataType ?? 0),
                transport: String(pointItem?.transport ?? ''),
                formula: String(pointItem?.formula ?? ''),
                unformula: String(pointItem?.unformula ?? ''),
                displayValue: '-',
                valueTime: pointListUpdatedAt.value || '-',
                quality: 'unknown',
            });
        }
    }
    // merge realtime values by pointId when present
    const map = realtimeMap.value;
    const orderMap = new Map();
    for (let i = 0; i < realtimeOrder.value.length; i++) {
        const k = String(realtimeOrder.value[i] ?? '').trim();
        if (!k)
            continue;
        if (!orderMap.has(k))
            orderMap.set(k, i);
    }
    if (map.size) {
        for (const r of rows) {
            const key = String(r.dataId ?? '').trim();
            if (!key)
                continue;
            const hit = map.get(key);
            if (!hit)
                continue;
            r.displayValue = hit.value;
            r.valueTime = hit.ts;
            r.status = String(hit.status ?? '').trim();
            r.orderIndex = orderMap.has(key) ? orderMap.get(key) : undefined;
        }
    }
    return rows;
});
function normalizeTs(ts) {
    if (ts == null || ts === '')
        return '-';
    const n = Number(ts);
    if (Number.isFinite(n)) {
        // backend may return seconds
        const ms = n < 10_000_000_000 ? n * 1000 : n;
        return new Date(ms).toLocaleString();
    }
    const s = String(ts).trim();
    return s || '-';
}
function parseRealtimeTsMs(ts) {
    if (ts == null || ts === '')
        return null;
    const n = Number(ts);
    if (Number.isFinite(n))
        return n < 10_000_000_000 ? n * 1000 : n;
    const t = Date.parse(String(ts));
    return Number.isFinite(t) ? t : null;
}
function normalizeValue(v) {
    if (v == null)
        return '-';
    const s = String(v).trim();
    if (!s)
        return '-';
    return s;
}
function extractRealtimeItems(raw) {
    // Try common shapes defensively:
    // - { return: { data: [...] } } (existing APIs)
    // - { data: [...] }
    // - [...] directly
    // Special shape from /tolink/timesequence/getdata:
    // { return: { points: string[], times: number[], values: any[] } }
    const points = raw?.return?.points;
    const times = raw?.return?.times;
    const values = raw?.return?.values;
    if (Array.isArray(points) && Array.isArray(values)) {
        return points.map((p, i) => ({
            dataId: String(p ?? '').trim(),
            value: values[i],
            ts: Array.isArray(times) ? times[i] : undefined,
            status: undefined,
        })).filter(x => !!x.dataId);
    }
    const list = raw?.return?.data ?? raw?.data ?? raw;
    if (Array.isArray(list)) {
        return list
            .map((x) => ({
            dataId: String(x?.dataId ?? x?.DataID ?? x?.dataID ?? x?.DataId ?? x?.id ?? '').trim(),
            value: x?.value ?? x?.Value ?? x?.val ?? x?.data,
            ts: x?.ts ?? x?.time ?? x?.Time ?? x?.timestamp,
            status: x?.status ?? x?.Status,
        }))
            .filter((x) => !!x.dataId);
    }
    // object keyed by id
    if (list && typeof list === 'object') {
        return Object.entries(list).map(([k, v]) => ({
            dataId: String(k).trim(),
            value: v?.value ?? v,
            ts: v?.ts ?? v?.time,
            status: v?.status ?? v?.Status,
        })).filter(x => !!x.dataId);
    }
    return [];
}
async function refreshRealtimeOnce() {
    if (!authStore.token || !gatewayId.value)
        return;
    if (realtimeLoading.value)
        return;
    const ids = gatewayPointRows.value
        .map((x) => String(x?.dataId ?? '').trim())
        .filter(Boolean);
    if (!ids.length)
        return;
    realtimeLoading.value = true;
    try {
        const { data } = await fetchGatewayRealtime({
            token: authStore.token,
            gatewayId: gatewayId.value,
            formula: 'undefined',
            dataIds: ids,
        });
        if (data?.code != null && data.code !== 200) {
            throw new Error(data?.msg || 'realtime api failed');
        }
        const items = extractRealtimeItems(data);
        if (!items.length) {
            // 短暂空响应只更新在线状态，保留上一批值，避免列表闪回占位符。
            lastRealtimeSuccessAt.value = null;
            return;
        }
        const next = new Map(realtimeMap.value);
        let latestTsMs = null;
        for (const it of items) {
            next.set(it.dataId, {
                value: normalizeValue(it.value),
                ts: normalizeTs(it.ts),
                status: String(it.status ?? '').trim(),
            });
            const tsMs = parseRealtimeTsMs(it.ts);
            if (tsMs != null && (latestTsMs == null || tsMs > latestTsMs))
                latestTsMs = tsMs;
        }
        realtimeMap.value = next;
        if (!realtimeOrder.value.length && Array.isArray(data?.return?.points)) {
            realtimeOrder.value = data.return.points.map((x) => String(x ?? '').trim()).filter(Boolean);
        }
        pointListUpdatedAt.value = new Date().toLocaleString();
        // Prefer backend realtime timestamp to decide online/offline.
        lastRealtimeSuccessAt.value = latestTsMs ?? Date.now();
    }
    catch (e) {
        // 静默保留上一批数据，避免网络抖动时列表闪烁。
        lastRealtimeSuccessAt.value = null;
    }
    finally {
        realtimeLoading.value = false;
    }
}
async function refreshOnlineStatusOnce() {
    if (!authStore.token || !gatewayId.value)
        return;
    const ids = gatewayPointRows.value
        .map((x) => String(x?.dataId ?? '').trim())
        .filter(Boolean);
    if (!ids.length) {
        lastRealtimeSuccessAt.value = null;
        return;
    }
    try {
        const { data } = await fetchGatewayRealtime({
            token: authStore.token,
            gatewayId: gatewayId.value,
            formula: 'undefined',
            dataIds: ids,
        });
        if (data?.code != null && data.code !== 200) {
            throw new Error(data?.msg || 'realtime api failed');
        }
        const items = extractRealtimeItems(data);
        if (!items.length) {
            lastRealtimeSuccessAt.value = null;
            return;
        }
        let latestTsMs = null;
        for (const it of items) {
            const tsMs = parseRealtimeTsMs(it.ts);
            if (tsMs != null && (latestTsMs == null || tsMs > latestTsMs))
                latestTsMs = tsMs;
        }
        lastRealtimeSuccessAt.value = latestTsMs ?? Date.now();
    }
    catch {
        lastRealtimeSuccessAt.value = null;
    }
}
function stopRealtimePolling() {
    if (realtimeTimer != null) {
        window.clearInterval(realtimeTimer);
        realtimeTimer = null;
    }
}
function startRealtimePolling(immediate = true) {
    stopRealtimePolling();
    if (detailViewMode.value !== 'data-list')
        return;
    if (immediate) {
        if (globalRefreshEnabled.value)
            void refreshRealtimeOnce();
        else
            void refreshOnlineStatusOnce();
    }
    realtimeTimer = window.setInterval(() => {
        if (detailViewMode.value !== 'data-list') {
            stopRealtimePolling();
            return;
        }
        if (globalRefreshEnabled.value)
            void refreshRealtimeOnce();
        else
            void refreshOnlineStatusOnce();
    }, 3000);
}
function stopOnlineStatusPolling() {
    if (onlineStatusTimer != null) {
        window.clearInterval(onlineStatusTimer);
        onlineStatusTimer = null;
    }
}
function startOnlineStatusPolling() {
    stopOnlineStatusPolling();
    if (detailViewMode.value !== 'config')
        return;
    void refreshOnlineStatusOnce();
    onlineStatusTimer = window.setInterval(() => {
        if (detailViewMode.value === 'config')
            void refreshOnlineStatusOnce();
    }, 30000);
}
function toggleGlobalRefresh() {
    globalRefreshEnabled.value = !globalRefreshEnabled.value;
}
const defaultForm = () => ({
    name: '',
    gatewayId: gatewayId.value,
    type: 1,
    transport: '无',
    ttdcConfig: {
        moduleSoi: '~',
        moduleVer: '',
        moduleAdr: '',
        moduleCid1: '',
        moduleCid2: '',
    },
    list: [
        {
            name: '',
            transport: '',
            dataType: 1,
            formula: 'x',
            unformula: 'x',
            __pt: { addr: null, bytes: null, convType: null, bit: null },
        },
    ],
    templateId: undefined,
    templateVersion: undefined,
    parserProfile: '',
    protocolCode: '',
});
const form = ref(defaultForm());
const TTDC_CONTROL_CODE_42_LITHIUM_SULFUR = '42_锂硫电池';
const TTDC_CONTROL_CODE_42_SODIUM = '42_钠电';
const TTDC_CONTROL_CODE_42_SOC = '42_SOC';
const TTDC_CONTROL_CODE_44_LITHIUM_SULFUR = '44_锂硫电池';
const TTDC_CONTROL_CODE_44_SODIUM = '44_钠电';
const TTDC_CONTROL_CODE_44_COMMON = '44_通用';
function normalizeTtdcControlCode(value) {
    return String(value ?? '').trim();
}
function formatTtdcControlCode(value) {
    const code = normalizeTtdcControlCode(value);
    return code || '-';
}
/** 与“控制标识码”选择器同步，避免编辑状态下已有旧值被默认模板误覆盖。 */
const lastControlCodeSelect = ref('');
const controlCodeStr = computed(() => normalizeTtdcControlCode(form.value?.ttdcConfig?.moduleCid2));
const pointTransportMode = computed(() => {
    if (form.value.parserProfile === 'derived_algorithm' || form.value.parserProfile === 'algorithm_only')
        return 'cid42_soc';
    if (form.value.parserProfile === 'raw_fields') {
        if (controlCodeStr.value === '42')
            return 'cid42';
        if (controlCodeStr.value === '44')
            return 'cid44';
    }
    if (form.value.parserProfile === 'cid42_lithium_sulfur')
        return 'cid42';
    if (form.value.parserProfile === 'cid42_sodium')
        return 'cid42_sodium';
    if (form.value.parserProfile === 'cid44_common')
        return 'cid44';
    if (controlCodeStr.value === TTDC_CONTROL_CODE_42_SOC)
        return 'cid42_soc';
    if (controlCodeStr.value === '42')
        return 'cid42';
    if (controlCodeStr.value === '44')
        return 'cid44';
    if (controlCodeStr.value === TTDC_CONTROL_CODE_42_LITHIUM_SULFUR)
        return 'cid42';
    if (controlCodeStr.value === TTDC_CONTROL_CODE_42_SODIUM)
        return 'cid42_sodium';
    if (controlCodeStr.value === TTDC_CONTROL_CODE_44_LITHIUM_SULFUR ||
        controlCodeStr.value === TTDC_CONTROL_CODE_44_SODIUM ||
        controlCodeStr.value === TTDC_CONTROL_CODE_44_COMMON)
        return 'cid44';
    return 'raw';
});
function isCid42TransportMode(mode) {
    return mode === 'cid42' || mode === 'cid42_sodium';
}
const isCid42PointMode = computed(() => isCid42TransportMode(pointTransportMode.value));
/** 已绑定公共模板的通道不能变更协议身份，但仍允许修改名称和数据点配置。 */
const templateIdentityLocked = computed(() => drawerMode.value === 'edit' && Number(form.value.templateId ?? 0) > 0);
/** 42_SOC 的原始协议字段没有编辑意义。 */
const soc42FieldsLocked = computed(() => pointTransportMode.value === 'cid42_soc');
const protocolIdentityLocked = computed(() => soc42FieldsLocked.value || templateIdentityLocked.value);
const ttdcAddressRequiredForPrefix = computed(() => isCid42PointMode.value ||
    pointTransportMode.value === 'cid42_soc' ||
    pointTransportMode.value === 'cid44');
const lastAppliedTtdcAddressPrefix = ref('');
function getTtdcAddressPrefix() {
    const address = String(form.value?.ttdcConfig?.moduleAdr ?? '').trim();
    return address ? `${address}_` : '';
}
function getLegacyDeviceCodePrefix() {
    const code = String(form.value?.ttdcConfig?.moduleCid1 ?? '').trim();
    return code ? `${code}_` : '';
}
function applyPointPrefix(value, prefix) {
    let text = String(value ?? '').trim();
    if (!text)
        return '';
    const oldPrefix = lastAppliedTtdcAddressPrefix.value;
    if (oldPrefix && text.startsWith(oldPrefix)) {
        text = text.slice(oldPrefix.length);
    }
    const legacyDevicePrefix = getLegacyDeviceCodePrefix();
    if (legacyDevicePrefix && text.startsWith(legacyDevicePrefix)) {
        text = text.slice(legacyDevicePrefix.length);
    }
    if (prefix && text.startsWith(prefix)) {
        text = text.slice(prefix.length);
    }
    return prefix ? `${prefix}${text}` : text;
}
function applyTtdcAddressPrefixToPoints() {
    const mode = pointTransportMode.value;
    if (!isCid42TransportMode(mode) && mode !== 'cid42_soc' && mode !== 'cid44') {
        lastAppliedTtdcAddressPrefix.value = '';
        return;
    }
    const prefix = getTtdcAddressPrefix();
    for (const row of form.value.list) {
        if (mode === 'cid42_soc') {
            row.soc42Id = applyPointPrefix(row.soc42Id, prefix);
        }
        if (row.dataId)
            row.dataId = applyPointPrefix(row.dataId, prefix);
        row.name = applyPointPrefix(row.name, prefix);
    }
    lastAppliedTtdcAddressPrefix.value = prefix;
}
function createSoc42PlaceholderPoints() {
    return [{ name: '', transport: '', dataType: 1, formula: 'x', unformula: 'x', soc42Id: '', __pt: { addr: null, bytes: null, convType: null, bit: null } }];
}
function formatSoc42PointId(row) {
    const sid = String(row?.soc42Id ?? '').trim();
    if (sid)
        return sid;
    const dataId = String(row?.dataId ?? row?.dataID ?? '').trim();
    if (dataId)
        return dataId;
    const id = row?.id;
    if (id != null && id !== '')
        return String(id);
    const tr = String(row?.transport ?? '').trim();
    if (tr)
        return tr;
    return '-';
}
/** 将接口返回的数据点转为表单行；42_SOC 时从 dataId / transport / id 还原 soc42Id。 */
function mapApiPointRowForForm(x, moduleCid2) {
    const cid2 = normalizeTtdcControlCode(moduleCid2);
    const row = {
        id: x?.id,
        name: String(x?.name ?? ''),
        transport: String(x?.transport ?? ''),
        dataType: Number(x?.dataType ?? 1),
        formula: String(x?.formula ?? 'x'),
        unformula: String(x?.unformula ?? 'x'),
        dataId: String(x?.dataId ?? ''),
        algorithmKey: x?.algorithmKey == null ? null : String(x.algorithmKey),
        __pt: { addr: null, bytes: null, convType: null, bit: null },
    };
    if (cid2 === TTDC_CONTROL_CODE_42_SOC) {
        row.soc42Id =
            String(x?.dataId ?? x?.dataID ?? '').trim() ||
                String(x?.soc42Id ?? '').trim() ||
                String(x?.transport ?? '').trim() ||
                (x?.id != null && x?.id !== '' ? String(x.id) : '');
    }
    return row;
}
function isFormulaLocked(row) {
    // 控制标识码 44 且进制转换类型为十六进制转二进制时，formula/unformula 固定为 x。
    return pointTransportMode.value === 'cid44' && Number(row?.__pt?.convType) === TTDC_CONV_HEX_TO_BIN;
}
function enforceLockedFormula(row) {
    if (!row)
        return;
    if (!isFormulaLocked(row))
        return;
    row.formula = 'x';
    row.unformula = 'x';
}
function ensurePointParts(row) {
    if (row && !row.__pt) {
        row.__pt = { addr: null, bytes: null, convType: null, bit: null };
    }
}
function parsePartsFromTransport(mode, transport) {
    if (mode === 'cid42_soc') {
        return { addr: null, bytes: null, convType: null, bit: null };
    }
    const raw = String(transport ?? '').trim();
    const seg = raw ? raw.split(/\s+/g) : [];
    const nOrNull = (v) => {
        const n = Number(String(v ?? '').trim());
        return Number.isFinite(n) ? n : null;
    };
    const conv42 = (v) => (isTtdcDecimalConversion(v) ? v : null);
    const conv44 = (v) => (isTtdcDecimalConversion(v) || v === TTDC_CONV_HEX_TO_BIN ? v : null);
    if (isCid42TransportMode(mode)) {
        return {
            addr: nOrNull(seg[0]),
            bytes: nOrNull(seg[1]),
            convType: conv42(nOrNull(seg[2])),
            bit: null,
        };
    }
    if (mode === 'cid44') {
        const fourth = nOrNull(seg[3]);
        const fifth = nOrNull(seg[4]);
        const isTakeFlag = (v) => v === 0 || v === 1;
        let bit = fourth;
        if (seg.length >= 5) {
            // 若第 4 段是是否取位标识，则第 5 段是位序号；否则尝试第 5 段作为标识。
            if (isTakeFlag(fourth))
                bit = fourth === 0 ? -1 : fifth;
            else if (isTakeFlag(fifth))
                bit = fifth === 0 ? -1 : fourth;
        }
        return {
            addr: nOrNull(seg[0]),
            bytes: nOrNull(seg[1]),
            convType: conv44(nOrNull(seg[2])),
            bit,
        };
    }
    return { addr: null, bytes: null, convType: null, bit: null };
}
function joinTransportFromParts(mode, parts) {
    if (mode === 'cid42_soc')
        return '';
    if (isCid42TransportMode(mode)) {
        if (parts.addr == null || parts.bytes == null || parts.convType == null)
            return '';
        return [parts.addr, parts.bytes, parts.convType].join(' ');
    }
    if (mode === 'cid44') {
        if (parts.addr == null || parts.bytes == null || parts.convType == null || parts.bit == null)
            return '';
        return [parts.addr, parts.bytes, parts.convType, parts.bit].join(' ');
    }
    return '';
}
function syncPointTransport(row) {
    if (!row)
        return;
    if (pointTransportMode.value === 'raw' || pointTransportMode.value === 'cid42_soc')
        return;
    ensurePointParts(row);
    row.transport = joinTransportFromParts(pointTransportMode.value, row.__pt);
}
function initPointPartsForForm() {
    const mode = pointTransportMode.value;
    if (mode === 'raw' || mode === 'cid42_soc')
        return;
    for (const row of form.value.list) {
        ensurePointParts(row);
        const next = parsePartsFromTransport(mode, row.transport);
        row.__pt = next;
        // 控制标识码 42：选择十六进制转十进制时，数据类型固定为浮点型。
        if (isCid42TransportMode(mode) && isTtdcDecimalConversion(next?.convType)) {
            row.dataType = 1;
        }
        if (mode === 'cid44' && (isTtdcDecimalConversion(next?.convType) || next?.convType === TTDC_CONV_HEX_TO_BIN)) {
            row.dataType = 1;
            if (isTtdcDecimalConversion(next.convType)) {
                row.__pt.bit = -1;
            }
            else if (next.convType === TTDC_CONV_HEX_TO_BIN && Number(row.__pt.bit) === -1) {
                row.__pt.bit = 0;
            }
        }
        const joined = joinTransportFromParts(mode, next);
        if (joined)
            row.transport = joined;
        enforceLockedFormula(row);
    }
}
function getBitOptions(row) {
    const bytes = Number(row?.__pt?.bytes ?? 0);
    const count = Number.isFinite(bytes) && bytes > 0 ? Math.min(bytes * 8, 256) : 0;
    if (Number(row?.__pt?.convType) === TTDC_CONV_HEX_TO_BIN) {
        return Array.from({ length: count }, (_, i) => i);
    }
    // value: -1 不取位；0=取位1，1=取位2...
    return [-1, ...Array.from({ length: count }, (_, i) => i)];
}
function formatDataType(value) {
    const n = Number(value);
    if (n === 1)
        return '浮点型';
    if (n === 2)
        return '字符串';
    return String(value ?? '-');
}
function formatBitFromTransport(transport) {
    const seg = String(transport ?? '').trim().split(/\s+/g).filter(Boolean);
    if (!seg.length)
        return '-';
    const n = (v) => {
        const x = Number(String(v ?? '').trim());
        return Number.isFinite(x) ? x : null;
    };
    const isTakeFlag = (v) => v === 0 || v === 1;
    const fourth = n(seg[3]);
    const fifth = n(seg[4]);
    let bit = fourth;
    if (seg.length >= 5) {
        if (isTakeFlag(fourth))
            bit = fourth === 0 ? -1 : fifth;
        else if (isTakeFlag(fifth))
            bit = fifth === 0 ? -1 : fourth;
    }
    if (bit == null)
        return '-';
    if (bit === -1)
        return '不取位';
    return `取位${bit + 1}`;
}
function mapTemplatePointForForm(x) {
    const bit = x?.bitIndex == null ? null : Number(x.bitIndex);
    const transport = String(x?.transport || (bit == null
        ? `${Number(x?.address ?? 0)} ${Number(x?.byteCount ?? 0)} ${Number(x?.convertType ?? 0)}`
        : `${Number(x?.address ?? 0)} ${Number(x?.byteCount ?? 0)} ${Number(x?.convertType ?? 0)} ${bit}`));
    return {
        name: String(x?.name ?? ''),
        dataId: String(x?.baseDataId ?? ''),
        soc42Id: String(x?.baseDataId ?? ''),
        algorithmKey: x?.algorithmKey == null ? null : String(x.algorithmKey),
        transport,
        dataType: Number(x?.dataType ?? 1),
        formula: String(x?.formula ?? 'x'),
        unformula: String(x?.unformula ?? 'x'),
        __pt: { addr: Number(x?.address ?? 0), bytes: Number(x?.byteCount ?? 0), convType: Number(x?.convertType ?? 0), bit },
    };
}
function formatConversionFromTransport(transport) {
    const type = String(transport ?? '').trim().split(/\s+/g).filter(Boolean)[2];
    return formatTtdcConversionType(type);
}
function isCid42ControlCode(value) {
    const code = normalizeTtdcControlCode(value);
    return code === '42' || code === TTDC_CONTROL_CODE_42_LITHIUM_SULFUR || code === TTDC_CONTROL_CODE_42_SODIUM;
}
function isCid44ControlCode(value) {
    const code = normalizeTtdcControlCode(value);
    return code === '44' || code === TTDC_CONTROL_CODE_44_LITHIUM_SULFUR || code === TTDC_CONTROL_CODE_44_SODIUM || code === TTDC_CONTROL_CODE_44_COMMON;
}
function onCid42ConvTypeChange(row) {
    ensurePointParts(row);
    if (isCid42PointMode.value && isTtdcDecimalConversion(row?.__pt?.convType)) {
        row.dataType = 1;
    }
    syncPointTransport(row);
}
function onCid44ConvTypeChange(row) {
    ensurePointParts(row);
    if (pointTransportMode.value !== 'cid44') {
        syncPointTransport(row);
        return;
    }
    if (isTtdcDecimalConversion(row?.__pt?.convType) || row?.__pt?.convType === TTDC_CONV_HEX_TO_BIN) {
        row.dataType = 1;
    }
    // 44 且十六进制转十进制：取位固定为不取位。
    if (isTtdcDecimalConversion(row?.__pt?.convType)) {
        row.__pt.bit = -1;
    }
    else if (row?.__pt?.convType === TTDC_CONV_HEX_TO_BIN && Number(row?.__pt?.bit) === -1) {
        // 十六进制转二进制时，默认取位1（值 0）。
        row.__pt.bit = 0;
    }
    syncPointTransport(row);
    enforceLockedFormula(row);
}
function backToList() {
    const from = String(route.query.from || '');
    if (from && from.startsWith('/')) {
        router.push(from);
        return;
    }
    router.back();
}
async function onTemplateSelectChange(value) {
    const templateId = Number(value ?? 0);
    if (!templateId || !authStore.token)
        return;
    templateLoading.value = true;
    try {
        const response = await fetchTtdcTemplateDetail(authStore.token, templateId);
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '模板详情加载失败');
        const detail = response.data?.return?.data;
        if (!detail)
            throw new Error('模板详情为空');
        form.value.templateId = Number(detail.id);
        form.value.templateVersion = Number(detail.version);
        form.value.parserProfile = String(detail.parserProfile ?? '');
        form.value.protocolCode = String(detail.protocolCode ?? '');
        form.value.ttdcConfig.moduleCid2 = normalizeTtdcControlCode(detail.machineCid2);
        form.value.list = (detail.points ?? []).filter((item) => item.enabled !== false).map(mapTemplatePointForForm);
        lastControlCodeSelect.value = form.value.ttdcConfig.moduleCid2;
        lastAppliedTtdcAddressPrefix.value = '';
        resetPointPage();
        applyTtdcAddressPrefixToPoints();
        initPointPartsForForm();
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || error?.message || '模板加载失败');
    }
    finally {
        templateLoading.value = false;
    }
}
async function loadTemplateOptions() {
    if (!authStore.token)
        return;
    try {
        const response = await fetchTtdcTemplateOptions(authStore.token, {
            gatewayId: gatewayId.value,
            moduleAdr: String(form.value?.ttdcConfig?.moduleAdr ?? '').trim(),
        });
        templateOptions.value = response.data?.return?.data ?? [];
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '公共模板选项加载失败');
    }
}
function openAddDrawer() {
    drawerMode.value = 'add';
    form.value = defaultForm();
    lastAppliedTtdcAddressPrefix.value = '';
    lastControlCodeSelect.value = '';
    inlineContextAdr.value = '';
    // 与详情互斥：打开新增时收起详情。
    inlineDetailVisible.value = false;
    detailModule.value = null;
    // 与修改互斥：打开新增时收起修改。
    inlineEditVisible.value = false;
    inlineAddVisible.value = true;
    resetPointPage();
    initPointPartsForForm();
}
function openEditDrawer(moduleItem) {
    inlineContextAdr.value = getModuleAdr(moduleItem);
    drawerMode.value = 'edit';
    form.value = {
        id: moduleItem?.id,
        name: String(moduleItem?.name ?? ''),
        gatewayId: String(moduleItem?.gatewayId ?? gatewayId.value),
        type: Number(moduleItem?.type ?? 1),
        transport: String(moduleItem?.transport ?? '无'),
        ttdcConfig: {
            id: moduleItem?.ttdcConfig?.id,
            moduleSoi: String(moduleItem?.ttdcConfig?.moduleSoi ?? '').trim() || '~',
            moduleVer: String(moduleItem?.ttdcConfig?.moduleVer ?? ''),
            moduleAdr: String(moduleItem?.ttdcConfig?.moduleAdr ?? ''),
            moduleCid1: String(moduleItem?.ttdcConfig?.moduleCid1 ?? ''),
            moduleCid2: normalizeTtdcControlCode(moduleItem?.ttdcConfig?.moduleCid2),
        },
        list: (() => {
            const cid2 = normalizeTtdcControlCode(moduleItem?.ttdcConfig?.moduleCid2);
            const mapped = Array.isArray(moduleItem?.list)
                ? moduleItem.list.map((x) => mapApiPointRowForForm(x, cid2))
                : [];
            if (cid2 === TTDC_CONTROL_CODE_42_SOC && !mapped.length)
                return createSoc42PlaceholderPoints();
            return mapped;
        })(),
        templateId: Number(moduleItem?.templateId ?? 0) || undefined,
        templateVersion: Number(moduleItem?.templateVersion ?? 0) || undefined,
        parserProfile: String(moduleItem?.parserProfile ?? ''),
        protocolCode: String(moduleItem?.protocolCode ?? ''),
    };
    inlineAddVisible.value = false;
    inlineDetailVisible.value = false;
    detailModule.value = null;
    inlineEditVisible.value = true;
    resetPointPage();
    lastControlCodeSelect.value = normalizeTtdcControlCode(moduleItem?.ttdcConfig?.moduleCid2);
    lastAppliedTtdcAddressPrefix.value = '';
    initPointPartsForForm();
    applyTtdcAddressPrefixToPoints();
    openAdrGroup(getModuleAdr(moduleItem));
}
function toggleDetail(moduleItem) {
    inlineContextAdr.value = getModuleAdr(moduleItem);
    // 与新增互斥：打开详情时收起新增
    inlineAddVisible.value = false;
    // 与修改互斥：打开详情时收起修改
    inlineEditVisible.value = false;
    if (detailModule.value?.id && moduleItem?.id && detailModule.value.id === moduleItem.id && inlineDetailVisible.value) {
        inlineDetailVisible.value = false;
        detailModule.value = null;
        return;
    }
    detailModule.value = moduleItem;
    inlineDetailVisible.value = true;
    openAdrGroup(getModuleAdr(moduleItem));
}
watch(() => form.value.ttdcConfig.moduleAdr, (nextAdr, previousAdr) => {
    applyTtdcAddressPrefixToPoints();
    const adrChanged = String(nextAdr ?? '').trim().toUpperCase() !== String(previousAdr ?? '').trim().toUpperCase();
    if (adrChanged && !templateIdentityLocked.value) {
        form.value.templateId = undefined;
        form.value.templateVersion = undefined;
    }
    void loadTemplateOptions();
});
watch(() => form.value.list.length, () => clampPointPage());
watch(() => globalRefreshEnabled.value, () => startRealtimePolling(), { immediate: true });
watch(() => detailViewMode.value, async (mode) => {
    if (mode !== 'data-list') {
        stopRealtimePolling();
        realtimeListContentReady.value = false;
        startOnlineStatusPolling();
        return;
    }
    stopOnlineStatusPolling();
    realtimeListContentReady.value = false;
    try {
        await refreshRealtimeOnce();
    }
    finally {
        realtimeListContentReady.value = true;
    }
    startRealtimePolling(false);
});
function closeInlineDetail() {
    inlineDetailVisible.value = false;
    detailModule.value = null;
    inlineContextAdr.value = '';
}
function addPoint() {
    if (pointTransportMode.value === 'cid42_soc') {
        form.value.list.push({
            name: '',
            transport: '',
            dataType: 1,
            formula: 'x',
            unformula: 'x',
            soc42Id: '',
            __pt: { addr: null, bytes: null, convType: null, bit: null },
        });
        goToLastPointPage();
        return;
    }
    const row = {
        name: '',
        transport: '',
        dataType: 1,
        formula: 'x',
        unformula: 'x',
    };
    if (pointTransportMode.value !== 'raw') {
        ensurePointParts(row);
        row.__pt = {
            addr: null,
            bytes: null,
            convType: null,
            bit: null,
        };
    }
    form.value.list.push(row);
    goToLastPointPage();
}
async function removePoint(row) {
    const index = form.value.list.indexOf(row);
    if (index < 0)
        return;
    const point = form.value.list[index];
    if (!point)
        return;
    // 修改模式且点位有 id：调用后端 deletepoint。
    if (drawerMode.value === 'edit' && point.id) {
        if (!authStore.token) {
            ElMessage.error('未登录或登录已失效');
            return;
        }
        try {
            const body = {
                id: form.value.id,
                name: form.value.name.trim(),
                gatewayId: form.value.gatewayId.trim(),
                type: Number(form.value.type),
                transport: form.value.transport.trim(),
                ttdcConfig: { ...form.value.ttdcConfig },
                list: [{ id: point.id }],
            };
            const { data } = await deleteGatewayPoint({
                token: authStore.token,
                gatewayId: form.value.gatewayId.trim(),
                body,
            });
            if (data?.code !== 200) {
                ElMessage.error(data?.msg || '删除数据点失败');
                return;
            }
            ElMessage.success(data?.msg || '删除成功');
            // 先同步当前编辑状态，确保界面立即变化。
            form.value.list.splice(index, 1);
            await loadGatewayModules();
            // 编辑中的模块按后端最新结果回填，避免新增后删除时前后端状态不同步。
            const latestModule = gatewayModules.value.find(item => Number(item?.id) === Number(form.value.id));
            const cid2Reload = normalizeTtdcControlCode(form.value?.ttdcConfig?.moduleCid2);
            if (latestModule) {
                form.value.list = Array.isArray(latestModule?.list)
                    ? latestModule.list.map((x) => mapApiPointRowForForm(x, cid2Reload))
                    : [];
            }
            if (detailModule.value?.id && form.value.id && detailModule.value.id === form.value.id) {
                detailModule.value = {
                    ...detailModule.value,
                    list: form.value.list.map(item => ({ ...item })),
                };
            }
            return;
        }
        catch (error) {
            ElMessage.error(error?.response?.data?.msg || '删除数据点请求失败');
            return;
        }
    }
    // 新增模式或未落库点位：仅前端删除。
    form.value.list.splice(index, 1);
}
async function loadGatewayDetail() {
    if (!authStore.token || !gatewayId.value)
        return;
    loading.value = true;
    try {
        const { data } = await fetchGatewayOne({
            token: authStore.token,
            gatewayId: gatewayId.value,
        });
        if (data?.code === 200) {
            const one = data?.return?.data ?? {};
            if (abortIfRouteMismatch(one, gatewayId.value)) {
                return;
            }
            gateway.value = one;
            await loadGatewayModules();
            startOnlineStatusPolling();
            return;
        }
        ElMessage.error(data?.msg || '查询网关详情失败');
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '网关详情请求失败');
    }
    finally {
        loading.value = false;
    }
}
async function loadGatewayModules() {
    if (!authStore.token || !gatewayId.value)
        return;
    try {
        const { data } = await fetchGatewayModules({
            token: authStore.token,
            gatewayId: gatewayId.value,
        });
        if (data?.code === 200) {
            const modules = data?.return?.data?.gatewayModules;
            gatewayModules.value = Array.isArray(modules) ? modules : [];
            pointListUpdatedAt.value = new Date().toLocaleString();
            return;
        }
        ElMessage.error(data?.msg || '查询模块通道失败');
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '模块通道请求失败');
    }
}
function validatePointList() {
    if (!Array.isArray(form.value.list) || form.value.list.length === 0) {
        ElMessage.warning('请至少添加一条数据点');
        return false;
    }
    if (pointTransportMode.value === 'cid42_soc') {
        const seen = new Set();
        for (let i = 0; i < form.value.list.length; i++) {
            const row = form.value.list[i];
            const rowNo = i + 1;
            const name = String(row?.name ?? '').trim();
            const dataType = Number(row?.dataType);
            const sid = String(row?.soc42Id ?? '').trim();
            const formula = String(row?.formula ?? '').trim();
            const unformula = String(row?.unformula ?? '').trim();
            if (!name) {
                ElMessage.warning(`第${rowNo}行：名称必填`);
                return false;
            }
            if (!Number.isFinite(dataType)) {
                ElMessage.warning(`第${rowNo}行：数据类型必填`);
                return false;
            }
            if (dataType !== 1 && dataType !== 2) {
                ElMessage.warning(`第${rowNo}行：数据类型无效`);
                return false;
            }
            if (!sid) {
                ElMessage.warning(`第${rowNo}行：ID 必填`);
                return false;
            }
            const key = sid.toLowerCase();
            if (seen.has(key)) {
                ElMessage.warning(`第${rowNo}行：数据点 ID 不能与其他行重复`);
                return false;
            }
            seen.add(key);
            if (!formula) {
                ElMessage.warning(`第${rowNo}行：formula 必填`);
                return false;
            }
            if (!unformula) {
                ElMessage.warning(`第${rowNo}行：unformula 必填`);
                return false;
            }
        }
        return true;
    }
    for (let i = 0; i < form.value.list.length; i++) {
        const row = form.value.list[i];
        const rowNo = i + 1;
        enforceLockedFormula(row);
        const name = String(row?.name ?? '').trim();
        const formula = String(row?.formula ?? '').trim();
        const unformula = String(row?.unformula ?? '').trim();
        const dataType = Number(row?.dataType);
        if (!name) {
            ElMessage.warning(`第${rowNo}行：名称必填`);
            return false;
        }
        if (!Number.isFinite(dataType)) {
            ElMessage.warning(`第${rowNo}行：数据类型必填`);
            return false;
        }
        if (isCid42PointMode.value && dataType !== 1) {
            ElMessage.warning(`第${rowNo}行：控制标识码 ${controlCodeStr.value} 时数据类型仅支持浮点型`);
            return false;
        }
        if (!formula) {
            ElMessage.warning(`第${rowNo}行：formula 必填`);
            return false;
        }
        if (!unformula) {
            ElMessage.warning(`第${rowNo}行：unformula 必填`);
            return false;
        }
        if (pointTransportMode.value === 'raw') {
            if (!String(row?.transport ?? '').trim()) {
                ElMessage.warning(`第${rowNo}行：transport 必填`);
                return false;
            }
            continue;
        }
        const pt = row?.__pt ?? {};
        if (pt.addr == null || Number.isNaN(Number(pt.addr))) {
            ElMessage.warning(`第${rowNo}行：地址必填`);
            return false;
        }
        if (pt.bytes == null || Number.isNaN(Number(pt.bytes))) {
            ElMessage.warning(`第${rowNo}行：字节数必填`);
            return false;
        }
        if (pt.convType == null || Number.isNaN(Number(pt.convType))) {
            ElMessage.warning(`第${rowNo}行：进制转换类型必填`);
            return false;
        }
        if (pointTransportMode.value === 'cid44') {
            if (pt.bit == null || Number.isNaN(Number(pt.bit))) {
                ElMessage.warning(`第${rowNo}行：取位必填`);
                return false;
            }
            if (isTtdcDecimalConversion(pt.convType) && Number(pt.bit) !== -1) {
                ElMessage.warning(`第${rowNo}行：十六进制转十进制时取位必须为不取位`);
                return false;
            }
            if (Number(pt.convType) === TTDC_CONV_HEX_TO_BIN && Number(pt.bit) === -1) {
                ElMessage.warning(`第${rowNo}行：十六进制转二进制时取位不能为不取位`);
                return false;
            }
        }
    }
    return true;
}
function validateModuleIdentityByAdr() {
    const adr = String(form.value?.ttdcConfig?.moduleAdr ?? '').trim();
    if (!adr)
        return true;
    if (drawerMode.value === 'edit') {
        const editingId = Number(form.value?.id ?? 0);
        const origin = gatewayModules.value.find((item) => Number(item?.id ?? 0) === editingId);
        if (origin) {
            const originAdr = String(origin?.ttdcConfig?.moduleAdr ?? '').trim();
            const originVer = String(origin?.ttdcConfig?.moduleVer ?? '').trim();
            const originCid1 = String(origin?.ttdcConfig?.moduleCid1 ?? '').trim();
            const originCid2 = normalizeTtdcControlCode(origin?.ttdcConfig?.moduleCid2);
            const currentVer = String(form.value?.ttdcConfig?.moduleVer ?? '').trim();
            const currentCid1 = String(form.value?.ttdcConfig?.moduleCid1 ?? '').trim();
            const currentCid2 = normalizeTtdcControlCode(form.value?.ttdcConfig?.moduleCid2);
            const unchangedTuple = adr === originAdr &&
                currentVer === originVer &&
                currentCid1 === originCid1 &&
                currentCid2 === originCid2;
            if (unchangedTuple)
                return true;
        }
    }
    const currentId = Number(form.value?.id ?? 0);
    const peers = gatewayModules.value.filter((item) => {
        const itemAdr = String(item?.ttdcConfig?.moduleAdr ?? '').trim();
        const itemId = Number(item?.id ?? 0);
        return itemAdr === adr && (!currentId || itemId !== currentId);
    });
    if (!peers.length)
        return true;
    const currentName = String(form.value?.name ?? '').trim();
    const duplicatedName = peers.some((item) => String(item?.name ?? '').trim() === currentName);
    if (duplicatedName) {
        ElMessage.warning(`TTDC地址 ${adr} 下，通道名称不能重复`);
        return false;
    }
    return true;
}
async function submitModuleForm() {
    if (!authStore.token) {
        ElMessage.error('未登录或登录已失效');
        return;
    }
    if (!form.value.name.trim()) {
        ElMessage.warning('请输入通道名称');
        return;
    }
    if (!form.value.gatewayId.trim()) {
        ElMessage.warning('缺少网关序列号');
        return;
    }
    form.value.ttdcConfig.moduleCid2 = normalizeTtdcControlCode(form.value.ttdcConfig.moduleCid2);
    if (ttdcAddressRequiredForPrefix.value &&
        !String(form.value.ttdcConfig.moduleAdr ?? '').trim()) {
        ElMessage.warning('请输入TTDC地址');
        return;
    }
    applyTtdcAddressPrefixToPoints();
    if (!validateModuleIdentityByAdr())
        return;
    if (!validatePointList())
        return;
    submitLoading.value = true;
    try {
        if (pointTransportMode.value !== 'raw' && pointTransportMode.value !== 'cid42_soc') {
            for (const row of form.value.list) {
                syncPointTransport(row);
                enforceLockedFormula(row);
            }
        }
        const isSoc42 = pointTransportMode.value === 'cid42_soc';
        const originalModule = drawerMode.value === 'edit'
            ? gatewayModules.value.find((item) => Number(item?.id ?? 0) === Number(form.value.id ?? 0))
            : undefined;
        const templateId = Number(form.value.templateId ?? originalModule?.templateId ?? 0);
        const templateVersion = Number(form.value.templateVersion ?? originalModule?.templateVersion ?? 0);
        const body = {
            name: form.value.name.trim(),
            title: form.value.name.trim(),
            gatewayId: form.value.gatewayId.trim(),
            type: Number(form.value.type),
            transport: form.value.transport.trim(),
            ttdcConfig: { ...form.value.ttdcConfig },
            list: form.value.list.map(item => {
                const base = {
                    ...(item.id ? { id: item.id } : {}),
                    name: item.name.trim(),
                    transport: isSoc42 ? '-' : item.transport.trim(),
                    dataType: Number(item.dataType),
                    formula: item.formula.trim(),
                    unformula: item.unformula.trim(),
                };
                if (isSoc42) {
                    base.dataId = String(item.soc42Id ?? '').trim();
                    base.algorithmKey = item.algorithmKey ?? null;
                }
                else if (item.dataId) {
                    base.dataId = String(item.dataId).trim();
                }
                return base;
            }),
        };
        if (drawerMode.value === 'edit') {
            body.id = form.value.id;
        }
        if (templateId) {
            body.templateId = templateId;
            body.templateVersion = templateVersion;
            body.parserProfile = form.value.parserProfile || String(originalModule?.parserProfile ?? '');
            body.protocolCode = form.value.protocolCode || String(originalModule?.protocolCode ?? '');
        }
        const { data } = drawerMode.value === 'edit'
            ? await updateGatewayModule({ token: authStore.token, body })
            : await addGatewayModule({ token: authStore.token, body });
        if (data?.code === 200) {
            ElMessage.success(data?.msg || (drawerMode.value === 'edit' ? '修改成功' : '添加成功'));
            if (drawerMode.value === 'edit') {
                drawerVisible.value = false;
                inlineEditVisible.value = false;
                inlineContextAdr.value = '';
            }
            else {
                inlineAddVisible.value = false;
            }
            void loadGatewayDetail();
            return;
        }
        ElMessage.error(data?.msg || (drawerMode.value === 'edit' ? '修改失败' : '添加失败'));
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '模块通道请求失败');
    }
    finally {
        submitLoading.value = false;
    }
}
function cancelInlineAdd() {
    inlineAddVisible.value = false;
    form.value = defaultForm();
    lastControlCodeSelect.value = '';
    inlineContextAdr.value = '';
    resetPointPage();
}
function cancelInlineEdit() {
    inlineEditVisible.value = false;
    form.value = defaultForm();
    lastControlCodeSelect.value = '';
    inlineContextAdr.value = '';
    resetPointPage();
}
async function onDeleteModule(moduleItem) {
    if (!authStore.token) {
        ElMessage.error('未登录或登录已失效');
        return;
    }
    const moduleId = Number(moduleItem?.id ?? 0);
    const gatewayIdValue = String(moduleItem?.gatewayId ?? gatewayId.value ?? '');
    if (!moduleId || !gatewayIdValue) {
        ElMessage.warning('缺少模块ID或网关序列号，无法删除');
        return;
    }
    try {
        await ElMessageBox.confirm('确定删除该模块通道吗？', '提示', {
            confirmButtonText: '删除',
            cancelButtonText: '取消',
            type: 'warning',
        });
    }
    catch {
        return;
    }
    try {
        const { data } = await deleteGatewayModule({
            token: authStore.token,
            gatewayId: gatewayIdValue,
            body: { id: moduleId },
        });
        if (data?.code === 200) {
            ElMessage.success(data?.msg || '删除成功');
            if (inlineDetailVisible.value && detailModule.value?.id === moduleId) {
                closeInlineDetail();
            }
            if (inlineEditVisible.value && Number(form.value.id) === moduleId) {
                cancelInlineEdit();
            }
            await loadGatewayModules();
            return;
        }
        ElMessage.error(data?.msg || '删除失败');
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '删除模块请求失败');
    }
}
function onDocumentClick(event) {
    if (!inlineAddVisible.value && !inlineEditVisible.value && !inlineDetailVisible.value)
        return;
    const target = event.target;
    if (!target)
        return;
    if (target.closest('.module-tile') ||
        target.closest('.module-detail-card') ||
        target.closest('.module-add-card') ||
        target.closest('.module-edit-card') ||
        target.closest('.el-select') ||
        target.closest('.el-select-dropdown') ||
        target.closest('.el-select-dropdown__item') ||
        target.closest('.el-popper') ||
        target.closest('.el-overlay') ||
        target.closest('.el-message-box') ||
        target.closest('.el-popconfirm') ||
        target.closest('.el-button') ||
        target.closest('.page-head')) {
        return;
    }
    // 其它空白处点击：收起详情/新增/修改
    if (inlineDetailVisible.value)
        closeInlineDetail();
    if (inlineAddVisible.value)
        cancelInlineAdd();
    if (inlineEditVisible.value)
        cancelInlineEdit();
}
onMounted(() => {
    syncThemeMode();
    themeObserver = new MutationObserver(() => syncThemeMode());
    themeObserver.observe(document.documentElement, {
        attributes: true,
        attributeFilter: ['class', 'data-theme'],
    });
    if (document.body) {
        themeObserver.observe(document.body, {
            attributes: true,
            attributeFilter: ['class', 'data-theme'],
        });
    }
    themeMediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    themeMediaQuery.addEventListener('change', syncThemeMode);
    void loadGatewayDetail();
    void loadTemplateOptions();
    document.addEventListener('click', onDocumentClick);
    statusTimer = window.setInterval(() => {
        statusNow.value = Date.now();
    }, 1000);
});
onUnmounted(() => {
    document.removeEventListener('click', onDocumentClick);
    stopRealtimePolling();
    stopOnlineStatusPolling();
    if (statusTimer != null) {
        window.clearInterval(statusTimer);
        statusTimer = null;
    }
    themeObserver?.disconnect();
    themeObserver = null;
    if (themeMediaQuery) {
        themeMediaQuery.removeEventListener('change', syncThemeMode);
        themeMediaQuery = null;
    }
});
</script>

<template>
  <main class="detail-page">
    <div class="page-head">
      <div class="left">
        <el-button plain :style="backButtonStyle" @click="backToList">返回</el-button>
        <div class="title-wrap">
          <h2>网关详情（TTDC）</h2>
          <p class="title-sub">BMS / TTDC 模块通道与数据点配置</p>
        </div>
      </div>
      <div class="page-actions">
        <div class="view-switch" :class="{ 'is-dark': isDarkTheme }" role="tablist" aria-label="详情视图切换">
          <button
            type="button"
            class="view-switch-btn"
            :class="{ 'is-active': detailViewMode === 'config' }"
            @click="detailViewMode = 'config'"
          >
            模块通道配置
          </button>
          <button
            type="button"
            class="view-switch-btn"
            :class="{ 'is-active': detailViewMode === 'data-list' }"
            @click="detailViewMode = 'data-list'"
          >
            网关数据列表
          </button>
        </div>
      </div>
    </div>

    <el-card v-loading="loading" class="base-card base-card--summary" shadow="never">
      <div class="summary-strip">
        <div class="summary-text">
          <div class="summary-label">网关序列号</div>
          <div class="summary-value">{{ gateway?.gatewayId || gatewayId }}</div>
        </div>
        <div class="summary-right">
          <div class="summary-status-badge" :class="{ 'is-offline': !gatewayOnline }">
            <span class="summary-status-flow" aria-hidden="true">
              <span class="summary-status-dot" />
              <span class="summary-status-track">
                <span class="summary-status-packet summary-status-packet--a" />
                <span class="summary-status-packet summary-status-packet--b" />
              </span>
              <span class="summary-status-platform">
                <span class="summary-status-platform-grid" />
              </span>
            </span>
            <span class="summary-status-main">
              <span class="summary-status-label">实时状态</span>
              <span class="summary-status-text">{{ gatewayOnlineText }}</span>
            </span>
          </div>
        </div>
      </div>
    </el-card>

    <RealtimePointListPanel
      v-if="detailViewMode === 'data-list'"
      :rows="gatewayPointRows"
      :loading="loading"
      :content-ready="realtimeListContentReady"
      :last-updated="pointListUpdatedAt"
      :gateway-id="gatewayId"
      :token="authStore.token || ''"
      :refresh-enabled="globalRefreshEnabled"
      @toggle-refresh="toggleGlobalRefresh"
    />

    <el-card v-else class="module-card">
      <template #header>
        <div class="module-head">
          <div class="module-head-left">
            <span>模块通道</span>
            <span class="module-head-tag">BMS采集配置</span>
          </div>
          <el-button type="primary" size="small" @click="openAddDrawer">添加模块通道</el-button>
        </div>
      </template>
      <div v-if="!gatewayModules.length" class="module-empty">
        <el-empty description="暂无模块通道，请点击右上角添加" :image-size="72" />
      </div>
      <div v-else class="module-groups">
        <section
          v-for="group in moduleGroupsByAdr"
          :key="group.adr"
          class="module-adr-group"
          :class="{ 'is-dark': isDarkTheme }"
        >
          <div class="module-adr-head" role="button" tabindex="0" @click="toggleAdrGroup(group.adr)">
            <div class="module-adr-left">
              <div class="module-adr-title">
                <span class="module-adr-caret" :class="{ 'is-collapsed': !isAdrGroupExpanded(group.adr) }" aria-hidden="true" />
                <span class="module-adr-k">TTDC 地址</span>
                <span class="module-adr-v">{{ group.adr }}</span>
              </div>
              <div
                v-if="!isAdrGroupExpanded(group.adr)"
                class="module-adr-names"
                :title="getGroupNames(group.items).join('、')"
              >
                <span
                  v-for="(nm, i) in getGroupNames(group.items).slice(0, 6)"
                  :key="`${group.adr}-nm-${i}`"
                  class="module-adr-chip"
                >
                  {{ nm }}
                </span>
                <span
                  v-if="getGroupNames(group.items).length > 6 && getGroupNames(group.items)[0] !== '-'"
                  class="module-adr-chip module-adr-chip--more"
                >
                  +{{ getGroupNames(group.items).length - 6 }}
                </span>
              </div>
            </div>
            <div class="module-adr-right">
              <div class="module-adr-count">通道数：{{ group.items.length }}</div>
            </div>
          </div>
          <transition name="adr-collapse">
            <div v-show="isAdrGroupExpanded(group.adr)" class="module-grid">
              <el-card
                v-for="(row, idx) in group.items"
                :key="row?.id ?? `${group.adr}-${idx}`"
                class="module-tile"
                shadow="never"
                @click="toggleDetail(row)"
              >
                <template #header>
                  <div class="tile-head">
                    <div class="tile-title-block">
                      <span class="tile-index" aria-hidden="true">{{ idx + 1 }}</span>
                      <span class="tile-name">{{ row?.name?.trim() ? row.name : '未命名通道' }}</span>
                    </div>
                    <div class="tile-actions" @click.stop>
                      <el-button size="small" text type="primary" @click.stop="openEditDrawer(row)">修改</el-button>
                      <el-button size="small" text type="danger" @click.stop="onDeleteModule(row)">删除</el-button>
                    </div>
                  </div>
                </template>
                <div class="tile-body">
                  <div class="tile-meta">
                    <div class="tile-field">
                      <span class="tile-k">TTDC 版本</span>
                      <span class="tile-v">{{ row?.ttdcConfig?.moduleVer ?? '-' }}</span>
                    </div>
                    <div class="tile-field">
                      <span class="tile-k">TTDC 地址</span>
                      <span class="tile-v">{{ row?.ttdcConfig?.moduleAdr ?? '-' }}</span>
                    </div>
                    <div class="tile-field">
                      <span class="tile-k">设备编码</span>
                      <span class="tile-v">{{ row?.ttdcConfig?.moduleCid1 ?? '-' }}</span>
                    </div>
                    <div class="tile-field">
                      <span class="tile-k">控制标识码</span>
                      <span class="tile-v">{{ formatTtdcControlCode(row?.ttdcConfig?.moduleCid2) }}</span>
                    </div>
                  </div>
                </div>
              </el-card>
            </div>
          </transition>

          <!-- 通道详情：展示在所属 TTDC 地址分组中 -->
          <el-card
            v-if="inlineDetailVisible && inlineContextAdr === group.adr"
            class="module-detail-card"
            shadow="never"
          >
            <template #header>
              <div class="tile-head">
                <span class="tile-name">通道详情：{{ detailModule?.name || '未命名通道' }}</span>
                <el-button size="small" link @click="closeInlineDetail">收起</el-button>
              </div>
            </template>

            <div class="detail-info two-col">
              <div>通道名称：{{ detailModule?.name || '-' }}</div>
              <div>网关序列号：{{ detailModule?.gatewayId || gatewayId }}</div>
              <div>TTDC SOI：{{ (detailModule?.ttdcConfig?.moduleSoi ?? '').trim() || '~' }}</div>
              <div>TTDC 版本：{{ detailModule?.ttdcConfig?.moduleVer ?? '-' }}</div>
              <div>TTDC 地址：{{ detailModule?.ttdcConfig?.moduleAdr ?? '-' }}</div>
              <div>设备编码：{{ detailModule?.ttdcConfig?.moduleCid1 ?? '-' }}</div>
              <div>控制标识码：{{ formatTtdcControlCode(detailModule?.ttdcConfig?.moduleCid2) }}</div>
            </div>

            <el-divider />
            <div class="detail-section-title">数据点列表</div>
            <el-table
              :data="detailModuleListSorted"
              border
              stripe
              style="width: 100%"
              :class="{ 'soc42-point-table': normalizeTtdcControlCode(detailModule?.ttdcConfig?.moduleCid2) === TTDC_CONTROL_CODE_42_SOC }"
              >
              <template v-if="normalizeTtdcControlCode(detailModule?.ttdcConfig?.moduleCid2) === TTDC_CONTROL_CODE_42_SOC">
                <el-table-column label="ID" width="220" align="center">
                  <template #default="{ row }">{{ formatSoc42PointId(row) }}</template>
                </el-table-column>
                <el-table-column prop="name" label="名称" min-width="160" show-overflow-tooltip />
                <el-table-column label="数据类型" width="228" align="center">
                  <template #default="{ row }">{{ formatDataType(row?.dataType) }}</template>
                </el-table-column>
              </template>
              <template v-else>
                <el-table-column prop="name" label="名称" min-width="160" />
                <template v-if="isCid42ControlCode(detailModule?.ttdcConfig?.moduleCid2)">
                  <el-table-column label="地址" min-width="120">
                    <template #default="{ row }">{{ String(row?.transport ?? '').trim().split(/\s+/g)[0] || '-' }}</template>
                  </el-table-column>
                  <el-table-column label="字节数" min-width="120">
                    <template #default="{ row }">{{ String(row?.transport ?? '').trim().split(/\s+/g)[1] || '-' }}</template>
                  </el-table-column>
                  <el-table-column label="进制转换类型" min-width="220">
                    <template #default="{ row }">
                      {{ formatConversionFromTransport(row?.transport) }}
                    </template>
                  </el-table-column>
                </template>
                <template v-else-if="isCid44ControlCode(detailModule?.ttdcConfig?.moduleCid2)">
                  <el-table-column label="地址" min-width="120">
                    <template #default="{ row }">{{ String(row?.transport ?? '').trim().split(/\s+/g)[0] || '-' }}</template>
                  </el-table-column>
                  <el-table-column label="字节数" min-width="120">
                    <template #default="{ row }">{{ String(row?.transport ?? '').trim().split(/\s+/g)[1] || '-' }}</template>
                  </el-table-column>
                  <el-table-column label="进制转换类型" min-width="220">
                    <template #default="{ row }">
                      {{ formatConversionFromTransport(row?.transport) }}
                    </template>
                  </el-table-column>
                  <el-table-column label="取位" min-width="120">
                    <template #default="{ row }">{{ formatBitFromTransport(row?.transport) }}</template>
                  </el-table-column>
                </template>
                <el-table-column v-else prop="transport" label="transport" min-width="200" />
                <el-table-column label="数据类型" min-width="120">
                  <template #default="{ row }">{{ formatDataType(row?.dataType) }}</template>
                </el-table-column>
                <el-table-column prop="formula" label="formula" min-width="140" />
                <el-table-column prop="unformula" label="unformula" min-width="140" />
              </template>
            </el-table>
          </el-card>

          <!-- 通道修改：展示在所属 TTDC 地址分组中 -->
          <el-card
            v-if="inlineEditVisible && inlineContextAdr === group.adr"
            class="module-edit-card"
            shadow="never"
          >
            <template #header>
              <div class="tile-head">
                <span class="tile-name">修改模块通道</span>
                <el-button size="small" link @click="cancelInlineEdit">收起</el-button>
              </div>
            </template>
            <el-form label-width="110px" label-position="right" class="module-edit-form">
              <el-form-item label="通道名称" required>
                <el-input v-model="form.name" placeholder="请输入通道名称" />
              </el-form-item>
              <el-form-item label="网关序列号" required>
                <el-input v-model="form.gatewayId" disabled />
              </el-form-item>
              <el-form-item label="TTDC SOI">
                <el-input v-model="form.ttdcConfig.moduleSoi" :disabled="protocolIdentityLocked" />
              </el-form-item>
              <el-form-item label="TTDC 版本">
                <el-input v-model="form.ttdcConfig.moduleVer" :disabled="protocolIdentityLocked" />
              </el-form-item>
              <el-form-item label="TTDC 地址" :required="ttdcAddressRequiredForPrefix">
                <el-input v-model="form.ttdcConfig.moduleAdr" :disabled="templateIdentityLocked" />
              </el-form-item>
              <el-form-item label="设备编码">
                <el-input v-model="form.ttdcConfig.moduleCid1" :disabled="protocolIdentityLocked" />
              </el-form-item>
              <el-form-item label="控制标识码">
                <el-select
                  v-model="form.templateId"
                  :loading="templateLoading"
                  :disabled="templateIdentityLocked"
                  teleported
                  placeholder="请选择公共模板"
                  style="width: 100%"
                  @change="onTemplateSelectChange"
                >
                  <el-option v-for="item in templateOptions" :key="item.id" :value="item.id" :label="`${item.groupName || ''} / ${item.templateName}`" />
                </el-select>
              </el-form-item>

              <div class="point-title">
                <span>数据点列表</span>
                <el-button size="small" type="primary" @click="addPoint">添加数据点</el-button>
              </div>
              <el-table :data="formListPaged" border style="width: 100%" :class="{ 'soc42-point-table': pointTransportMode === 'cid42_soc' }">
                <template v-if="pointTransportMode === 'cid42_soc'">
                  <el-table-column label="ID" width="220" align="center">
                    <template #default="{ row }">
                      <el-input v-model="row.soc42Id" class="soc42-id-input" placeholder="请输入 ID" clearable />
                    </template>
                  </el-table-column>
                  <el-table-column label="名称" min-width="160">
                    <template #default="{ row }"><el-input v-model="row.name" class="soc42-name-input" /></template>
                  </el-table-column>
                  <el-table-column label="数据类型" width="228" align="center">
                    <template #default="{ row }">
                      <el-select v-model="row.dataType" teleported placeholder="请选择类型" class="soc42-dtype-select">
                        <el-option :value="1" label="浮点型" />
                        <el-option :value="2" label="字符串" />
                      </el-select>
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="76" align="center">
                    <template #default="{ row }">
                      <el-button size="small" type="danger" @click.stop="removePoint(row)">删</el-button>
                    </template>
                  </el-table-column>
                </template>
                <template v-else>
                <el-table-column label="名称" min-width="140">
                  <template #default="{ row }"><el-input v-model="row.name" /></template>
                </el-table-column>
                <template v-if="isCid42PointMode">
                  <el-table-column label="地址" min-width="120">
                    <template #default="{ row }">
                      <el-input-number v-model="row.__pt.addr" :min="0" :controls="false" style="width: 100%" @change="syncPointTransport(row)" />
                    </template>
                  </el-table-column>
                  <el-table-column label="字节数" min-width="120">
                    <template #default="{ row }">
                      <el-input-number v-model="row.__pt.bytes" :min="1" :controls="false" style="width: 100%" @change="syncPointTransport(row)" />
                    </template>
                  </el-table-column>
                  <el-table-column label="进制转换类型" min-width="220">
                    <template #default="{ row }">
                      <el-select v-model="row.__pt.convType" teleported placeholder="请选择" style="width: 100%" @change="onCid42ConvTypeChange(row)">
                        <el-option :value="1" label="十六进制转有符号十进制" />
                        <el-option :value="3" label="十六进制转无符号十进制" />
                      </el-select>
                    </template>
                  </el-table-column>
                </template>
                <template v-else-if="pointTransportMode === 'cid44'">
                  <el-table-column label="地址" min-width="120">
                    <template #default="{ row }">
                      <el-input-number v-model="row.__pt.addr" :min="0" :controls="false" style="width: 100%" @change="syncPointTransport(row)" />
                    </template>
                  </el-table-column>
                  <el-table-column label="字节数" min-width="120">
                    <template #default="{ row }">
                      <el-input-number v-model="row.__pt.bytes" :min="1" :controls="false" style="width: 100%" @change="syncPointTransport(row)" />
                    </template>
                  </el-table-column>
                  <el-table-column label="进制转换类型" min-width="220">
                    <template #default="{ row }">
                      <el-select v-model="row.__pt.convType" teleported placeholder="请选择" style="width: 100%" @change="onCid44ConvTypeChange(row)">
                        <el-option :value="1" label="十六进制转有符号十进制" />
                        <el-option :value="2" label="十六进制转二进制" />
                        <el-option :value="3" label="十六进制转无符号十进制" />
                      </el-select>
                    </template>
                  </el-table-column>
                  <el-table-column label="取位" min-width="120">
                    <template #default="{ row }">
                      <el-select
                        v-model="row.__pt.bit"
                        teleported
                        placeholder="请选择"
                        style="width: 100%"
                        :disabled="row.__pt.convType !== TTDC_CONV_HEX_TO_BIN"
                        @change="syncPointTransport(row)"
                      >
                        <el-option v-for="n in getBitOptions(row)" :key="n" :value="n" :label="n === -1 ? '不取位' : `取位${n + 1}`" />
                      </el-select>
                    </template>
                  </el-table-column>
                </template>
                <el-table-column v-else label="transport" min-width="200">
                  <template #default="{ row }"><el-input v-model="row.transport" /></template>
                </el-table-column>
                <el-table-column label="数据类型" min-width="120">
                  <template #default="{ row }">
                    <el-select
                      v-if="isCid42PointMode || pointTransportMode === 'cid44'"
                      v-model="row.dataType"
                      teleported
                      placeholder="浮点型"
                      style="width: 100%"
                      disabled
                    >
                      <el-option :value="1" label="浮点型" />
                    </el-select>
                    <el-select v-else v-model="row.dataType" teleported placeholder="请选择类型" style="width: 100%">
                      <el-option :value="1" label="浮点型" />
                      <el-option :value="2" label="字符串" />
                    </el-select>
                  </template>
                </el-table-column>
                <el-table-column label="formula" min-width="120">
                  <template #default="{ row }"><el-input v-model="row.formula" :disabled="isFormulaLocked(row)" /></template>
                </el-table-column>
                <el-table-column label="unformula" min-width="120">
                  <template #default="{ row }"><el-input v-model="row.unformula" :disabled="isFormulaLocked(row)" /></template>
                </el-table-column>
                <el-table-column label="操作" width="80">
                  <template #default="{ row }">
                    <el-button size="small" type="danger" @click.stop="removePoint(row)">删</el-button>
                  </template>
                </el-table-column>
                </template>
              </el-table>

              <div v-if="pointPaginationVisible" class="point-pagination">
                <el-pagination
                  v-model:current-page="pointPage"
                  :page-size="POINT_PAGE_SIZE"
                  :total="formListSorted.length"
                  :pager-count="5"
                  layout="total, prev, pager, next"
                  background
                />
              </div>

              <div class="inline-footer">
                <el-button @click="cancelInlineEdit">取消</el-button>
                <el-button type="primary" :loading="submitLoading" @click="submitModuleForm">保存</el-button>
              </div>
            </el-form>
          </el-card>
        </section>
      </div>

      <el-card v-if="inlineAddVisible" class="module-add-card" shadow="never">
        <template #header>
          <div class="tile-head">
            <span class="tile-name">新增模块通道</span>
          </div>
        </template>
        <el-form label-width="110px" label-position="right" class="module-add-form two-col-form">
          <el-form-item label="通道名称" required>
            <el-input v-model="form.name" placeholder="请输入通道名称" />
          </el-form-item>
          <el-form-item label="网关序列号" required>
            <el-input v-model="form.gatewayId" disabled />
          </el-form-item>
          <el-form-item label="TTDC SOI">
            <el-input v-model="form.ttdcConfig.moduleSoi" :disabled="protocolIdentityLocked" />
          </el-form-item>
          <el-form-item label="TTDC 版本">
            <el-input v-model="form.ttdcConfig.moduleVer" :disabled="protocolIdentityLocked" />
          </el-form-item>
          <el-form-item label="TTDC 地址" :required="ttdcAddressRequiredForPrefix">
            <el-input v-model="form.ttdcConfig.moduleAdr" :disabled="templateIdentityLocked" />
          </el-form-item>
          <el-form-item label="设备编码">
            <el-input v-model="form.ttdcConfig.moduleCid1" :disabled="protocolIdentityLocked" />
          </el-form-item>
          <el-form-item label="控制标识码">
            <el-select
              v-model="form.templateId"
              :loading="templateLoading"
              :disabled="templateIdentityLocked"
              teleported
              placeholder="请选择公共模板"
              style="width: 100%"
              @change="onTemplateSelectChange"
            >
              <el-option v-for="item in templateOptions" :key="item.id" :value="item.id" :label="`${item.groupName || ''} / ${item.templateName}`" />
            </el-select>
          </el-form-item>

          <div class="point-title">
            <span>数据点列表</span>
            <el-button size="small" type="primary" @click="addPoint">添加数据点</el-button>
          </div>
          <el-table :data="formListPaged" border style="width: 100%" :class="{ 'soc42-point-table': pointTransportMode === 'cid42_soc' }">
            <template v-if="pointTransportMode === 'cid42_soc'">
              <el-table-column label="ID" width="220" align="center">
                <template #default="{ row }">
                  <el-input v-model="row.soc42Id" class="soc42-id-input" placeholder="请输入 ID" clearable />
                </template>
              </el-table-column>
              <el-table-column label="名称" min-width="160">
                <template #default="{ row }"><el-input v-model="row.name" class="soc42-name-input" /></template>
              </el-table-column>
              <el-table-column label="数据类型" width="228" align="center">
                <template #default="{ row }">
                  <el-select v-model="row.dataType" teleported placeholder="请选择类型" class="soc42-dtype-select">
                    <el-option :value="1" label="浮点型" />
                    <el-option :value="2" label="字符串" />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="76" align="center">
                <template #default="{ row }">
                  <el-button size="small" type="danger" @click.stop="removePoint(row)">删</el-button>
                </template>
              </el-table-column>
            </template>
            <template v-else>
            <el-table-column label="名称" min-width="140">
              <template #default="{ row }"><el-input v-model="row.name" /></template>
            </el-table-column>
            <template v-if="isCid42PointMode">
              <el-table-column label="地址" min-width="120">
                <template #default="{ row }">
                  <el-input-number v-model="row.__pt.addr" :min="0" :controls="false" style="width: 100%" @change="syncPointTransport(row)" />
                </template>
              </el-table-column>
              <el-table-column label="字节数" min-width="120">
                <template #default="{ row }">
                  <el-input-number v-model="row.__pt.bytes" :min="1" :controls="false" style="width: 100%" @change="syncPointTransport(row)" />
                </template>
              </el-table-column>
              <el-table-column label="进制转换类型" min-width="220">
                <template #default="{ row }">
                  <el-select v-model="row.__pt.convType" teleported placeholder="请选择" style="width: 100%" @change="onCid42ConvTypeChange(row)">
                    <el-option :value="1" label="十六进制转有符号十进制" />
                    <el-option :value="3" label="十六进制转无符号十进制" />
                  </el-select>
                </template>
              </el-table-column>
            </template>
            <template v-else-if="pointTransportMode === 'cid44'">
              <el-table-column label="地址" min-width="120">
                <template #default="{ row }">
                  <el-input-number v-model="row.__pt.addr" :min="0" :controls="false" style="width: 100%" @change="syncPointTransport(row)" />
                </template>
              </el-table-column>
              <el-table-column label="字节数" min-width="120">
                <template #default="{ row }">
                  <el-input-number v-model="row.__pt.bytes" :min="1" :controls="false" style="width: 100%" @change="syncPointTransport(row)" />
                </template>
              </el-table-column>
              <el-table-column label="进制转换类型" min-width="220">
                <template #default="{ row }">
                  <el-select v-model="row.__pt.convType" teleported placeholder="请选择" style="width: 100%" @change="onCid44ConvTypeChange(row)">
                    <el-option :value="1" label="十六进制转有符号十进制" />
                    <el-option :value="2" label="十六进制转二进制" />
                    <el-option :value="3" label="十六进制转无符号十进制" />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column label="取位" min-width="120">
                <template #default="{ row }">
                  <el-select
                    v-model="row.__pt.bit"
                    teleported
                    placeholder="请选择"
                    style="width: 100%"
                    :disabled="row.__pt.convType !== TTDC_CONV_HEX_TO_BIN"
                    @change="syncPointTransport(row)"
                  >
                    <el-option v-for="n in getBitOptions(row)" :key="n" :value="n" :label="n === -1 ? '不取位' : `取位${n + 1}`" />
                  </el-select>
                </template>
              </el-table-column>
            </template>
            <el-table-column v-else label="transport" min-width="200">
              <template #default="{ row }"><el-input v-model="row.transport" /></template>
            </el-table-column>
            <el-table-column label="数据类型" min-width="120">
              <template #default="{ row }">
                <el-select
                  v-if="isCid42PointMode || pointTransportMode === 'cid44'"
                  v-model="row.dataType"
                  teleported
                  placeholder="浮点型"
                  style="width: 100%"
                  disabled
                >
                  <el-option :value="1" label="浮点型" />
                </el-select>
                <el-select v-else v-model="row.dataType" teleported placeholder="请选择类型" style="width: 100%">
                  <el-option :value="1" label="浮点型" />
                  <el-option :value="2" label="字符串" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="formula" min-width="120">
              <template #default="{ row }"><el-input v-model="row.formula" :disabled="isFormulaLocked(row)" /></template>
            </el-table-column>
            <el-table-column label="unformula" min-width="120">
              <template #default="{ row }"><el-input v-model="row.unformula" :disabled="isFormulaLocked(row)" /></template>
            </el-table-column>
            <el-table-column label="操作" width="80">
              <template #default="{ row }">
                <el-button size="small" type="danger" @click.stop="removePoint(row)">删</el-button>
              </template>
            </el-table-column>
            </template>
          </el-table>

          <div v-if="pointPaginationVisible" class="point-pagination">
            <el-pagination
              v-model:current-page="pointPage"
              :page-size="POINT_PAGE_SIZE"
              :total="formListSorted.length"
              :pager-count="5"
              layout="total, prev, pager, next"
              background
            />
          </div>

          <div class="inline-footer">
            <el-button @click="cancelInlineAdd">取消</el-button>
            <el-button type="primary" :loading="submitLoading" @click="submitModuleForm">保存</el-button>
          </div>
        </el-form>
      </el-card>
    </el-card>

    <el-drawer
      v-model="drawerVisible"
      :title="drawerMode === 'edit' ? '修改模块通道' : '添加模块通道'"
      direction="rtl"
      size="680px"
    >
      <el-form label-width="110px" label-position="right">
        <el-form-item label="通道名称" required>
          <el-input v-model="form.name" placeholder="请输入通道名称" />
        </el-form-item>
        <el-form-item label="网关序列号" required>
          <el-input v-model="form.gatewayId" disabled />
        </el-form-item>
        <el-form-item label="类型">
          <el-input-number v-model="form.type" :min="1" :max="9" :disabled="soc42FieldsLocked" />
        </el-form-item>
        <el-form-item label="传输配置">
          <el-input v-model="form.transport" placeholder="例如: undefined true 192.168.1.149 9998 4" :disabled="soc42FieldsLocked" />
        </el-form-item>
        <el-form-item label="TTDC SOI">
          <el-input v-model="form.ttdcConfig.moduleSoi" :disabled="protocolIdentityLocked" />
        </el-form-item>
        <el-form-item label="TTDC 版本">
          <el-input v-model="form.ttdcConfig.moduleVer" :disabled="protocolIdentityLocked" />
        </el-form-item>
        <el-form-item label="TTDC 地址" :required="ttdcAddressRequiredForPrefix">
          <el-input v-model="form.ttdcConfig.moduleAdr" :disabled="templateIdentityLocked" />
        </el-form-item>
        <el-form-item label="设备编码">
          <el-input v-model="form.ttdcConfig.moduleCid1" :disabled="protocolIdentityLocked" />
        </el-form-item>
        <el-form-item label="控制标识码">
          <el-select
            v-model="form.templateId"
            :loading="templateLoading"
            :disabled="templateIdentityLocked"
            teleported
            placeholder="请选择公共模板"
            style="width: 100%"
            @change="onTemplateSelectChange"
          >
            <el-option v-for="item in templateOptions" :key="item.id" :value="item.id" :label="`${item.groupName || ''} / ${item.templateName}`" />
          </el-select>
        </el-form-item>

        <div class="point-title">
          <span>数据点列表</span>
          <el-button size="small" type="primary" @click="addPoint">添加数据点</el-button>
        </div>
        <el-table :data="formListPaged" border style="width: 100%" :class="{ 'soc42-point-table': pointTransportMode === 'cid42_soc' }">
          <template v-if="pointTransportMode === 'cid42_soc'">
            <el-table-column label="ID" width="220" align="center">
              <template #default="{ row }">
                <el-input v-model="row.soc42Id" class="soc42-id-input" placeholder="请输入 ID" clearable />
              </template>
            </el-table-column>
            <el-table-column label="名称" min-width="160">
              <template #default="{ row }"><el-input v-model="row.name" class="soc42-name-input" /></template>
            </el-table-column>
            <el-table-column label="数据类型" width="228" align="center">
              <template #default="{ row }">
                <el-select v-model="row.dataType" teleported placeholder="请选择类型" class="soc42-dtype-select">
                  <el-option :value="1" label="浮点型" />
                  <el-option :value="2" label="字符串" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="76" align="center">
              <template #default="{ row }">
                <el-button size="small" type="danger" @click.stop="removePoint(row)">删</el-button>
              </template>
            </el-table-column>
          </template>
          <template v-else>
          <el-table-column label="名称" min-width="140">
            <template #default="{ row }"><el-input v-model="row.name" /></template>
          </el-table-column>
          <template v-if="isCid42PointMode">
            <el-table-column label="地址" min-width="120">
              <template #default="{ row }">
                <el-input-number v-model="row.__pt.addr" :min="0" :controls="false" style="width: 100%" @change="syncPointTransport(row)" />
              </template>
            </el-table-column>
            <el-table-column label="字节数" min-width="120">
              <template #default="{ row }">
                <el-input-number v-model="row.__pt.bytes" :min="1" :controls="false" style="width: 100%" @change="syncPointTransport(row)" />
              </template>
            </el-table-column>
            <el-table-column label="进制转换类型" min-width="220">
              <template #default="{ row }">
                <el-select v-model="row.__pt.convType" teleported placeholder="请选择" style="width: 100%" @change="onCid42ConvTypeChange(row)">
                  <el-option :value="1" label="十六进制转有符号十进制" />
                  <el-option :value="3" label="十六进制转无符号十进制" />
                </el-select>
              </template>
            </el-table-column>
          </template>
          <template v-else-if="pointTransportMode === 'cid44'">
            <el-table-column label="地址" min-width="120">
              <template #default="{ row }">
                <el-input-number v-model="row.__pt.addr" :min="0" :controls="false" style="width: 100%" @change="syncPointTransport(row)" />
              </template>
            </el-table-column>
            <el-table-column label="字节数" min-width="120">
              <template #default="{ row }">
                <el-input-number v-model="row.__pt.bytes" :min="1" :controls="false" style="width: 100%" @change="syncPointTransport(row)" />
              </template>
            </el-table-column>
            <el-table-column label="进制转换类型" min-width="220">
              <template #default="{ row }">
                <el-select v-model="row.__pt.convType" teleported placeholder="请选择" style="width: 100%" @change="onCid44ConvTypeChange(row)">
                  <el-option :value="1" label="十六进制转有符号十进制" />
                  <el-option :value="2" label="十六进制转二进制" />
                  <el-option :value="3" label="十六进制转无符号十进制" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="取位" min-width="120">
              <template #default="{ row }">
                <el-select
                  v-model="row.__pt.bit"
                  teleported
                  placeholder="请选择"
                  style="width: 100%"
                  :disabled="row.__pt.convType !== TTDC_CONV_HEX_TO_BIN"
                  @change="syncPointTransport(row)"
                >
                  <el-option v-for="n in getBitOptions(row)" :key="n" :value="n" :label="n === -1 ? '不取位' : `取位${n + 1}`" />
                </el-select>
              </template>
            </el-table-column>
          </template>
          <el-table-column v-else label="transport" min-width="200">
            <template #default="{ row }"><el-input v-model="row.transport" /></template>
          </el-table-column>
          <el-table-column label="数据类型" min-width="120">
            <template #default="{ row }">
              <el-select
                v-if="isCid42PointMode || pointTransportMode === 'cid44'"
                v-model="row.dataType"
                teleported
                placeholder="浮点型"
                style="width: 100%"
                disabled
              >
                <el-option :value="1" label="浮点型" />
              </el-select>
              <el-select v-else v-model="row.dataType" teleported placeholder="请选择类型" style="width: 100%">
                <el-option :value="1" label="浮点型" />
                <el-option :value="2" label="字符串" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="formula" min-width="120">
            <template #default="{ row }"><el-input v-model="row.formula" :disabled="isFormulaLocked(row)" /></template>
          </el-table-column>
          <el-table-column label="unformula" min-width="120">
            <template #default="{ row }"><el-input v-model="row.unformula" :disabled="isFormulaLocked(row)" /></template>
          </el-table-column>
          <el-table-column label="操作" width="80">
            <template #default="{ row }">
              <el-button size="small" type="danger" @click.stop="removePoint(row)">删</el-button>
            </template>
          </el-table-column>
          </template>
        </el-table>

        <div v-if="pointPaginationVisible" class="point-pagination">
          <el-pagination
            v-model:current-page="pointPage"
            :page-size="POINT_PAGE_SIZE"
            :total="formListSorted.length"
            :pager-count="5"
            layout="total, prev, pager, next"
            background
          />
        </div>
      </el-form>

      <template #footer>
        <div class="drawer-footer">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="submitLoading" @click="submitModuleForm">确定</el-button>
        </div>
      </template>
    </el-drawer>
  </main>
</template>

<style scoped>
.detail-page {
  width: 100%;
  min-height: 100%;
  padding: 20px 16px 28px;
  background: #f0f2f5;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  padding: 16px 20px;
  border-radius: 8px;
  background: #fff;
  border: 1px solid #f0f0f0;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.03), 0 2px 8px rgba(0, 0, 0, 0.06);
}

.left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.title-wrap {
  display: grid;
  gap: 4px;
  min-width: 0;
}

h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
  letter-spacing: 0.2px;
}

.title-sub {
  margin: 0;
  font-size: 13px;
  color: rgba(0, 0, 0, 0.45);
  line-height: 1.4;
}

.page-actions {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.view-switch {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  border-radius: 8px;
  background: #f5f7fa;
  border: 1px solid #e6ebf2;
}

.view-switch-btn {
  border: 0;
  background: transparent;
  color: rgba(0, 0, 0, 0.55);
  font-size: 13px;
  font-weight: 600;
  line-height: 1;
  height: 32px;
  padding: 0 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.view-switch-btn:hover {
  color: rgba(0, 0, 0, 0.78);
  background: #eef2f7;
}

.view-switch-btn.is-active {
  color: #1677ff;
  background: #fff;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
}

/* 组件内主题感知：深色主题时按钮使用深色视觉 */
.view-switch.is-dark {
  background: rgba(11, 24, 40, 0.92);
  border-color: rgba(71, 95, 124, 0.58);
  border-radius: 10px;
  padding: 3px;
  box-shadow:
    inset 0 1px 0 rgba(148, 163, 184, 0.1),
    0 6px 14px rgba(2, 6, 23, 0.3);
}

.view-switch.is-dark .view-switch-btn {
  color: #95a9c2;
  border-radius: 7px;
  height: 30px;
  padding: 0 13px;
}

.view-switch.is-dark .view-switch-btn:hover {
  color: #e4edf8;
  background: rgba(73, 101, 135, 0.32);
}

.view-switch.is-dark .view-switch-btn.is-active {
  color: #9ddcff;
  background: linear-gradient(180deg, rgba(20, 42, 66, 0.98) 0%, rgba(12, 29, 47, 0.98) 100%);
  box-shadow:
    inset 0 0 0 1px rgba(96, 186, 255, 0.52),
    0 2px 8px rgba(2, 6, 23, 0.35);
}

.base-card,
.module-card,
.module-add-card,
.module-edit-card,
.module-detail-card {
  border-radius: 8px;
  border: 1px solid #f0f0f0;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.03), 0 2px 8px rgba(0, 0, 0, 0.06);
}

.base-card--summary :deep(.el-card__body) {
  padding: 0;
}

.summary-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  min-height: 88px;
  padding: 18px 22px;
  background: linear-gradient(105deg, #e6f7ff 0%, #f6ffed 35%, #fff 70%);
  border-radius: 0 0 7px 7px;
}

.summary-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.summary-label {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.45);
}

.summary-value {
  font-size: 20px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.02em;
  word-break: break-all;
  line-height: 1.35;
}

.summary-status {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.summary-status-v {
  color: #52c41a;
  font-weight: 600;
}

.summary-status-v.is-offline {
  color: #ff4d4f;
}

.summary-right {
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.summary-status-badge {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 12px;
  padding: 9px 14px 9px 10px;
  border-radius: 14px;
  background:
    linear-gradient(180deg, rgba(82, 196, 26, 0.28) 0%, rgba(82, 196, 26, 0.14) 100%),
    rgba(255, 255, 255, 0.68);
  border: 1px solid rgba(82, 196, 26, 0.56);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.7),
    0 8px 18px rgba(34, 197, 94, 0.26),
    0 14px 28px rgba(15, 23, 42, 0.16),
    0 0 0 1px rgba(134, 239, 172, 0.22);
  backdrop-filter: blur(4px);
  overflow: hidden;
}

.summary-status-badge::after {
  content: '';
  position: absolute;
  top: -30%;
  left: -45%;
  width: 38%;
  height: 160%;
  background: linear-gradient(90deg, rgba(255, 255, 255, 0), rgba(255, 255, 255, 0.32), rgba(255, 255, 255, 0));
  transform: skewX(-18deg);
  animation: statusShimmer 3.2s ease-in-out infinite;
}

.summary-status-flow {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-width: 62px;
}

.summary-status-main {
  display: inline-flex;
  flex-direction: column;
  line-height: 1.05;
  gap: 3px;
}

.summary-status-label {
  font-size: 10px;
  color: rgba(34, 108, 26, 0.88);
  letter-spacing: 0.06em;
}

.summary-status-text {
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.03em;
  color: #1f7a15;
}

.summary-status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #69d426;
  box-shadow:
    0 0 0 0 rgba(105, 212, 38, 0.52),
    0 0 14px rgba(105, 212, 38, 0.55);
  animation: statusPulse 1.6s ease-out infinite;
}

.summary-status-track {
  position: relative;
  width: 26px;
  height: 2px;
  border-radius: 999px;
  background: linear-gradient(90deg, rgba(82, 196, 26, 0.16), rgba(82, 196, 26, 0.58), rgba(82, 196, 26, 0.16));
  overflow: hidden;
}

.summary-status-packet {
  position: absolute;
  top: 50%;
  left: -7px;
  width: 6px;
  height: 6px;
  margin-top: -3px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.98) 0%, rgba(190, 242, 100, 0.96) 70%, rgba(82, 196, 26, 0) 100%);
  box-shadow: 0 0 12px rgba(132, 204, 22, 0.8);
  animation: dataPacketMove 1.8s linear infinite;
}

.summary-status-packet--b {
  animation-delay: 0.9s;
}

.summary-status-platform {
  position: relative;
  width: 14px;
  height: 14px;
  border-radius: 4px;
  border: 1px solid rgba(76, 175, 80, 0.68);
  background: linear-gradient(180deg, rgba(82, 196, 26, 0.3) 0%, rgba(82, 196, 26, 0.1) 100%);
  box-shadow:
    inset 0 0 0 1px rgba(220, 252, 231, 0.42),
    0 0 8px rgba(132, 204, 22, 0.28);
}

.summary-status-platform-grid {
  position: absolute;
  inset: 2px;
  border-radius: 2px;
  background:
    linear-gradient(90deg, rgba(255, 255, 255, 0.28) 1px, transparent 1px) 0 0 / 4px 100%,
    linear-gradient(0deg, rgba(255, 255, 255, 0.24) 1px, transparent 1px) 0 0 / 100% 4px;
  opacity: 0.8;
}

.summary-status-badge.is-offline {
  background:
    linear-gradient(180deg, rgba(255, 77, 79, 0.2) 0%, rgba(255, 77, 79, 0.12) 100%),
    rgba(255, 255, 255, 0.68);
  border-color: rgba(255, 77, 79, 0.5);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.7),
    0 8px 18px rgba(239, 68, 68, 0.2),
    0 14px 26px rgba(30, 41, 59, 0.16);
}

.summary-status-badge.is-offline .summary-status-dot {
  background: #ff4d4f;
  box-shadow:
    0 0 0 0 rgba(255, 77, 79, 0.45),
    0 0 10px rgba(255, 77, 79, 0.42);
  animation-duration: 2.4s;
}

.summary-status-badge.is-offline .summary-status-track {
  background: linear-gradient(90deg, rgba(255, 77, 79, 0.12), rgba(255, 77, 79, 0.36), rgba(255, 77, 79, 0.12));
}

.summary-status-badge.is-offline .summary-status-packet {
  background: radial-gradient(circle, rgba(255, 255, 255, 0.9) 0%, rgba(252, 165, 165, 0.78) 70%, rgba(255, 77, 79, 0) 100%);
  box-shadow: 0 0 8px rgba(255, 77, 79, 0.46);
  animation-duration: 2.6s;
}

.summary-status-badge.is-offline .summary-status-platform {
  border-color: rgba(255, 77, 79, 0.56);
  background: linear-gradient(180deg, rgba(252, 165, 165, 0.24) 0%, rgba(255, 77, 79, 0.1) 100%);
  box-shadow: inset 0 0 0 1px rgba(254, 226, 226, 0.28);
}

.summary-status-badge.is-offline .summary-status-text {
  color: #b42318;
}

.summary-status-badge.is-offline .summary-status-label {
  color: rgba(180, 35, 24, 0.75);
}

@keyframes statusPulse {
  0% {
    box-shadow: 0 0 0 0 rgba(82, 196, 26, 0.45);
  }
  70% {
    box-shadow: 0 0 0 9px rgba(82, 196, 26, 0);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(82, 196, 26, 0);
  }
}

@keyframes dataPacketMove {
  0% {
    transform: translateX(0) scale(0.82);
    opacity: 0;
  }
  20% {
    opacity: 0.95;
  }
  85% {
    opacity: 0.9;
  }
  100% {
    transform: translateX(33px) scale(1.08);
    opacity: 0;
  }
}

@keyframes statusShimmer {
  0% {
    left: -45%;
  }
  50% {
    left: 120%;
  }
  100% {
    left: 120%;
  }
}

.base-card :deep(.el-card__body),
.module-card :deep(.el-card__body),
.module-add-card :deep(.el-card__body),
.module-edit-card :deep(.el-card__body),
.module-detail-card :deep(.el-card__body) {
  padding: 16px;
}

.point-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: 10px 0 10px;
}

.point-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.module-empty {
  padding: 32px 0 24px;
}

.module-add-card,
.module-detail-card,
.module-edit-card {
  margin-top: 16px;
}

.module-grid {
  display: grid;
  /* 与原先每列最大宽度一致；网格可居中，最后一行不足列数时从左侧排列 */
  grid-template-columns: repeat(
    auto-fill,
    minmax(min(100%, var(--tile-card-max)), var(--tile-card-max))
  );
  justify-content: center;
  gap: 14px;
  width: 100%;
  align-items: start;
  --tile-label-w: 5.5rem;
  --tile-card-max: 340px;
}

.module-groups {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.module-adr-group {
  border: 1px solid rgba(24, 144, 255, 0.22);
  background: linear-gradient(180deg, rgba(24, 144, 255, 0.04) 0%, rgba(255, 255, 255, 0.92) 42%, #ffffff 100%);
  border-radius: 12px;
  padding: 12px 12px 14px;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04), 0 6px 18px rgba(15, 23, 42, 0.06);
}

.module-adr-group.is-dark {
  border: 1px solid rgba(71, 95, 124, 0.62);
  background:
    radial-gradient(120% 100% at 50% 0%, rgba(56, 189, 248, 0.08) 0%, rgba(56, 189, 248, 0) 58%),
    linear-gradient(180deg, rgba(16, 30, 47, 0.96) 0%, rgba(11, 23, 38, 0.96) 100%);
  box-shadow:
    inset 0 1px 0 rgba(148, 163, 184, 0.1),
    0 8px 18px rgba(2, 6, 23, 0.3);
}

.module-adr-group.is-dark .module-adr-head {
  color: #dbe9ff;
}

.module-adr-group.is-dark .module-adr-title {
  padding: 2px 8px;
  border-radius: 8px;
  background: rgba(20, 42, 66, 0.68);
  border: 1px solid rgba(88, 175, 255, 0.28);
}

.module-adr-group.is-dark .module-adr-k {
  color: #8ea8c6;
}

.module-adr-group.is-dark .module-adr-v {
  color: #e5f0ff;
}

.module-adr-group.is-dark .module-adr-caret::before {
  border-left-color: rgba(159, 188, 220, 0.92);
}

.module-adr-group.is-dark .module-adr-count {
  color: #b8cce6;
  background: rgba(20, 42, 66, 0.72);
  border: 1px solid rgba(88, 175, 255, 0.3);
}

.module-adr-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 4px 6px 10px;
  cursor: pointer;
  user-select: none;
}

.module-adr-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.module-adr-right {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.module-adr-names {
  max-width: min(760px, 56vw);
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
  padding: 0;
  border: 0;
  background: transparent;
  border-radius: 0;
}

.module-adr-chip {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  font-size: 12px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.65);
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(24, 144, 255, 0.18);
  border-radius: 999px;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 0 1 auto;
}

.module-adr-chip--more {
  color: rgba(24, 144, 255, 0.85);
  background: rgba(24, 144, 255, 0.08);
  border-color: rgba(24, 144, 255, 0.22);
}

.module-adr-head:focus-visible {
  outline: 2px solid rgba(24, 144, 255, 0.35);
  outline-offset: 2px;
  border-radius: 10px;
}

.module-adr-title {
  display: inline-flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}

.module-adr-caret {
  display: inline-block;
  width: 14px;
  height: 14px;
  position: relative;
  top: 1px;
}

.module-adr-caret::before {
  content: '';
  position: absolute;
  left: 3px;
  top: 4px;
  width: 0;
  height: 0;
  border-left: 5px solid rgba(0, 0, 0, 0.55);
  border-top: 4px solid transparent;
  border-bottom: 4px solid transparent;
  transform: rotate(90deg);
  transform-origin: 50% 50%;
  transition: transform 0.18s ease;
}

.module-adr-caret.is-collapsed::before {
  transform: rotate(0deg);
}

.module-adr-k {
  font-size: 12px;
  font-weight: 500;
  color: rgba(0, 0, 0, 0.55);
}

.module-adr-v {
  font-size: 16px;
  font-weight: 700;
  color: rgba(0, 0, 0, 0.88);
  letter-spacing: 0.02em;
}

.module-adr-count {
  flex-shrink: 0;
  font-size: 12px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.45);
  padding: 4px 10px;
  border-radius: 999px;
  background: rgba(24, 144, 255, 0.08);
  border: 1px solid rgba(24, 144, 255, 0.18);
}

.adr-collapse-enter-active,
.adr-collapse-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
  will-change: opacity, transform;
}

.adr-collapse-enter-from,
.adr-collapse-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

/* module-tile 与 el-card 同根节点，边框和圆角写在当前选择器 */
.module-tile {
  cursor: pointer;
  width: 100%;
  min-width: 0;
  height: auto;
  display: flex;
  flex-direction: column;
  border: 1px solid #e8ecf0;
  border-radius: 10px;
  background: #fff;
  box-shadow:
    0 1px 2px rgba(15, 23, 42, 0.04),
    0 4px 20px rgba(15, 23, 42, 0.06);
  overflow: hidden;
  transition:
    transform 0.22s cubic-bezier(0.34, 1.56, 0.64, 1),
    border-color 0.2s ease,
    box-shadow 0.22s ease;
}

.module-tile :deep(.el-card__header) {
  padding: 0;
  border-bottom: none;
  background: linear-gradient(180deg, #f8fbff 0%, #ffffff 55%);
}

.module-tile :deep(.el-card__body) {
  flex: 0 1 auto;
  padding: 0;
  display: flex;
  flex-direction: column;
  background: #fafbfc;
}

.module-tile:hover {
  transform: translateY(-2px);
  border-color: rgba(24, 144, 255, 0.45);
  box-shadow:
    0 2px 8px rgba(24, 144, 255, 0.08),
    0 12px 32px rgba(15, 23, 42, 0.08);
}

.tile-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px 8px 10px;
  border-bottom: 1px solid rgba(24, 144, 255, 0.1);
  position: relative;
}

.tile-head::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 20px;
  border-radius: 0 3px 3px 0;
  background: linear-gradient(180deg, #1890ff 0%, #69c0ff 100%);
}

.tile-title-block {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding-left: 8px;
}

.tile-index {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 5px;
  font-size: 11px;
  font-weight: 700;
  color: #fff;
  letter-spacing: -0.02em;
  background: linear-gradient(135deg, #1890ff 0%, #40a9ff 100%);
  border-radius: 4px;
  box-shadow: 0 1px 4px rgba(24, 144, 255, 0.3);
}

.tile-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

/* 卡片操作区：删除按钮带轻微红色阴影 */
.tile-actions :deep(.el-button--danger.is-text) {
  box-shadow: 0 2px 8px rgba(255, 77, 79, 0.18);
}

.tile-actions :deep(.el-button--danger.is-text:hover),
.tile-actions :deep(.el-button--danger.is-text:focus-visible) {
  box-shadow: 0 4px 12px rgba(255, 77, 79, 0.28);
}

.tile-name {
  font-size: 14px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.88);
  letter-spacing: 0.01em;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tile-body {
  flex: 0 1 auto;
  padding: 8px 10px 10px;
}

.tile-meta {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  width: 100%;
  column-gap: 8px;
  row-gap: 6px;
  align-content: start;
}

.tile-field {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  padding: 5px 8px;
  background: #fff;
  border: 1px solid #ebeef2;
  border-radius: 6px;
  transition:
    border-color 0.18s ease,
    background 0.18s ease,
    box-shadow 0.18s ease;
}

.module-tile:hover .tile-field {
  border-color: #d3e8ff;
  box-shadow: 0 1px 3px rgba(24, 144, 255, 0.06);
}

.tile-k {
  flex: 0 0 var(--tile-label-w);
  width: var(--tile-label-w);
  font-size: 11px;
  line-height: 1.35;
  font-weight: 500;
  color: rgba(0, 0, 0, 0.45);
  white-space: nowrap;
}

.tile-v {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.35;
  color: rgba(0, 0, 0, 0.88);
  word-break: break-all;
  font-variant-numeric: tabular-nums;
}

.detail-section-title {
  margin: 0 0 12px;
  font-size: 15px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
}

.detail-info {
  display: grid;
  gap: 12px;
  font-size: 14px;
  color: rgba(0, 0, 0, 0.85);
  padding: 4px 0 8px;
}

.detail-info.two-col {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 24px;
}

.module-edit-form,
.module-add-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 18px;
}

.module-edit-form :deep(.el-form-item),
.module-add-form :deep(.el-form-item) {
  margin-bottom: 12px;
}

.module-add-form :deep(.el-form-item__content) {
  max-width: 280px;
  flex: 0 0 280px;
}

.module-edit-form :deep(.el-form-item__content) {
  max-width: 280px;
  flex: 0 0 280px;
}

.module-add-form :deep(.el-form-item__content .el-input),
.module-add-form :deep(.el-form-item__content .el-input-number),
.module-add-form :deep(.el-form-item__content .el-select) {
  width: 100%;
}

.module-edit-form :deep(.el-form-item__content .el-input),
.module-edit-form :deep(.el-form-item__content .el-input-number),
.module-edit-form :deep(.el-form-item__content .el-select) {
  width: 100%;
}

.inline-footer {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.module-card :deep(.el-card__header),
.module-add-card :deep(.el-card__header),
.module-edit-card :deep(.el-card__header),
.module-detail-card :deep(.el-card__header) {
  background: #fafafa;
  border-bottom: 1px solid #f0f0f0;
}

.module-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  color: rgba(0, 0, 0, 0.85);
  font-weight: 600;
  font-size: 15px;
}

.module-head-left {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}

.module-head-tag {
  font-size: 12px;
  font-weight: 500;
  color: #389e0d;
  background: #f6ffed;
  border: 1px solid #b7eb8f;
  border-radius: 4px;
  padding: 3px 10px;
}

.module-add-card :deep(.el-form-item),
.module-edit-card :deep(.el-form-item) {
  margin-bottom: 14px;
}

.point-title,
.module-edit-form :deep(.el-table),
.module-add-form :deep(.el-table),
.point-pagination,
.module-edit-form .inline-footer,
.module-add-form .inline-footer {
  grid-column: 1 / -1;
}

.module-detail-card :deep(.el-table),
.module-add-card :deep(.el-table),
.module-edit-card :deep(.el-table) {
  border-radius: 8px;
  overflow: hidden;
}

.module-detail-card :deep(.el-table th),
.module-add-card :deep(.el-table th),
.module-edit-card :deep(.el-table th) {
  background: #fafafa !important;
}

.module-detail-card :deep(.el-table td),
.module-add-card :deep(.el-table td),
.module-edit-card :deep(.el-table td) {
  vertical-align: middle;
}

.module-detail-card :deep(.el-table .cell),
.module-add-card :deep(.el-table .cell),
.module-edit-card :deep(.el-table .cell) {
  display: flex;
  align-items: center;
  min-height: 34px;
}

/* 深色主题下，网关详情页保持浅色风格 */
:global(html[data-theme='dark']) .detail-page {
  background: #f0f2f5 !important;
}

:global(html[data-theme='dark']) .base-card,
:global(html[data-theme='dark']) .module-card,
:global(html[data-theme='dark']) .module-add-card,
:global(html[data-theme='dark']) .module-edit-card,
:global(html[data-theme='dark']) .module-detail-card {
  background: #ffffff !important;
  border-color: #f0f0f0 !important;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.03), 0 2px 8px rgba(0, 0, 0, 0.06) !important;
}

:global(html[data-theme='dark']) h2,
:global(html[data-theme='dark']) .summary-value,
:global(html[data-theme='dark']) .module-head,
:global(html[data-theme='dark']) .tile-name,
:global(html[data-theme='dark']) .detail-section-title,
:global(html[data-theme='dark']) .detail-info,
:global(html[data-theme='dark']) .tile-v {
  color: rgba(0, 0, 0, 0.88) !important;
}

:global(html[data-theme='dark']) .title-sub,
:global(html[data-theme='dark']) .summary-label,
:global(html[data-theme='dark']) .tile-k {
  color: rgba(0, 0, 0, 0.45) !important;
}

/* 指定三块：网关序列号 / 模块通道标题 / 卡片字段字体，与浅色样式一致 */
:global(html[data-theme='dark']) .summary-strip .summary-label {
  color: rgba(0, 0, 0, 0.45) !important;
}

:global(html[data-theme='dark']) .summary-strip .summary-value {
  color: rgba(0, 0, 0, 0.85) !important;
  font-weight: 600 !important;
  text-shadow: none !important;
}

:global(html[data-theme='dark']) .module-head {
  color: rgba(0, 0, 0, 0.85) !important;
  font-weight: 600 !important;
}

:global(html[data-theme='dark']) .module-head-tag {
  color: #389e0d !important;
  font-weight: 500 !important;
}

:global(html[data-theme='dark']) .tile-k {
  color: rgba(0, 0, 0, 0.45) !important;
  font-weight: 500 !important;
}

:global(html[data-theme='dark']) .tile-v,
:global(html[data-theme='dark']) .tile-name {
  color: rgba(0, 0, 0, 0.88) !important;
  font-weight: 600 !important;
}

:global(html[data-theme='dark']) .summary-strip {
  background: linear-gradient(105deg, #e6f7ff 0%, #f6ffed 35%, #fff 70%) !important;
}

:global(html[data-theme='dark']) .module-card :deep(.el-card__header),
:global(html[data-theme='dark']) .module-add-card :deep(.el-card__header),
:global(html[data-theme='dark']) .module-edit-card :deep(.el-card__header),
:global(html[data-theme='dark']) .module-detail-card :deep(.el-card__header) {
  background: #fafafa !important;
  border-bottom: 1px solid #f0f0f0 !important;
}

:global(html[data-theme='dark']) .module-head-tag {
  color: #389e0d !important;
  background: #f6ffed !important;
  border-color: #b7eb8f !important;
}

:global(html[data-theme='dark']) .module-tile,
:global(html[data-theme='dark']) .module-tile :deep(.el-card),
:global(html[data-theme='dark']) .module-tile :deep(.el-card__header),
:global(html[data-theme='dark']) .module-tile :deep(.el-card__body),
:global(html[data-theme='dark']) .tile-field {
  background: #ffffff !important;
  border-color: #e8ecf0 !important;
}

/* 提高优先级：锁定三块内容与浅色主题一致。 */
:global(html[data-theme='dark']) .detail-page .base-card.base-card--summary {
  background: #ffffff !important;
  border-color: #f0f0f0 !important;
}

/* 深色主题下，页头（返回旁标题区）使用深色样式 */
:global(html[data-theme='dark']) .detail-page .page-head {
  background: #0b1220 !important;
  border-color: #22324a !important;
  box-shadow: 0 8px 22px rgba(2, 6, 23, 0.42) !important;
}

:global(html[data-theme='dark']) .detail-page .page-head h2 {
  color: #f1f5f9 !important;
}

/* 深色主题下单独确保“网关详情（TTDC）”标题为白色 */
:global(html[data-theme='dark']) .detail-page .page-head .title-wrap > h2 {
  color: #ffffff !important;
}

:global(html[data-theme='dark']) .detail-page .page-head .title-sub {
  color: #a9b8ce !important;
}

:global(html[data-theme='dark']) .detail-page .page-head .el-button {
  border-color: #3a4f6c !important;
  color: #e2e8f0 !important;
  background: #132033 !important;
}

:global(html[data-theme='dark']) .detail-page .page-head .left .el-button.is-plain {
  background: linear-gradient(180deg, #16314d 0%, #12263d 100%) !important;
  border-color: #4f7fab !important;
  color: #eaf4ff !important;
  box-shadow:
    inset 0 0 0 1px rgba(143, 212, 255, 0.2),
    0 2px 8px rgba(2, 6, 23, 0.28) !important;
}

:global(html[data-theme='dark']) .detail-page .page-head .left .el-button.is-plain:hover {
  background: linear-gradient(180deg, #1b3a5c 0%, #15304d 100%) !important;
  border-color: #6a9cc8 !important;
  color: #ffffff !important;
}

:global(html[data-theme='dark']) .detail-page .page-head .left .el-button.is-plain:active {
  background: #10273f !important;
}

/* 深色主题下，TTDC 地址分组容器改为深色底 */
:global(html[data-theme='dark']) .detail-page .module-adr-group,
:global(html.dark) .detail-page .module-adr-group,
:global(body.dark) .detail-page .module-adr-group,
:global([data-theme='dark']) .detail-page .module-adr-group,
:global(.theme-dark) .detail-page .module-adr-group,
:global(.dark-theme) .detail-page .module-adr-group {
  border: 1px solid rgba(71, 95, 124, 0.62) !important;
  background:
    radial-gradient(120% 100% at 50% 0%, rgba(56, 189, 248, 0.08) 0%, rgba(56, 189, 248, 0) 58%),
    linear-gradient(180deg, rgba(16, 30, 47, 0.96) 0%, rgba(11, 23, 38, 0.96) 100%) !important;
  box-shadow:
    inset 0 1px 0 rgba(148, 163, 184, 0.1),
    0 8px 18px rgba(2, 6, 23, 0.3) !important;
}

:global(html[data-theme='dark']) .detail-page .module-adr-head,
:global(html.dark) .detail-page .module-adr-head,
:global(body.dark) .detail-page .module-adr-head,
:global([data-theme='dark']) .detail-page .module-adr-head,
:global(.theme-dark) .detail-page .module-adr-head,
:global(.dark-theme) .detail-page .module-adr-head {
  color: #dbe9ff !important;
}

/* 深色主题下视图切换器单独适配 */
:global(html[data-theme='dark']) .detail-page .page-head .view-switch {
  background: rgba(11, 24, 40, 0.92) !important;
  border-color: rgba(71, 95, 124, 0.58) !important;
  border-radius: 10px !important;
  padding: 3px !important;
  box-shadow:
    inset 0 1px 0 rgba(148, 163, 184, 0.1),
    0 6px 14px rgba(2, 6, 23, 0.3) !important;
}

:global(html[data-theme='dark']) .detail-page .page-head .view-switch .view-switch-btn {
  color: #95a9c2 !important;
  border-radius: 7px !important;
  height: 30px !important;
  padding: 0 13px !important;
}

:global(html[data-theme='dark']) .detail-page .page-head .view-switch .view-switch-btn:hover {
  color: #e4edf8 !important;
  background: rgba(73, 101, 135, 0.32) !important;
}

:global(html[data-theme='dark']) .detail-page .page-head .view-switch .view-switch-btn.is-active {
  color: #9ddcff !important;
  background: linear-gradient(180deg, rgba(20, 42, 66, 0.98) 0%, rgba(12, 29, 47, 0.98) 100%) !important;
  box-shadow:
    inset 0 0 0 1px rgba(96, 186, 255, 0.52),
    0 2px 8px rgba(2, 6, 23, 0.35) !important;
}

/* 兼容 class 深色主题标记（部分页面不是 data-theme） */
:global(html.dark) .detail-page .page-head .view-switch,
:global(body.dark) .detail-page .page-head .view-switch {
  background: rgba(11, 24, 40, 0.92) !important;
  border-color: rgba(71, 95, 124, 0.58) !important;
  border-radius: 10px !important;
  padding: 3px !important;
  box-shadow:
    inset 0 1px 0 rgba(148, 163, 184, 0.1),
    0 6px 14px rgba(2, 6, 23, 0.3) !important;
}

:global(html.dark) .detail-page .page-head .view-switch .view-switch-btn,
:global(body.dark) .detail-page .page-head .view-switch .view-switch-btn {
  color: #95a9c2 !important;
  border-radius: 7px !important;
  height: 30px !important;
  padding: 0 13px !important;
}

:global(html.dark) .detail-page .page-head .view-switch .view-switch-btn:hover,
:global(body.dark) .detail-page .page-head .view-switch .view-switch-btn:hover {
  color: #e4edf8 !important;
  background: rgba(73, 101, 135, 0.32) !important;
}

:global(html.dark) .detail-page .page-head .view-switch .view-switch-btn.is-active,
:global(body.dark) .detail-page .page-head .view-switch .view-switch-btn.is-active {
  color: #9ddcff !important;
  background: linear-gradient(180deg, rgba(20, 42, 66, 0.98) 0%, rgba(12, 29, 47, 0.98) 100%) !important;
  box-shadow:
    inset 0 0 0 1px rgba(96, 186, 255, 0.52),
    0 2px 8px rgba(2, 6, 23, 0.35) !important;
}

:global(html.dark) .detail-page .page-head .left .el-button.is-plain,
:global(body.dark) .detail-page .page-head .left .el-button.is-plain,
:global([data-theme='dark']) .detail-page .page-head .left .el-button.is-plain,
:global(.theme-dark) .detail-page .page-head .left .el-button.is-plain,
:global(.dark-theme) .detail-page .page-head .left .el-button.is-plain {
  background: linear-gradient(180deg, #16314d 0%, #12263d 100%) !important;
  border-color: #4f7fab !important;
  color: #eaf4ff !important;
  box-shadow:
    inset 0 0 0 1px rgba(143, 212, 255, 0.2),
    0 2px 8px rgba(2, 6, 23, 0.28) !important;
}

:global(html.dark) .detail-page .page-head .left .el-button.is-plain:hover,
:global(body.dark) .detail-page .page-head .left .el-button.is-plain:hover,
:global([data-theme='dark']) .detail-page .page-head .left .el-button.is-plain:hover,
:global(.theme-dark) .detail-page .page-head .left .el-button.is-plain:hover,
:global(.dark-theme) .detail-page .page-head .left .el-button.is-plain:hover {
  background: linear-gradient(180deg, #1b3a5c 0%, #15304d 100%) !important;
  border-color: #6a9cc8 !important;
  color: #ffffff !important;
}

/* 兜底：任意祖先标记 data-theme='dark' / theme-dark / dark-theme 时都启用深色按钮 */
:global([data-theme='dark']) .detail-page .page-head .view-switch,
:global(.theme-dark) .detail-page .page-head .view-switch,
:global(.dark-theme) .detail-page .page-head .view-switch {
  background: rgba(11, 24, 40, 0.92) !important;
  border-color: rgba(71, 95, 124, 0.58) !important;
}

:global([data-theme='dark']) .detail-page .page-head .view-switch .view-switch-btn,
:global(.theme-dark) .detail-page .page-head .view-switch .view-switch-btn,
:global(.dark-theme) .detail-page .page-head .view-switch .view-switch-btn {
  color: #95a9c2 !important;
}

:global([data-theme='dark']) .detail-page .page-head .view-switch .view-switch-btn.is-active,
:global(.theme-dark) .detail-page .page-head .view-switch .view-switch-btn.is-active,
:global(.dark-theme) .detail-page .page-head .view-switch .view-switch-btn.is-active {
  color: #9ddcff !important;
  background: linear-gradient(180deg, rgba(20, 42, 66, 0.98) 0%, rgba(12, 29, 47, 0.98) 100%) !important;
}

/* 最终兜底：只要进入深色主题，切换按钮统一深色化 */
:global(html[data-theme='dark']) .view-switch,
:global([data-theme='dark']) .view-switch,
:global(html.dark) .view-switch,
:global(body.dark) .view-switch,
:global(.theme-dark) .view-switch,
:global(.dark-theme) .view-switch {
  background: rgba(11, 24, 40, 0.92) !important;
  border-color: rgba(71, 95, 124, 0.58) !important;
}

:global(html[data-theme='dark']) .view-switch .view-switch-btn,
:global([data-theme='dark']) .view-switch .view-switch-btn,
:global(html.dark) .view-switch .view-switch-btn,
:global(body.dark) .view-switch .view-switch-btn,
:global(.theme-dark) .view-switch .view-switch-btn,
:global(.dark-theme) .view-switch .view-switch-btn {
  color: #95a9c2 !important;
}

:global(html[data-theme='dark']) .view-switch .view-switch-btn:hover,
:global([data-theme='dark']) .view-switch .view-switch-btn:hover,
:global(html.dark) .view-switch .view-switch-btn:hover,
:global(body.dark) .view-switch .view-switch-btn:hover,
:global(.theme-dark) .view-switch .view-switch-btn:hover,
:global(.dark-theme) .view-switch .view-switch-btn:hover {
  color: #e4edf8 !important;
  background: rgba(73, 101, 135, 0.32) !important;
}

:global(html[data-theme='dark']) .view-switch .view-switch-btn.is-active,
:global([data-theme='dark']) .view-switch .view-switch-btn.is-active,
:global(html.dark) .view-switch .view-switch-btn.is-active,
:global(body.dark) .view-switch .view-switch-btn.is-active,
:global(.theme-dark) .view-switch .view-switch-btn.is-active,
:global(.dark-theme) .view-switch .view-switch-btn.is-active {
  color: #9ddcff !important;
  background: linear-gradient(180deg, rgba(20, 42, 66, 0.98) 0%, rgba(12, 29, 47, 0.98) 100%) !important;
  box-shadow:
    inset 0 0 0 1px rgba(96, 186, 255, 0.52),
    0 2px 8px rgba(2, 6, 23, 0.35) !important;
}

/* 系统深色模式兜底（无主题标记时） */
@media (prefers-color-scheme: dark) {
  .view-switch {
    background: rgba(11, 24, 40, 0.92) !important;
    border-color: rgba(71, 95, 124, 0.58) !important;
  }

  .view-switch .view-switch-btn {
    color: #95a9c2 !important;
  }

  .view-switch .view-switch-btn:hover {
    color: #e4edf8 !important;
    background: rgba(73, 101, 135, 0.32) !important;
  }

  .view-switch .view-switch-btn.is-active {
    color: #9ddcff !important;
    background: linear-gradient(180deg, rgba(20, 42, 66, 0.98) 0%, rgba(12, 29, 47, 0.98) 100%) !important;
    box-shadow:
      inset 0 0 0 1px rgba(96, 186, 255, 0.52),
      0 2px 8px rgba(2, 6, 23, 0.35) !important;
  }
}

:global(html[data-theme='dark']) .detail-page .base-card.base-card--summary .summary-strip {
  background:
    radial-gradient(120% 180% at 0% 0%, rgba(56, 189, 248, 0.18) 0%, rgba(56, 189, 248, 0) 55%),
    linear-gradient(108deg, #0f2238 0%, #13253d 42%, #111827 100%) !important;
  border: 1px solid rgba(71, 85, 105, 0.65);
  box-shadow:
    inset 0 1px 0 rgba(148, 163, 184, 0.16),
    0 8px 20px rgba(2, 6, 23, 0.32);
}

:global(html[data-theme='dark']) .detail-page .base-card.base-card--summary .summary-label {
  color: #9fb2ca !important;
  opacity: 1 !important;
}

:global(html[data-theme='dark']) .detail-page .base-card.base-card--summary .summary-value {
  color: #eef6ff !important;
  font-weight: 700 !important;
  opacity: 1 !important;
}

:global(html[data-theme='dark']) .detail-page .module-card {
  background: #ffffff !important;
  border-color: #f0f0f0 !important;
}

:global(html[data-theme='dark']) .detail-page .module-card :deep(.el-card__header) {
  background: #111827 !important;
  border-bottom: 1px solid #334155 !important;
}

:global(html[data-theme='dark']) .detail-page .module-card .module-head {
  color: #000 !important;
  opacity: 1 !important;
}

:global(html[data-theme='dark']) .detail-page .module-card .module-head > span:first-child {
  color: #e2e8f0 !important;
  font-weight: 700 !important;
}

:global(html[data-theme='dark']) .detail-page .module-card .module-head-tag {
  color: #389e0d !important;
  background: #f6ffed !important;
  border-color: #b7eb8f !important;
}

:global(html[data-theme='dark']) .detail-page .module-tile,
:global(html[data-theme='dark']) .detail-page .module-tile :deep(.el-card),
:global(html[data-theme='dark']) .detail-page .module-tile :deep(.el-card__header),
:global(html[data-theme='dark']) .detail-page .module-tile :deep(.el-card__body),
:global(html[data-theme='dark']) .detail-page .module-tile .tile-field {
  border-color: #334155 !important;
}

:global(html[data-theme='dark']) .detail-page .module-tile,
:global(html[data-theme='dark']) .detail-page .module-tile :deep(.el-card) {
  background: linear-gradient(180deg, #0f1b2e 0%, #0b1526 100%) !important;
  box-shadow:
    0 8px 20px rgba(2, 6, 23, 0.42),
    inset 0 1px 0 rgba(148, 163, 184, 0.12) !important;
}

:global(html[data-theme='dark']) .detail-page .module-tile :deep(.el-card__header) {
  background: linear-gradient(180deg, #15263f 0%, #0f1b2e 72%) !important;
  border-bottom: 1px solid rgba(100, 116, 139, 0.35) !important;
}

:global(html[data-theme='dark']) .detail-page .module-tile :deep(.el-card__body) {
  background: transparent !important;
}

:global(html[data-theme='dark']) .detail-page .module-tile .tile-field {
  background: rgba(15, 23, 42, 0.75) !important;
  border-color: rgba(100, 116, 139, 0.4) !important;
}

:global(html[data-theme='dark']) .detail-page .module-tile .tile-k {
  color: #9fb2ca !important;
  opacity: 1 !important;
}

:global(html[data-theme='dark']) .detail-page .module-tile .tile-v,
:global(html[data-theme='dark']) .detail-page .module-tile .tile-name {
  color: #e6eef9 !important;
  opacity: 1 !important;
}

:global(html[data-theme='dark']) .detail-page .module-card .module-head,
:global(html[data-theme='dark']) .detail-page .summary-strip .summary-label,
:global(html[data-theme='dark']) .detail-page .summary-strip .summary-value,
:global(html[data-theme='dark']) .detail-page .summary-strip .summary-status,
:global(html[data-theme='dark']) .detail-page .module-tile .tile-k,
:global(html[data-theme='dark']) .detail-page .module-tile .tile-v {
  text-shadow: none !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge,
:global(html.dark) .detail-page .summary-status-badge,
:global(body.dark) .detail-page .summary-status-badge,
:global([data-theme='dark']) .detail-page .summary-status-badge,
:global(.theme-dark) .detail-page .summary-status-badge,
:global(.dark-theme) .detail-page .summary-status-badge {
  background:
    linear-gradient(180deg, rgba(40, 134, 85, 0.52) 0%, rgba(24, 108, 64, 0.36) 100%),
    rgba(18, 37, 58, 0.84) !important;
  border-color: rgba(134, 239, 172, 0.58) !important;
  box-shadow:
    inset 0 1px 0 rgba(236, 253, 245, 0.2),
    0 10px 20px rgba(2, 6, 23, 0.34),
    0 18px 34px rgba(2, 6, 23, 0.46),
    0 0 0 1px rgba(134, 239, 172, 0.2) !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge .summary-status-text,
:global(html.dark) .detail-page .summary-status-badge .summary-status-text,
:global(body.dark) .detail-page .summary-status-badge .summary-status-text,
:global([data-theme='dark']) .detail-page .summary-status-badge .summary-status-text,
:global(.theme-dark) .detail-page .summary-status-badge .summary-status-text,
:global(.dark-theme) .detail-page .summary-status-badge .summary-status-text {
  color: #dcffd8 !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge .summary-status-label,
:global(html.dark) .detail-page .summary-status-badge .summary-status-label,
:global(body.dark) .detail-page .summary-status-badge .summary-status-label,
:global([data-theme='dark']) .detail-page .summary-status-badge .summary-status-label,
:global(.theme-dark) .detail-page .summary-status-badge .summary-status-label,
:global(.dark-theme) .detail-page .summary-status-badge .summary-status-label {
  color: rgba(220, 255, 216, 0.84) !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge .summary-status-dot,
:global(html.dark) .detail-page .summary-status-badge .summary-status-dot,
:global(body.dark) .detail-page .summary-status-badge .summary-status-dot,
:global([data-theme='dark']) .detail-page .summary-status-badge .summary-status-dot,
:global(.theme-dark) .detail-page .summary-status-badge .summary-status-dot,
:global(.dark-theme) .detail-page .summary-status-badge .summary-status-dot {
  background: #86efac !important;
  box-shadow:
    0 0 0 0 rgba(134, 239, 172, 0.5),
    0 0 13px rgba(134, 239, 172, 0.52) !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge .summary-status-packet,
:global(html.dark) .detail-page .summary-status-badge .summary-status-packet,
:global(body.dark) .detail-page .summary-status-badge .summary-status-packet,
:global([data-theme='dark']) .detail-page .summary-status-badge .summary-status-packet,
:global(.theme-dark) .detail-page .summary-status-badge .summary-status-packet,
:global(.dark-theme) .detail-page .summary-status-badge .summary-status-packet {
  background: radial-gradient(circle, rgba(240, 253, 244, 0.92) 0%, rgba(187, 247, 208, 0.82) 70%, rgba(134, 239, 172, 0) 100%) !important;
  box-shadow: 0 0 10px rgba(134, 239, 172, 0.48) !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge .summary-status-platform,
:global(html.dark) .detail-page .summary-status-badge .summary-status-platform,
:global(body.dark) .detail-page .summary-status-badge .summary-status-platform,
:global([data-theme='dark']) .detail-page .summary-status-badge .summary-status-platform,
:global(.theme-dark) .detail-page .summary-status-badge .summary-status-platform,
:global(.dark-theme) .detail-page .summary-status-badge .summary-status-platform {
  border-color: rgba(134, 239, 172, 0.68) !important;
  background: linear-gradient(180deg, rgba(74, 222, 128, 0.34) 0%, rgba(34, 197, 94, 0.2) 100%) !important;
  box-shadow:
    inset 0 0 0 1px rgba(236, 253, 245, 0.24),
    0 0 9px rgba(134, 239, 172, 0.26) !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge .summary-status-track,
:global(html.dark) .detail-page .summary-status-badge .summary-status-track,
:global(body.dark) .detail-page .summary-status-badge .summary-status-track,
:global([data-theme='dark']) .detail-page .summary-status-badge .summary-status-track,
:global(.theme-dark) .detail-page .summary-status-badge .summary-status-track,
:global(.dark-theme) .detail-page .summary-status-badge .summary-status-track {
  background: linear-gradient(90deg, rgba(134, 239, 172, 0.22), rgba(134, 239, 172, 0.58), rgba(134, 239, 172, 0.22)) !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge.is-offline,
:global(html.dark) .detail-page .summary-status-badge.is-offline,
:global(body.dark) .detail-page .summary-status-badge.is-offline,
:global([data-theme='dark']) .detail-page .summary-status-badge.is-offline,
:global(.theme-dark) .detail-page .summary-status-badge.is-offline,
:global(.dark-theme) .detail-page .summary-status-badge.is-offline {
  background:
    linear-gradient(180deg, rgba(255, 77, 79, 0.28) 0%, rgba(255, 77, 79, 0.16) 100%),
    rgba(15, 29, 48, 0.78) !important;
  border-color: rgba(255, 120, 117, 0.44) !important;
  box-shadow:
    inset 0 1px 0 rgba(254, 226, 226, 0.14),
    0 8px 16px rgba(2, 6, 23, 0.38),
    0 16px 30px rgba(2, 6, 23, 0.48),
    0 0 0 1px rgba(255, 120, 117, 0.16) !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge.is-offline .summary-status-text,
:global(html.dark) .detail-page .summary-status-badge.is-offline .summary-status-text,
:global(body.dark) .detail-page .summary-status-badge.is-offline .summary-status-text,
:global([data-theme='dark']) .detail-page .summary-status-badge.is-offline .summary-status-text,
:global(.theme-dark) .detail-page .summary-status-badge.is-offline .summary-status-text,
:global(.dark-theme) .detail-page .summary-status-badge.is-offline .summary-status-text {
  color: #ffb4b8 !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge.is-offline .summary-status-label,
:global(html.dark) .detail-page .summary-status-badge.is-offline .summary-status-label,
:global(body.dark) .detail-page .summary-status-badge.is-offline .summary-status-label,
:global([data-theme='dark']) .detail-page .summary-status-badge.is-offline .summary-status-label,
:global(.theme-dark) .detail-page .summary-status-badge.is-offline .summary-status-label,
:global(.dark-theme) .detail-page .summary-status-badge.is-offline .summary-status-label {
  color: rgba(255, 180, 184, 0.8) !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge.is-offline .summary-status-platform,
:global(html.dark) .detail-page .summary-status-badge.is-offline .summary-status-platform,
:global(body.dark) .detail-page .summary-status-badge.is-offline .summary-status-platform,
:global([data-theme='dark']) .detail-page .summary-status-badge.is-offline .summary-status-platform,
:global(.theme-dark) .detail-page .summary-status-badge.is-offline .summary-status-platform,
:global(.dark-theme) .detail-page .summary-status-badge.is-offline .summary-status-platform {
  border-color: rgba(255, 120, 117, 0.54) !important;
  background: linear-gradient(180deg, rgba(252, 165, 165, 0.28) 0%, rgba(239, 68, 68, 0.16) 100%) !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge.is-offline .summary-status-dot,
:global(html.dark) .detail-page .summary-status-badge.is-offline .summary-status-dot,
:global(body.dark) .detail-page .summary-status-badge.is-offline .summary-status-dot,
:global([data-theme='dark']) .detail-page .summary-status-badge.is-offline .summary-status-dot,
:global(.theme-dark) .detail-page .summary-status-badge.is-offline .summary-status-dot,
:global(.dark-theme) .detail-page .summary-status-badge.is-offline .summary-status-dot {
  background: #ff7875 !important;
  box-shadow:
    0 0 0 0 rgba(255, 120, 117, 0.5),
    0 0 12px rgba(255, 120, 117, 0.46) !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge.is-offline .summary-status-track,
:global(html.dark) .detail-page .summary-status-badge.is-offline .summary-status-track,
:global(body.dark) .detail-page .summary-status-badge.is-offline .summary-status-track,
:global([data-theme='dark']) .detail-page .summary-status-badge.is-offline .summary-status-track,
:global(.theme-dark) .detail-page .summary-status-badge.is-offline .summary-status-track,
:global(.dark-theme) .detail-page .summary-status-badge.is-offline .summary-status-track {
  background: linear-gradient(90deg, rgba(255, 120, 117, 0.16), rgba(255, 120, 117, 0.46), rgba(255, 120, 117, 0.16)) !important;
}

:global(html[data-theme='dark']) .detail-page .summary-status-badge.is-offline .summary-status-packet,
:global(html.dark) .detail-page .summary-status-badge.is-offline .summary-status-packet,
:global(body.dark) .detail-page .summary-status-badge.is-offline .summary-status-packet,
:global([data-theme='dark']) .detail-page .summary-status-badge.is-offline .summary-status-packet,
:global(.theme-dark) .detail-page .summary-status-badge.is-offline .summary-status-packet,
:global(.dark-theme) .detail-page .summary-status-badge.is-offline .summary-status-packet {
  background: radial-gradient(circle, rgba(254, 242, 242, 0.9) 0%, rgba(252, 165, 165, 0.78) 70%, rgba(255, 120, 117, 0) 100%) !important;
  box-shadow: 0 0 10px rgba(255, 120, 117, 0.45) !important;
}

/* 42_SOC 数据点表：表格铺满容器；名称列 min-width 吸收剩余宽度 */
.soc42-point-table {
  width: 100%;
}

.soc42-point-table :deep(.soc42-id-input),
.soc42-point-table :deep(.soc42-id-input.el-input),
.soc42-point-table :deep(.soc42-name-input),
.soc42-point-table :deep(.soc42-name-input.el-input) {
  width: 100%;
}

.soc42-point-table :deep(.soc42-dtype-select) {
  width: 100%;
  max-width: 218px;
}
</style>
