<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
// 采集云使用原生抽屉，保留源项目的挂载位置与插槽行为。
import { ElDrawer, ElMessage, ElMessageBox } from 'element-plus';
import { useAuthStore } from '../stores/auth';
import { createStreamLabels, fetchStreamLabelsIntact, fetchStreamPage } from '../api/stream';
import { findEquipmentPage } from '../api/equipment';
import { fetchGatewayPage } from '../api/gateway';
import { fetchGatewayModules } from '../api/gateway-module';
import { syncEquipmentDataDetailsFromStreamLabels } from '../utils/syncDataDetails';
import { formatStreamReportType, normalizeStreamDataType } from '../utils/streamDataType';
const route = useRoute();
const authStore = useAuthStore();
const isDarkTheme = ref(false);
let themeObserver = null;
let themeMediaQuery = null;
function detectDarkTheme() {
    if (typeof window === 'undefined')
        return false;
    const html = document.documentElement;
    const body = document.body;
    const byAttr = html.getAttribute('data-theme') === 'dark' || body?.getAttribute?.('data-theme') === 'dark';
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
const streamCode = computed(() => String(route.params.streamCode || ''));
const streamName = computed(() => String(route.query.name || streamCode.value || ''));
const apiKey = computed(() => String(route.query.apiKey || ''));
const streamId = computed(() => String(route.query.streamId ?? route.query.id ?? ''));
const resolvedStreamId = ref(0);
const resolvedEquipmentId = ref(0);
const loading = ref(false);
const rows = ref([]);
const selectedIds = ref([]);
const selectedRows = ref([]);
const page = ref(1);
const pageSize = ref(30);
const keyword = ref('');
const total = computed(() => rows.value.length);
const filteredRows = computed(() => {
    const kw = keyword.value.trim().toLowerCase();
    if (!kw)
        return rows.value;
    return rows.value.filter(row => [row.tagName, row.gatewayId, row.linkedGateway].some(x => String(x || '').toLowerCase().includes(kw)));
});
const pagedRows = computed(() => {
    const start = (page.value - 1) * pageSize.value;
    return filteredRows.value.slice(start, start + pageSize.value);
});
const drawerVisible = ref(false);
const form = ref({
    id: '',
    streamId: '',
    tagName: '',
    gatewayId: '',
    linkedGateway: '',
    linkPermission: '',
    dataId: '',
    dataType: '1',
    reportType: '',
    formula: '',
    unformula: '',
});
const pointSelectMode = ref('pick');
const selectedPoints = ref([]);
const gatewayOptions = ref([]);
const gatewayPointMap = ref({});
const gatewayOptionsLoading = ref(false);
const pointOptionsLoading = ref(false);
const pointOptions = ref([]);
const pointMatchKeyword = ref('');
const pointSearchResults = ref([]);
const gatewayPointDetailMap = ref({});
const ALL_POINTS_LABEL = '全部';
watch(() => selectedPoints.value, (list) => {
    if (!Array.isArray(list) || list.length <= 1)
        return;
    const hasAll = list.includes(ALL_POINTS_LABEL);
    if (!hasAll)
        return;
    const last = list[list.length - 1];
    selectedPoints.value = last === ALL_POINTS_LABEL
        ? [ALL_POINTS_LABEL]
        : list.filter(x => x !== ALL_POINTS_LABEL);
}, { deep: true });
function normalizeTagRow(item) {
    const dataType = normalizeDataType(item?.dataType, item);
    const formula = String(item?.formula ?? '').trim() || 'x';
    const unformula = String(item?.unformula ?? '').trim() || 'x';
    return {
        id: String(item?.labelId ?? item?.id ?? ''),
        streamId: String(item?.streamId ?? item?.streamID ?? ''),
        tagName: String(item?.name ?? ''),
        gatewayId: String(item?.gatewayId ?? ''),
        linkedGateway: String(item?.gatewayName ?? ''),
        linkPermission: String(item?.dataName ?? ''),
        dataId: String(item?.dataId ?? ''),
        dataType,
        reportType: formatStreamReportType(item?.dataType, item),
        formula,
        unformula,
    };
}
function pickValidStreamId(input) {
    const n = Number(input);
    return Number.isFinite(n) && n > 0 ? n : 0;
}
function normalizeDataType(input, source = {}) {
    return normalizeStreamDataType(input, source);
}
async function resolveStreamIdIfMissing() {
    if (!authStore.token)
        return;
    if (resolvedStreamId.value > 0)
        return;
    resolvedStreamId.value = pickValidStreamId(streamId.value);
    if (resolvedStreamId.value > 0)
        return;
    try {
        const { data } = await fetchStreamPage({
            token: authStore.token,
            page: 1,
            size: 100000,
            search: '',
        });
        const payload = data?.return ?? {};
        const list = payload?.list ?? payload?.data ?? payload?.rows ?? payload?.records ?? [];
        if (!Array.isArray(list))
            return;
        const match = list.find((item) => {
            const itemCode = String(item?.streamCode ?? '').trim();
            const itemName = String(item?.name ?? '').trim();
            const itemApiKey = String(item?.apiKey ?? '').trim();
            return (streamCode.value && itemCode === streamCode.value)
                || (streamName.value && itemName === streamName.value)
                || (apiKey.value && itemApiKey === apiKey.value);
        });
        resolvedStreamId.value = pickValidStreamId(match?.streamId ?? match?.id ?? match?.streamID ?? match?.stream_id);
    }
    catch {
        // 解析失败时先忽略，提交时会给出明确提示。
    }
}
async function resolveEquipmentIdIfMissing() {
    if (!authStore.token)
        return 0;
    if (resolvedEquipmentId.value > 0)
        return resolvedEquipmentId.value;
    const code = streamCode.value.trim();
    if (!code)
        return 0;
    try {
        const { data } = await findEquipmentPage({
            token: authStore.token,
            pageNum: 1,
            pageSize: 100000,
            filterFunc: 'BinC',
        });
        const payload = data?.return ?? {};
        const list = payload?.data ?? payload?.list ?? payload?.rows ?? [];
        if (!Array.isArray(list))
            return 0;
        const match = list.find((item) => String(item?.equipmentCode ?? '').trim() === code);
        const id = Number(match?.id ?? 0);
        if (Number.isFinite(id) && id > 0) {
            resolvedEquipmentId.value = id;
            return id;
        }
    }
    catch {
        // 同步业务数据点失败不影响标签本身创建。
    }
    return 0;
}
async function syncCurrentEquipmentDataDetails(labels) {
    const equipmentId = await resolveEquipmentIdIfMissing();
    if (!equipmentId)
        return { ok: false, added: 0, deleted: 0, msg: '未找到绑定设备，业务数据点未同步' };
    return syncEquipmentDataDetailsFromStreamLabels({
        token: authStore.token,
        equipmentId,
        streamName: streamName.value.trim(),
        labels,
    });
}
async function loadLabelRows() {
    if (!authStore.token)
        return;
    loading.value = true;
    try {
        const { data } = await fetchStreamLabelsIntact({
            token: authStore.token,
            streamName: streamName.value,
        });
        if (data?.code === 400) {
            ElMessage.error(data?.msg || '获取标签数据失败');
            return;
        }
        const payload = data?.return ?? {};
        const list = payload?.list ?? payload?.data ?? payload?.rows ?? [];
        rows.value = Array.isArray(list) ? list.map(normalizeTagRow) : [];
        if (resolvedStreamId.value <= 0) {
            resolvedStreamId.value = pickValidStreamId(payload?.streamId ??
                payload?.id ??
                (Array.isArray(list) ? (list[0]?.streamId ?? list[0]?.id) : 0));
        }
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '标签数据请求失败');
    }
    finally {
        loading.value = false;
    }
}
function openAdd() {
    form.value = {
        id: '',
        streamId: '',
        tagName: '',
        gatewayId: '',
        linkedGateway: '',
        linkPermission: '',
        dataId: '',
        dataType: '1',
        reportType: '',
        formula: '',
        unformula: '',
    };
    pointSelectMode.value = 'pick';
    selectedPoints.value = [];
    pointOptions.value = [];
    pointSearchResults.value = [];
    gatewayPointDetailMap.value = {};
    pointMatchKeyword.value = '';
    void loadGatewayOptions();
    drawerVisible.value = true;
}
async function loadGatewayOptions() {
    if (!authStore.token)
        return;
    gatewayOptionsLoading.value = true;
    try {
        const { data } = await fetchGatewayPage({
            token: authStore.token,
            pageNum: 1,
            pageSize: 100000,
            name: '',
        });
        if (data?.code === 400)
            return;
        const payload = data?.return ?? {};
        const records = Array.isArray(payload)
            ? payload
            : (payload?.data ?? payload?.records ?? payload?.list ?? payload?.rows ?? payload?.items ?? []);
        const list = Array.isArray(records) ? records : [];
        const nextMap = {};
        gatewayOptions.value = list
            .flatMap((item) => {
            const collectPointNames = (arr) => {
                const set = new Set();
                for (const p of arr) {
                    const n = String(p?.name ?? p?.dataName ?? '').trim();
                    if (n)
                        set.add(n);
                }
                return Array.from(set);
            };
            const rootPointCandidates = [
                ...(Array.isArray(item?.gatewayDatas) ? item.gatewayDatas : []),
                ...(Array.isArray(item?.data) ? item.data : []),
                ...(Array.isArray(item?.dataList) ? item.dataList : []),
                ...(Array.isArray(item?.list) ? item.list : []),
                ...(Array.isArray(item?.rows) ? item.rows : []),
            ];
            const rootPoints = collectPointNames(rootPointCandidates);
            const gatewayList = Array.isArray(item?.gateway) ? item.gateway : [];
            if (gatewayList.length) {
                return gatewayList.map((g) => {
                    const gatewayId = String(g?.gatewayId ??
                        g?.gatewayID ??
                        g?.gatewayid ??
                        g?.gateway_id ??
                        g?.id ??
                        item?.gatewayId ??
                        item?.gatewayID ??
                        '').trim();
                    const name = String(g?.name ??
                        g?.gatewayName ??
                        g?.gatewayname ??
                        g?.gateway_name ??
                        item?.name ??
                        item?.gatewayName ??
                        gatewayId).trim();
                    const childPoints = collectPointNames(Array.isArray(g?.data) ? g.data : []);
                    const points = childPoints.length ? childPoints : rootPoints;
                    if (gatewayId)
                        nextMap[gatewayId] = points;
                    return { name, gatewayId };
                });
            }
            const gatewayId = String(item?.gatewayId ??
                item?.gatewayID ??
                item?.gatewayid ??
                item?.gateway_id ??
                item?.id ??
                '').trim();
            const name = String(item?.name ??
                item?.gatewayName ??
                item?.gatewayname ??
                item?.gateway_name ??
                item?.gatewayId ??
                item?.gatewayID ??
                '').trim();
            if (gatewayId)
                nextMap[gatewayId] = rootPoints;
            return [{ name, gatewayId }];
        })
            .filter((x) => !!x.name && !!x.gatewayId);
        gatewayPointMap.value = nextMap;
        if (!gatewayOptions.value.length) {
            ElMessage.warning('未获取到网关数据，请检查接口返回');
        }
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '获取网关数据失败');
    }
    finally {
        gatewayOptionsLoading.value = false;
    }
}
function onGatewaySelectVisibleChange(visible) {
    if (visible && !gatewayOptions.value.length) {
        void loadGatewayOptions();
    }
}
watch(() => drawerVisible.value, (visible) => {
    if (visible && !gatewayOptions.value.length) {
        void loadGatewayOptions();
    }
});
async function loadPointOptionsByGateway(gatewayId) {
    const gid = String(gatewayId || '').trim();
    if (!gid || !authStore.token) {
        pointOptions.value = [];
        selectedPoints.value = [];
        return;
    }
    pointOptionsLoading.value = true;
    try {
        gatewayPointDetailMap.value[gid] = [];
        const namesBuffer = Array.isArray(gatewayPointMap.value[gid]) ? [...gatewayPointMap.value[gid]] : [];
        const detailBuffer = [];
        const { data } = await fetchGatewayModules({
            token: authStore.token,
            gatewayId: gid,
        });
        const root = data?.return ?? data;
        const rawRows = Array.isArray(root)
            ? root
            : (Array.isArray(root?.data)
                ? root.data
                : (root?.data ? [root.data] : (root?.list ?? root?.rows ?? [])));
        const rows = Array.isArray(rawRows) ? rawRows : [];
        const collectNames = (arr) => {
            for (const p of arr) {
                const n = String(p?.name ??
                    p?.dataName ??
                    p?.pointName ??
                    p?.label ??
                    '').trim();
                if (!n)
                    continue;
                const dataId = String(p?.dataId ?? p?.id ?? '').trim();
                const dataType = normalizeDataType(p?.dataType ?? p?.type ?? 1, p);
                namesBuffer.push(n);
                detailBuffer.push({ name: n, dataId, dataType });
            }
        };
        for (const row of rows) {
            const model = String(row?.model ?? '').trim().toUpperCase();
            if (model === 'TTDC') {
                const modules = Array.isArray(row?.gatewayModules) ? row.gatewayModules : [];
                for (const mod of modules) {
                    collectNames(Array.isArray(mod?.list) ? mod.list : []);
                    collectNames(Array.isArray(mod?.data) ? mod.data : []);
                    collectNames(Array.isArray(mod?.pointList) ? mod.pointList : []);
                }
                continue;
            }
            collectNames(Array.isArray(row?.data) ? row.data : []);
            collectNames(Array.isArray(row?.gatewayDatas) ? row.gatewayDatas : []);
            collectNames(Array.isArray(row?.points) ? row.points : []);
            collectNames(Array.isArray(row?.pointList) ? row.pointList : []);
            if (Array.isArray(row?.gateway)) {
                for (const g of row.gateway) {
                    collectNames(Array.isArray(g?.data) ? g.data : []);
                    collectNames(Array.isArray(g?.gatewayDatas) ? g.gatewayDatas : []);
                    collectNames(Array.isArray(g?.points) ? g.points : []);
                }
            }
        }
        let names = namesBuffer;
        gatewayPointDetailMap.value[gid] = detailBuffer;
        gatewayPointMap.value[gid] = names;
        if (!names.length) {
            const globalNames = [];
            Object.values(gatewayPointMap.value).forEach(arr => {
                arr.forEach(n => {
                    const s = String(n || '').trim();
                    if (s)
                        globalNames.push(s);
                });
            });
            if (globalNames.length)
                names = globalNames;
        }
        pointOptions.value = [ALL_POINTS_LABEL, ...names];
        if (!names.length) {
            ElMessage.warning('该网关未返回可选数据点');
        }
        selectedPoints.value = selectedPoints.value.filter(x => x === ALL_POINTS_LABEL || names.includes(x));
    }
    catch (error) {
        pointOptions.value = [ALL_POINTS_LABEL];
        selectedPoints.value = [];
        ElMessage.error(error?.response?.data?.msg || '获取数据点失败');
    }
    finally {
        pointOptionsLoading.value = false;
    }
}
function onGatewayChange(gatewayId) {
    form.value.gatewayId = String(gatewayId || '').trim();
    const hit = gatewayOptions.value.find(x => x.gatewayId === form.value.gatewayId);
    form.value.linkedGateway = hit?.name || '';
    selectedPoints.value = [];
    pointSearchResults.value = [];
    pointMatchKeyword.value = '';
    void loadPointOptionsByGateway(form.value.gatewayId);
}
function applyPointMatch() {
    const kw = pointMatchKeyword.value.trim().toLowerCase();
    if (!kw) {
        ElMessage.warning('请输入搜索关键字');
        return;
    }
    const hits = pointOptions.value.filter(x => x.toLowerCase().includes(kw));
    if (!hits.length) {
        pointSearchResults.value = [];
        ElMessage.warning('未搜索到数据点');
        return;
    }
    pointSearchResults.value = hits;
    ElMessage.success(`已搜索到 ${hits.length} 个数据点，请手动选择`);
}
async function submit() {
    if (!String(form.value.gatewayId || '').trim()) {
        ElMessage.warning('请选择网关');
        return;
    }
    if (!authStore.token) {
        ElMessage.warning('登录已失效，请重新登录');
        return;
    }
    await resolveStreamIdIfMissing();
    const sid = resolvedStreamId.value || pickValidStreamId(streamId.value);
    if (!Number.isFinite(sid) || sid <= 0) {
        ElMessage.warning('缺少数据池ID，无法添加标签');
        return;
    }
    if (!streamName.value.trim() || !apiKey.value.trim()) {
        ElMessage.warning('缺少数据池名称或 ApiKey，无法添加标签');
        return;
    }
    const allPointNames = pointOptions.value.filter(x => x !== ALL_POINTS_LABEL);
    const effectivePoints = selectedPoints.value.includes(ALL_POINTS_LABEL) ? allPointNames : selectedPoints.value;
    if (!effectivePoints.length) {
        ElMessage.warning('请选择数据点');
        return;
    }
    const pointDetails = gatewayPointDetailMap.value[String(form.value.gatewayId || '').trim()] ?? [];
    if (!pointDetails.length) {
        ElMessage.warning('未获取到数据点详情，无法提交');
        return;
    }
    const submitFormula = String(form.value.formula || '').trim() || 'x';
    const submitUnformula = String(form.value.unformula || '').trim() || 'x';
    const pickedDetails = selectedPoints.value.includes(ALL_POINTS_LABEL)
        ? pointDetails
        : effectivePoints.map(pointName => {
            const hit = pointDetails.find(x => x.name === pointName);
            return hit ?? { name: pointName, dataId: '', dataType: '1' };
        });
    const invalid = pickedDetails.filter(x => !String(x.dataId || '').trim());
    if (invalid.length) {
        ElMessage.warning('存在缺少 dataId 的数据点，请重新选择');
        return;
    }
    const labelName = String(form.value.tagName || '').trim();
    const gatewayName = String(form.value.linkedGateway || '').trim();
    const payload = pickedDetails.map(item => ({
        streamId: sid,
        name: labelName || item.name,
        gatewayName,
        gatewayId: String(form.value.gatewayId || '').trim(),
        dataName: item.name,
        dataId: item.dataId,
        formula: submitFormula,
        unformula: submitUnformula,
        dataType: normalizeDataType(item.dataType, item),
    }));
    try {
        const { data } = await createStreamLabels({
            token: authStore.token,
            streamName: streamName.value.trim(),
            apiKey: apiKey.value.trim(),
            body: payload,
        });
        if (Number(data?.code) !== 200) {
            ElMessage.error(data?.msg || '添加标签失败');
            return;
        }
        const sync = await syncCurrentEquipmentDataDetails(payload);
        if (!sync.ok) {
            ElMessage.warning(`添加标签成功，${sync.msg}`);
        }
        else {
            ElMessage.success(sync.added > 0 ? `添加标签成功，${sync.msg}` : '添加标签成功');
        }
        drawerVisible.value = false;
        await loadLabelRows();
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '添加标签失败');
    }
}
async function onDeleteRow(_row) {
    try {
        await ElMessageBox.confirm('确定要删除该标签吗？', '提示', {
            confirmButtonText: '删除',
            cancelButtonText: '取消',
            type: 'warning',
        });
    }
    catch {
        return;
    }
    if (!authStore.token) {
        ElMessage.warning('登录已失效，请重新登录');
        return;
    }
    if (!streamName.value.trim() || !apiKey.value.trim()) {
        ElMessage.warning('缺少数据池名称或 ApiKey，无法删除');
        return;
    }
    await resolveStreamIdIfMissing();
    const sid = resolvedStreamId.value || pickValidStreamId(streamId.value) || pickValidStreamId(_row.streamId);
    if (!sid) {
        ElMessage.warning('缺少数据池ID，无法删除');
        return;
    }
    await loadLabelRows();
    const latest = rows.value.find(item => item.id === _row.id);
    const rowForDelete = latest ?? _row;
    const body = [{
            streamId: sid,
            labelId: String(rowForDelete.id || '').trim(),
            name: String(rowForDelete.tagName || '').trim() || String(rowForDelete.linkPermission || '').trim(),
            gatewayName: String(rowForDelete.linkedGateway || '').trim(),
            gatewayId: String(rowForDelete.gatewayId || '').trim(),
            dataName: String(rowForDelete.linkPermission || '').trim(),
            dataId: String(rowForDelete.dataId || '').trim(),
            formula: String(rowForDelete.formula || '').trim() || 'x',
            unformula: String(rowForDelete.unformula || '').trim() || 'x',
            dataType: normalizeDataType(rowForDelete.dataType, rowForDelete),
        }];
    try {
        const { data } = await createStreamLabels({
            token: authStore.token,
            streamName: streamName.value.trim(),
            apiKey: apiKey.value.trim(),
            functionName: 'delete',
            body,
        });
        if (Number(data?.code) !== 200) {
            ElMessage.error(data?.msg || '删除标签失败');
            return;
        }
        await loadLabelRows();
        const sync = await syncCurrentEquipmentDataDetails();
        if (!sync.ok) {
            ElMessage.warning(`删除标签成功，${sync.msg}`);
        }
        else {
            ElMessage.success(`删除标签成功，${sync.msg}`);
        }
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '删除标签失败');
    }
}
async function onBatchDelete() {
    if (!selectedIds.value.length)
        return;
    try {
        await ElMessageBox.confirm(`确定要批量删除已勾选的 ${selectedIds.value.length} 个标签吗？`, '提示', {
            confirmButtonText: '删除',
            cancelButtonText: '取消',
            type: 'warning',
        });
    }
    catch {
        return;
    }
    if (!authStore.token) {
        ElMessage.warning('登录已失效，请重新登录');
        return;
    }
    if (!streamName.value.trim() || !apiKey.value.trim()) {
        ElMessage.warning('缺少数据池名称或 ApiKey，无法删除');
        return;
    }
    await resolveStreamIdIfMissing();
    const sid = resolvedStreamId.value || pickValidStreamId(streamId.value);
    if (!sid) {
        ElMessage.warning('缺少数据池ID，无法删除');
        return;
    }
    const selectedIdSnapshot = [...selectedIds.value];
    const selectedRowsSnapshot = [...selectedRows.value];
    await loadLabelRows();
    const latestMap = new Map(rows.value.map(item => [item.id, item]));
    const targets = selectedIdSnapshot
        .map(id => latestMap.get(id) || selectedRowsSnapshot.find(row => row.id === id))
        .filter((x) => !!x);
    if (!targets.length) {
        ElMessage.warning('未找到可删除的标签数据');
        return;
    }
    const body = targets.map(item => ({
        streamId: sid,
        labelId: String(item.id || '').trim(),
        name: String(item.tagName || '').trim() || String(item.linkPermission || '').trim(),
        gatewayName: String(item.linkedGateway || '').trim(),
        gatewayId: String(item.gatewayId || '').trim(),
        dataName: String(item.linkPermission || '').trim(),
        dataId: String(item.dataId || '').trim(),
        formula: String(item.formula || '').trim() || 'x',
        unformula: String(item.unformula || '').trim() || 'x',
        dataType: normalizeDataType(item.dataType, item),
    })).filter(item => !!item.labelId);
    if (!body.length) {
        ElMessage.warning('勾选标签缺少 labelId，无法批量删除');
        return;
    }
    try {
        const { data } = await createStreamLabels({
            token: authStore.token,
            streamName: streamName.value.trim(),
            apiKey: apiKey.value.trim(),
            functionName: 'delete',
            body,
        });
        if (Number(data?.code) !== 200) {
            const failed = [];
            for (const item of body) {
                try {
                    const { data: oneData } = await createStreamLabels({
                        token: authStore.token,
                        streamName: streamName.value.trim(),
                        apiKey: apiKey.value.trim(),
                        functionName: 'delete',
                        body: [item],
                    });
                    if (Number(oneData?.code) !== 200)
                        failed.push(item.labelId);
                }
                catch {
                    failed.push(item.labelId);
                }
            }
            if (failed.length) {
                ElMessage.error(`批量删除失败 ${failed.length} 条`);
                return;
            }
        }
        selectedIds.value = [];
        selectedRows.value = [];
        await loadLabelRows();
        const sync = await syncCurrentEquipmentDataDetails();
        if (!sync.ok) {
            ElMessage.warning(`批量删除成功，${sync.msg}`);
        }
        else {
            ElMessage.success(`批量删除成功，${sync.msg}`);
        }
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '批量删除失败');
    }
}
function onSelectionChange(list) {
    selectedRows.value = list;
    selectedIds.value = list.map(x => x.id).filter(Boolean);
}
function onSearch() {
    page.value = 1;
    void loadLabelRows();
}
function onReset() {
    keyword.value = '';
    page.value = 1;
}
async function copyApiKey() {
    const text = apiKey.value.trim();
    if (!text) {
        ElMessage.warning('当前没有可复制的 apikey');
        return;
    }
    try {
        if (navigator.clipboard && window.isSecureContext) {
            await navigator.clipboard.writeText(text);
        }
        else {
            const textarea = document.createElement('textarea');
            textarea.value = text;
            textarea.style.position = 'fixed';
            textarea.style.left = '-9999px';
            document.body.appendChild(textarea);
            textarea.focus();
            textarea.select();
            document.execCommand('copy');
            document.body.removeChild(textarea);
        }
        ElMessage.success('apikey 已复制');
    }
    catch {
        ElMessage.error('复制失败');
    }
}
onMounted(() => {
    syncThemeMode();
    themeObserver = new MutationObserver(() => syncThemeMode());
    themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class', 'data-theme'] });
    if (document.body)
        themeObserver.observe(document.body, { attributes: true, attributeFilter: ['class', 'data-theme'] });
    themeMediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    themeMediaQuery.addEventListener('change', syncThemeMode);
    void resolveStreamIdIfMissing();
    void loadLabelRows();
});
onUnmounted(() => {
    themeObserver?.disconnect();
    themeObserver = null;
    if (themeMediaQuery) {
        themeMediaQuery.removeEventListener('change', syncThemeMode);
        themeMediaQuery = null;
    }
});
</script>

<template>
  <main class="stream-detail" :class="{ 'is-dark': isDarkTheme }">
    <div class="page-head">
      <div class="left">
        <div class="title-wrap">
          <h2>标签列表</h2>
          <div class="stat-strip">
            <span class="stat-chip"><span class="k">标签总数</span><span class="v">{{ total }}</span></span>
            <span class="stat-chip"><span class="k">apikey</span><span class="v mono">{{ apiKey || '-' }}</span></span>
          </div>
        </div>
      </div>
      <div class="right">
        <el-button plain @click="copyApiKey">复制 apikey</el-button>
        <el-button type="primary" @click="openAdd">添加标签</el-button>
        <el-button type="danger" :disabled="!selectedIds.length" @click="onBatchDelete">批量删除</el-button>
      </div>
    </div>

    <el-card shadow="never" class="table-card" v-loading="loading">
      <div class="toolbar">
        <el-input
          v-model="keyword"
          class="keyword-input"
          clearable
          placeholder="搜索标签名/网关ID/关联网关"
          @keyup.enter="onSearch"
        />
        <el-button type="primary" @click="onSearch">查询</el-button>
        <el-button @click="onReset">重置</el-button>
      </div>

      <el-table
        class="tag-table"
        :data="pagedRows"
        stripe
        fit
        style="width: 100%"
        @selection-change="onSelectionChange"
      >
        <el-table-column type="selection" width="46" align="center" header-align="center" />
        <el-table-column label="序号" width="90" align="center" header-align="center">
          <template #default="{ $index }">{{ (page - 1) * pageSize + $index + 1 }}</template>
        </el-table-column>
        <el-table-column prop="tagName" label="标签名" min-width="220" align="center" header-align="center" />
        <el-table-column prop="gatewayId" label="网关ID" min-width="220" align="center" header-align="center" />
        <el-table-column prop="linkedGateway" label="关联网关" min-width="160" align="center" header-align="center" />
        <el-table-column prop="linkPermission" label="关联数据" min-width="160" align="center" header-align="center" />
        <el-table-column prop="reportType" label="上报类型" min-width="140" align="center" header-align="center" />
        <el-table-column prop="formula" label="处理公式" min-width="140" align="center" header-align="center" />
        <el-table-column prop="unformula" label="反处理公式" min-width="140" align="center" header-align="center" />
        <el-table-column label="操作" width="120" align="center" header-align="center">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="onDeleteRow(row)">删除</el-button>
          </template>
        </el-table-column>
        <template #empty>
          <el-empty description="暂无标签数据" />
        </template>
      </el-table>

      <div class="pager">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="filteredRows.length"
          layout="total, sizes, prev, pager, next, jumper"
          :page-sizes="[30, 50, 100]"
          background
        />
      </div>
    </el-card>

    <el-drawer
      v-model="drawerVisible"
      title="添加标签"
      direction="rtl"
      size="420px"
      :destroy-on-close="false"
      class="stream-tag-drawer"
    >
      <el-form label-width="112px" label-position="right" class="tag-form">
        <el-form-item label="标签名">
          <el-input v-model="form.tagName" placeholder="请输入标签名" />
        </el-form-item>
        <el-form-item label="选择网关" required>
          <el-select
            v-model="form.gatewayId"
            class="drawer-full"
            placeholder="请选择"
            filterable
            clearable
            :teleported="false"
            :loading="gatewayOptionsLoading"
            @visible-change="onGatewaySelectVisibleChange"
            @change="onGatewayChange"
          >
            <el-option
              v-for="item in gatewayOptions"
              :key="`gw-${item.gatewayId || item.name}`"
              :label="item.name"
              :value="item.gatewayId"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="数据点选择方式">
          <el-radio-group v-model="pointSelectMode">
            <el-radio label="pick">点选</el-radio>
            <el-radio label="match">匹配</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="选择数据点" required>
          <template v-if="pointSelectMode === 'pick'">
            <el-select
              v-model="selectedPoints"
              class="drawer-full"
              :placeholder="form.gatewayId ? '请选择数据点' : '请先选择网关'"
              multiple
              collapse-tags
              collapse-tags-tooltip
              filterable
              clearable
              :disabled="!form.gatewayId"
              :loading="pointOptionsLoading"
              :teleported="false"
            >
              <el-option v-for="(item, idx) in pointOptions" :key="`pt-${idx}-${item}`" :label="item" :value="item" />
            </el-select>
          </template>
          <template v-else>
            <div class="match-wrap">
              <el-input v-model="pointMatchKeyword" placeholder="输入关键字搜索数据点名" />
              <el-button type="primary" plain @click="applyPointMatch">搜索</el-button>
            </div>
            <el-select
              v-model="selectedPoints"
              class="drawer-full"
              placeholder="搜索结果"
              multiple
              collapse-tags
              collapse-tags-tooltip
              filterable
              clearable
              :disabled="!pointSearchResults.length"
              :teleported="false"
            >
              <el-option
                v-for="(item, idx) in pointSearchResults"
                :key="`sel-${idx}-${item}`"
                :label="item"
                :value="item"
              />
            </el-select>
          </template>
        </el-form-item>

        <el-form-item label="处理公式" required>
          <el-input v-model="form.formula" placeholder="选填，x 代表变量，例如 (x-0.5)*100" />
        </el-form-item>
        <el-form-item label="反处理公式" required>
          <el-input v-model="form.unformula" placeholder="选填，x 代表变量，例如 (x-0.5)*100" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="drawer-footer">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" @click="submit">确定</el-button>
        </div>
      </template>
    </el-drawer>
  </main>
</template>

<style scoped>
.stream-detail {
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

.stat-strip {
  font-size: 12px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.stat-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  border-radius: 999px;
  background: linear-gradient(180deg, #f7fbff 0%, #eef4ff 100%);
  border: 1px solid #cfdef7;
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.85),
    0 3px 10px rgba(37, 99, 235, 0.12);
}

.k {
  color: #5b6b84;
  font-weight: 500;
}

.k::after {
  content: ':';
  margin-left: 1px;
}

.v {
  color: #1e3a8a;
  font-weight: 700;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
}

.right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.table-card {
  border-radius: 10px;
  border: 1px solid #e9eef5;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.02), 0 6px 18px rgba(15, 23, 42, 0.05);
}

.tag-table :deep(.el-table__cell .cell) {
  text-align: center;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.keyword-input {
  width: 320px;
}

.match-wrap {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.match-wrap :deep(.el-input) {
  flex: 1;
}

.pager {
  margin-top: 12px;
  display: flex;
  justify-content: flex-start;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.tag-form :deep(.el-form-item__label) {
  white-space: nowrap;
}

/* dark */
.stream-detail.is-dark {
  background: #0f172a;
}

.stream-detail.is-dark .page-head {
  background: linear-gradient(180deg, #13253d 0%, #0f1d31 100%);
  border-color: #294362;
  box-shadow: inset 0 1px 0 rgba(148, 163, 184, 0.1), 0 10px 22px rgba(2, 6, 23, 0.34);
}

.stream-detail.is-dark h2 {
  color: #ffffff;
}

.stream-detail.is-dark .stat-strip,
.stream-detail.is-dark .k,
.stream-detail.is-dark .v {
  color: #dbe9ff;
}

.stream-detail.is-dark .stat-chip {
  background: linear-gradient(180deg, rgba(26, 47, 74, 0.96) 0%, rgba(17, 35, 58, 0.96) 100%);
  border-color: rgba(96, 165, 250, 0.48);
  box-shadow:
    inset 0 1px 0 rgba(191, 219, 254, 0.2),
    0 6px 14px rgba(2, 6, 23, 0.38);
}

.stream-detail.is-dark .stat-chip .k {
  color: #9fc2ef;
}

.stream-detail.is-dark .stat-chip .v {
  color: #e6f1ff;
}

.stream-detail.is-dark .table-card {
  background: #111827;
  border-color: #243244;
  box-shadow: inset 0 1px 0 rgba(148, 163, 184, 0.08), 0 10px 22px rgba(2, 6, 23, 0.34);
}

.stream-detail.is-dark .toolbar :deep(.el-input__wrapper) {
  background: rgba(15, 29, 48, 0.95) !important;
  box-shadow: 0 0 0 1px rgba(71, 95, 124, 0.85) inset !important;
}

.stream-detail.is-dark .toolbar :deep(.el-input__inner) {
  color: #dbe9ff !important;
}


.stream-detail.is-dark :deep(.el-table) {
  --el-table-bg-color: #0b1526;
  --el-table-tr-bg-color: #0b1526;
  --el-table-header-bg-color: #0f1b2e;
  --el-table-row-hover-bg-color: #12213a;
  --el-table-border-color: #334155;
  --el-table-text-color: #cbd5e1;
  --el-table-header-text-color: #e2e8f0;
}

.stream-detail.is-dark :deep(.el-table th.el-table__cell) {
  font-weight: 600;
}

.stream-detail.is-dark :deep(.el-table td.el-table__cell) {
  background: #0b1526 !important;
  color: #dbe9ff !important;
}

.stream-detail.is-dark :deep(.el-table tr.el-table__row--striped td.el-table__cell) {
  background: #0e1a2d !important;
}

.stream-detail.is-dark :deep(.el-table .el-table__row:hover > td.el-table__cell) {
  background: #12213a !important;
}

.stream-detail.is-dark :deep(.el-table .cell) {
  color: #dbe9ff !important;
}

.stream-detail.is-dark :deep(.el-table .el-input__wrapper),
.stream-detail.is-dark :deep(.el-table .el-select__wrapper) {
  background: rgba(15, 29, 48, 0.95) !important;
  box-shadow: 0 0 0 1px rgba(71, 95, 124, 0.85) inset !important;
}

.stream-detail.is-dark :deep(.el-table .el-input__inner),
.stream-detail.is-dark :deep(.el-table .el-select__selected-item) {
  color: #dbe9ff !important;
}

.stream-detail.is-dark :deep(.el-table .el-checkbox__inner) {
  background: rgba(15, 29, 48, 0.96) !important;
  border-color: rgba(96, 165, 250, 0.72) !important;
}

.stream-detail.is-dark :deep(.el-table .el-checkbox__inner:hover) {
  border-color: #7dc3ff !important;
}

.stream-detail.is-dark :deep(.el-table .el-checkbox__input.is-checked .el-checkbox__inner),
.stream-detail.is-dark :deep(.el-table .el-checkbox__input.is-indeterminate .el-checkbox__inner) {
  background: linear-gradient(180deg, #2b74ba 0%, #1f5f9f 100%) !important;
  border-color: #69b8ff !important;
}

.stream-detail.is-dark :deep(.el-table .el-checkbox__input.is-disabled .el-checkbox__inner) {
  background: rgba(18, 34, 54, 0.88) !important;
  border-color: rgba(96, 165, 250, 0.42) !important;
}

.stream-detail.is-dark :deep(.el-pagination .btn-prev),
.stream-detail.is-dark :deep(.el-pagination .btn-next),
.stream-detail.is-dark :deep(.el-pagination .el-pager li) {
  background: rgba(15, 29, 48, 0.95) !important;
  border: 1px solid rgba(71, 95, 124, 0.72) !important;
  color: #cfe1f8 !important;
}

.stream-detail.is-dark :deep(.el-pagination .el-pager li.is-active) {
  background: linear-gradient(180deg, #225b8e 0%, #1a4f80 100%) !important;
  border-color: #5fa9e6 !important;
  color: #ffffff !important;
}

.stream-detail.is-dark :deep(.el-pagination .el-select__wrapper),
.stream-detail.is-dark :deep(.el-pagination .el-input__wrapper) {
  background: rgba(15, 29, 48, 0.95) !important;
  box-shadow: 0 0 0 1px rgba(71, 95, 124, 0.85) inset !important;
}

.stream-detail.is-dark :deep(.el-pagination .el-select__selected-item),
.stream-detail.is-dark :deep(.el-pagination .el-input__inner),
.stream-detail.is-dark :deep(.el-pagination .el-select__placeholder) {
  color: #dbe9ff !important;
}

/* 深色主题：添加标签抽屉（Drawer 使用 teleported，需要全局命中） */
.stream-detail.is-dark :deep(.stream-tag-drawer.el-drawer) {
  background: linear-gradient(180deg, #13253d 0%, #0f1d31 100%) !important;
}

.stream-detail.is-dark :deep(.stream-tag-drawer .el-drawer__header) {
  color: #eaf3ff !important;
  border-bottom: 1px solid #2b4666 !important;
  margin-bottom: 0 !important;
  padding-bottom: 14px !important;
}

.stream-detail.is-dark :deep(.stream-tag-drawer .el-drawer__body) {
  background: transparent !important;
}

.stream-detail.is-dark :deep(.stream-tag-drawer .el-form-item__label) {
  color: #9fb2ca !important;
}

.stream-detail.is-dark :deep(.stream-tag-drawer .el-input__wrapper),
.stream-detail.is-dark :deep(.stream-tag-drawer .el-select__wrapper),
.stream-detail.is-dark :deep(.stream-tag-drawer .el-textarea__inner) {
  background: rgba(15, 29, 48, 0.95) !important;
  box-shadow: 0 0 0 1px rgba(71, 95, 124, 0.85) inset !important;
}

.stream-detail.is-dark :deep(.stream-tag-drawer .el-input__inner),
.stream-detail.is-dark :deep(.stream-tag-drawer .el-select__selected-item),
.stream-detail.is-dark :deep(.stream-tag-drawer .el-textarea__inner),
.stream-detail.is-dark :deep(.stream-tag-drawer .el-input__inner::placeholder),
.stream-detail.is-dark :deep(.stream-tag-drawer .el-textarea__inner::placeholder) {
  color: #dbe9ff !important;
}

.stream-detail.is-dark :deep(.stream-tag-drawer .drawer-footer) {
  border-top: 1px solid #2b4666;
  padding-top: 12px;
}
</style>

