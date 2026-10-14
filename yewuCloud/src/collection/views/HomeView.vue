<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
// 采集云使用原生抽屉，保留源项目的挂载位置与插槽行为。
import { ElDrawer, ElMessage, ElMessageBox } from 'element-plus';
import { useAuthStore } from '../stores/auth';
import { gatewayDetailPath, normalizeGatewayModelString, pickGatewayModelField, } from '../config/gateway-detail';
import { createDefaultLabelNameResolver } from '../config/default-stream-labels';
import { createTtdcCid42SocSubmitPoints } from '../config/ttdc-cid42-soc-template';
import { addGateway, deleteGateway, fetchGatewayPage, updateGateway } from '../api/gateway';
import { addGatewayModule, fetchGatewayModules, fetchGatewayOne } from '../api/gateway-module';
import { batchImportStreams, createStream, createStreamLabels, deleteStream, downloadStreamBatchImportTemplate, fetchStreamPage, updateStream, } from '../api/stream';
import { findEquipmentPage, updateEquipment } from '../api/equipment';
import { syncEquipmentDataDetailsFromStreamLabels } from '../utils/syncDataDetails';
import { normalizeStreamDataType } from '../utils/streamDataType';
const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const activeModule = ref('gateway');
const isDarkTheme = ref(false);
let themeObserver = null;
let themeMediaQuery = null;
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
const gatewayRows = ref([]);
const gatewayOptions = ref([]);
const streamRows = ref([]);
const gatewayPage = ref(1);
const gatewayPageSize = ref(10);
const gatewayTotal = ref(0);
const gatewayLoading = ref(false);
const gatewayDrawerVisible = ref(false);
const gatewayFormLoading = ref(false);
const gatewayBatchDeleting = ref(false);
const selectedGatewayIds = ref([]);
const gatewayFormMode = ref('add');
const gatewayCopySourceGatewayId = ref('');
const defaultGatewayForm = {
    id: 0,
    name: '',
    gatewayId: '',
    model: 'TTDC',
    card: '',
};
const gatewayForm = ref({
    ...defaultGatewayForm,
});
const streamPage = ref(1);
const streamPageSize = ref(10);
const streamTotal = ref(0);
const streamLoading = ref(false);
const streamBatchDeleting = ref(false);
const selectedStreamApiKeys = ref([]);
const streamDrawerVisible = ref(false);
const streamFormLoading = ref(false);
const streamFormMode = ref('add');
const streamEditApiKey = ref('');
const gatewayOptionsLoading = ref(false);
const equipmentOptionsLoading = ref(false);
const ttdcAddressOptionsLoading = ref(false);
const equipmentCodeOptions = ref([]);
const ttdcAddressOptions = ref([]);
const defaultStreamForm = {
    id: 0,
    name: '',
    admin: '',
    apiKey: '',
    streamId: 0,
    gatewayId: '',
    streamCode: '',
    ttdcAdr: '',
};
const streamForm = ref({
    ...defaultStreamForm,
});
const batchImportVisible = ref(false);
const batchImportLoading = ref(false);
const batchImportConfirmLoading = ref(false);
const batchImportFile = ref(null);
const batchImportValidationFailed = ref(false);
const batchImportPreview = ref({ total: 0, issues: [], results: [] });
const gatewayName = ref('');
const streamKeyword = ref('');
const filteredStreamRows = computed(() => streamRows.value);
const selectedStreamGateway = computed(() => {
    const gid = String(streamForm.value.gatewayId || '').trim();
    if (!gid)
        return null;
    return gatewayOptions.value.find(item => String(item.gatewayId || '').trim() === gid) ?? null;
});
const isSelectedStreamGatewayTtdc = computed(() => {
    return String(selectedStreamGateway.value?.model ?? '').trim().toUpperCase() === 'TTDC';
});
function toPositiveInt(value, fallback) {
    const n = Number(value);
    return Number.isFinite(n) && n > 0 ? Math.floor(n) : fallback;
}
function buildHomeStatePath() {
    const query = {
        module: activeModule.value,
        gatewayPage: String(gatewayPage.value),
        gatewayPageSize: String(gatewayPageSize.value),
        gatewayName: gatewayName.value.trim(),
        streamPage: String(streamPage.value),
        streamPageSize: String(streamPageSize.value),
        streamKeyword: streamKeyword.value.trim(),
    };
    return router.resolve({ path: '/collection', query }).fullPath;
}
function goStreamDetail(row) {
    if (!row?.streamCode) {
        ElMessage.warning('缺少电池ID');
        return;
    }
    const currentStreamId = Number(row.streamId || row.id || 0);
    if (!Number.isFinite(currentStreamId) || currentStreamId <= 0) {
        ElMessage.warning('缺少数据池ID');
        return;
    }
    void router.push({
        name: 'stream-detail',
        params: { streamCode: String(row.streamCode).trim() },
        query: {
            streamId: String(currentStreamId),
            apiKey: String(row.apiKey || '').trim(),
            name: String(row.name || '').trim(),
            from: buildHomeStatePath(),
        },
    });
}
function applyStateFromQuery() {
    const q = route.query;
    const moduleKey = String(q.module || '');
    if (moduleKey === 'gateway' || moduleKey === 'stream') {
        activeModule.value = moduleKey;
    }
    gatewayPage.value = toPositiveInt(q.gatewayPage, gatewayPage.value);
    gatewayPageSize.value = toPositiveInt(q.gatewayPageSize, gatewayPageSize.value);
    gatewayName.value = String(q.gatewayName || '');
    streamPage.value = toPositiveInt(q.streamPage, streamPage.value);
    streamPageSize.value = toPositiveInt(q.streamPageSize, streamPageSize.value);
    streamKeyword.value = String(q.streamKeyword || '');
}
function onGatewaySearch() {
    gatewayPage.value = 1;
    void loadGatewayRows();
}
function onGatewayReset() {
    gatewayName.value = '';
    gatewayPage.value = 1;
    void loadGatewayRows();
}
function openGatewayDrawer() {
    gatewayFormMode.value = 'add';
    gatewayForm.value = { ...defaultGatewayForm };
    gatewayDrawerVisible.value = true;
}
function openGatewayEditDrawer(row) {
    gatewayFormMode.value = 'edit';
    gatewayForm.value = {
        id: Number(row.id ?? 0),
        name: row.name ?? '',
        gatewayId: row.gatewayId ?? '',
        model: 'TTDC',
        card: row.card ?? '',
    };
    gatewayDrawerVisible.value = true;
}
function openGatewayCopyDrawer(row) {
    gatewayFormMode.value = 'copy';
    gatewayCopySourceGatewayId.value = String(row.gatewayId || '').trim();
    gatewayForm.value = {
        id: 0,
        name: `${row.name || ''}-副本`,
        gatewayId: '',
        model: 'TTDC',
        card: row.card ?? '',
    };
    gatewayDrawerVisible.value = true;
}
function closeGatewayDrawer() {
    gatewayDrawerVisible.value = false;
    gatewayCopySourceGatewayId.value = '';
}
async function submitGatewayForm() {
    if (!authStore.token) {
        ElMessage.error('未登录或登录已失效');
        return;
    }
    if (!gatewayForm.value.name.trim()) {
        ElMessage.warning('请输入网关名');
        return;
    }
    if (!gatewayForm.value.gatewayId.trim()) {
        ElMessage.warning('请输入序列号');
        return;
    }
    if (gatewayFormMode.value === 'edit' && !gatewayForm.value.id) {
        ElMessage.warning('缺少网关id，无法修改');
        return;
    }
    if (gatewayFormMode.value === 'copy' && !gatewayCopySourceGatewayId.value) {
        ElMessage.warning('缺少复制源网关，无法复制');
        return;
    }
    gatewayFormLoading.value = true;
    try {
        const payload = {
            token: authStore.token,
            id: gatewayForm.value.id,
            name: gatewayForm.value.name.trim(),
            gatewayId: gatewayForm.value.gatewayId.trim(),
            model: normalizeGatewayModelString(gatewayForm.value.model),
            card: gatewayForm.value.card?.trim() || '',
        };
        if (gatewayFormMode.value === 'copy') {
            const { data: srcModulesData } = await fetchGatewayModules({
                token: authStore.token,
                gatewayId: gatewayCopySourceGatewayId.value,
            });
            if (srcModulesData?.code !== 200) {
                ElMessage.error(srcModulesData?.msg || '复制失败：读取源网关配置失败');
                return;
            }
            const srcModules = srcModulesData?.return?.data?.gatewayModules;
            const modules = Array.isArray(srcModules) ? srcModules : [];
            const copyBodies = [];
            for (const m of modules) {
                const moduleCid2 = String(m?.ttdcConfig?.moduleCid2 ?? '');
                const moduleAdr = String(m?.ttdcConfig?.moduleAdr ?? '');
                if (moduleCid2.trim() === '42_SOC' && !moduleAdr.trim()) {
                    ElMessage.warning(`通道 ${String(m?.name ?? '') || '42_SOC'} 缺少 TTDC 地址，无法复制`);
                    return;
                }
                copyBodies.push({
                    name: String(m?.name ?? ''),
                    gatewayId: payload.gatewayId,
                    type: Number(m?.type ?? 1),
                    transport: String(m?.transport ?? ''),
                    ttdcConfig: {
                        moduleSoi: String(m?.ttdcConfig?.moduleSoi ?? ''),
                        moduleVer: String(m?.ttdcConfig?.moduleVer ?? ''),
                        moduleAdr: String(m?.ttdcConfig?.moduleAdr ?? ''),
                        moduleCid1: String(m?.ttdcConfig?.moduleCid1 ?? ''),
                        moduleCid2: String(m?.ttdcConfig?.moduleCid2 ?? ''),
                    },
                    list: moduleCid2.trim() === '42_SOC'
                        ? createTtdcCid42SocSubmitPoints(moduleAdr)
                        : Array.isArray(m?.list)
                            ? m.list.map((p) => ({
                                name: String(p?.name ?? ''),
                                transport: String(p?.transport ?? ''),
                                dataType: Number(p?.dataType ?? 1),
                                formula: String(p?.formula ?? 'x'),
                                unformula: String(p?.unformula ?? 'x'),
                            }))
                            : [],
                });
            }
            const { data: addData } = await addGateway(payload);
            if (addData?.code !== 200) {
                ElMessage.error(addData?.msg || '复制失败：创建目标网关失败');
                return;
            }
            for (const body of copyBodies) {
                const { data: addModuleData } = await addGatewayModule({
                    token: authStore.token,
                    body,
                });
                if (addModuleData?.code !== 200) {
                    ElMessage.error(addModuleData?.msg || `复制失败：通道 ${body.name} 创建失败`);
                    return;
                }
            }
            ElMessage.success(`复制成功，已创建网关并复制 ${modules.length} 个通道`);
            closeGatewayDrawer();
            void loadGatewayRows();
            return;
        }
        const { data } = gatewayFormMode.value === 'edit'
            ? await updateGateway(payload)
            : await addGateway(payload);
        if (data?.code === 200) {
            ElMessage.success(data?.msg || (gatewayFormMode.value === 'edit' ? '修改成功' : '添加成功'));
            closeGatewayDrawer();
            void loadGatewayRows();
            return;
        }
        ElMessage.error(data?.msg || (gatewayFormMode.value === 'edit' ? '修改失败' : '添加失败'));
    }
    finally {
        gatewayFormLoading.value = false;
    }
}
function onGatewaySelectionChange(rows) {
    selectedGatewayIds.value = rows.map(row => row.gatewayId).filter(Boolean);
}
function goGatewayDetail(row) {
    void goGatewayDetailAsync(row);
}
async function goGatewayDetailAsync(row) {
    const gid = String(row.gatewayId || '').trim();
    if (!gid) {
        ElMessage.warning('缺少网关序列号');
        return;
    }
    let modelNorm = normalizeGatewayModelString(pickGatewayModelField(row) || String(row.model ?? ''));
    let path = gatewayDetailPath(modelNorm, gid);
    if (!path && authStore.token) {
        try {
            const { data } = await fetchGatewayOne({ token: authStore.token, gatewayId: gid });
            if (data?.code === 200) {
                const one = data?.return?.data ?? {};
                modelNorm = normalizeGatewayModelString(pickGatewayModelField(one));
                path = gatewayDetailPath(modelNorm, gid);
            }
        }
        catch {
            // 忽略，下面统一提示
        }
    }
    if (!path) {
        ElMessage.warning('该机型暂无详情页');
        return;
    }
    const resolved = router.resolve({
        path,
        query: {
            from: buildHomeStatePath(),
        },
    });
    window.open(resolved.href, '_blank', 'noopener,noreferrer');
}
async function onDeleteGateway(gatewayId) {
    if (!authStore.token) {
        ElMessage.error('未登录或登录已失效');
        return;
    }
    if (!gatewayId) {
        ElMessage.warning('gatewayId 为空，无法删除');
        return;
    }
    try {
        await ElMessageBox.confirm('确定要删除该网关吗？', '提示', {
            confirmButtonText: '删除',
            cancelButtonText: '取消',
            type: 'warning',
        });
    }
    catch {
        return;
    }
    try {
        const { data } = await deleteGateway({ token: authStore.token, gatewayId });
        if (data?.code === 200) {
            ElMessage.success(data?.msg || '删除成功');
            void loadGatewayRows();
            return;
        }
        ElMessage.error(data?.msg || '删除失败');
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '删除请求失败');
    }
}
async function onBatchDeleteGateways() {
    if (!authStore.token) {
        ElMessage.error('未登录或登录已失效');
        return;
    }
    const ids = [...selectedGatewayIds.value];
    if (!ids.length) {
        ElMessage.warning('请先勾选要删除的网关');
        return;
    }
    try {
        await ElMessageBox.confirm(`确定要批量删除已勾选的 ${ids.length} 个网关吗？`, '提示', {
            confirmButtonText: '删除',
            cancelButtonText: '取消',
            type: 'warning',
        });
    }
    catch {
        return;
    }
    gatewayBatchDeleting.value = true;
    let ok = 0;
    const fail = [];
    try {
        for (const gatewayId of ids) {
            try {
                const { data } = await deleteGateway({ token: authStore.token, gatewayId });
                if (data?.code === 200) {
                    ok += 1;
                }
                else {
                    fail.push(gatewayId);
                }
            }
            catch {
                fail.push(gatewayId);
            }
            // 轮询/顺序删除：给后端一点间隔，避免突发压测
            await new Promise(resolve => setTimeout(resolve, 150));
        }
    }
    finally {
        gatewayBatchDeleting.value = false;
    }
    if (fail.length === 0) {
        ElMessage.success(`批量删除成功，共删除 ${ok} 条`);
    }
    else {
        ElMessage.warning(`批量删除完成：成功 ${ok} 条，失败 ${fail.length} 条`);
    }
    selectedGatewayIds.value = [];
    void loadGatewayRows();
}
function onStreamSearch() {
    streamPage.value = 1;
    void loadStreamRows();
}
function onStreamReset() {
    streamKeyword.value = '';
    streamPage.value = 1;
    void loadStreamRows();
}
function onStreamSelectionChange(rows) {
    selectedStreamApiKeys.value = rows
        .map(row => String(row.apiKey || '').trim())
        .filter(Boolean);
}
async function onDeleteStream(apiKey) {
    if (!authStore.token) {
        ElMessage.error('未登录或登录已失效');
        return;
    }
    if (!apiKey) {
        ElMessage.warning('apiKey 为空，无法删除');
        return;
    }
    try {
        await ElMessageBox.confirm('确定要删除该数据池吗？', '提示', {
            confirmButtonText: '删除',
            cancelButtonText: '取消',
            type: 'warning',
        });
    }
    catch {
        return;
    }
    try {
        const { data } = await deleteStream({ token: authStore.token, apiKey });
        if (data?.code === 200) {
            ElMessage.success(data?.msg || '删除成功');
            void loadStreamRows();
            return;
        }
        ElMessage.error(data?.msg || '删除失败');
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '删除请求失败');
    }
}
async function onBatchDeleteStreams() {
    if (!authStore.token) {
        ElMessage.error('未登录或登录已失效');
        return;
    }
    const keys = [...selectedStreamApiKeys.value];
    if (!keys.length) {
        ElMessage.warning('请先勾选要删除的数据池');
        return;
    }
    try {
        await ElMessageBox.confirm(`确定要批量删除已勾选的 ${keys.length} 个数据池吗？`, '提示', {
            confirmButtonText: '删除',
            cancelButtonText: '取消',
            type: 'warning',
        });
    }
    catch {
        return;
    }
    streamBatchDeleting.value = true;
    let ok = 0;
    const fail = [];
    try {
        for (const apiKey of keys) {
            try {
                const { data } = await deleteStream({ token: authStore.token, apiKey });
                if (data?.code === 200)
                    ok += 1;
                else
                    fail.push(apiKey);
            }
            catch {
                fail.push(apiKey);
            }
            await new Promise(resolve => setTimeout(resolve, 120));
        }
    }
    finally {
        streamBatchDeleting.value = false;
    }
    if (fail.length === 0) {
        ElMessage.success(`批量删除成功，共删除 ${ok} 条`);
    }
    else {
        ElMessage.warning(`批量删除完成：成功 ${ok} 条，失败 ${fail.length} 条`);
    }
    selectedStreamApiKeys.value = [];
    void loadStreamRows();
}
function openStreamDrawer() {
    streamFormMode.value = 'add';
    streamEditApiKey.value = '';
    streamForm.value = { ...defaultStreamForm };
    ttdcAddressOptions.value = [];
    streamDrawerVisible.value = true;
}
function openStreamEditDrawer(row) {
    streamFormMode.value = 'edit';
    streamEditApiKey.value = String(row.apiKey || '').trim();
    streamForm.value = {
        id: Number(row.id || 0),
        name: String(row.name || '').trim(),
        admin: String(row.admin || '').trim(),
        apiKey: String(row.apiKey || '').trim(),
        streamId: Number(row.streamId || row.id || 0),
        gatewayId: '',
        streamCode: String(row.streamCode || '').trim(),
        ttdcAdr: '',
    };
    ttdcAddressOptions.value = [];
    streamDrawerVisible.value = true;
}
function closeStreamDrawer() {
    streamDrawerVisible.value = false;
    streamFormMode.value = 'add';
    streamEditApiKey.value = '';
}
function openBatchImport() {
    batchImportFile.value = null;
    batchImportValidationFailed.value = false;
    batchImportPreview.value = { total: 0, issues: [], results: [] };
    batchImportVisible.value = true;
}
function downloadBlob(blob, filename) {
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = filename;
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    window.setTimeout(() => URL.revokeObjectURL(url), 0);
}
function validationErrorFilename(filename) {
    const base = String(filename || '数据池批量导入').replace(/\.xlsx$/i, '');
    return `${base}_校验失败.xlsx`;
}
async function downloadBatchImportTemplate() {
    if (!authStore.token)
        return;
    try {
        const response = await downloadStreamBatchImportTemplate(authStore.token);
        const url = URL.createObjectURL(response.data);
        const anchor = document.createElement('a');
        anchor.href = url;
        anchor.download = '数据池批量导入模板.xlsx';
        anchor.click();
        URL.revokeObjectURL(url);
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '下载模板失败');
    }
}
async function uploadBatchImport(options) {
    if (!authStore.token) {
        options.onError(Object.assign(new Error('登录已失效'), { status: 401, method: 'POST', url: '/tolink/stream/batchimport' }));
        return;
    }
    batchImportFile.value = options.file;
    batchImportLoading.value = true;
    batchImportValidationFailed.value = false;
    batchImportPreview.value = { total: 0, issues: [], results: [] };
    try {
        const response = await batchImportStreams(authStore.token, options.file, true);
        const contentType = String(response.headers?.['content-type'] ?? '').toLowerCase();
        const disposition = String(response.headers?.['content-disposition'] ?? '').toLowerCase();
        const isValidationWorkbook = contentType.includes('application/vnd.openxmlformats-officedocument.spreadsheetml.sheet') ||
            disposition.includes('.xlsx');
        if (isValidationWorkbook) {
            batchImportValidationFailed.value = true;
            downloadBlob(response.data, validationErrorFilename(options.file.name));
            ElMessage.warning('校验失败，整张表禁止导入');
            options.onSuccess(response.data);
            return;
        }
        const responseBody = response.data instanceof Blob
            ? JSON.parse(await response.data.text())
            : response.data;
        const payload = responseBody?.return ?? {};
        batchImportPreview.value = {
            total: Number(payload.total ?? 0),
            issues: Array.isArray(payload.issues) ? payload.issues : [],
            results: Array.isArray(payload.results) ? payload.results : [],
        };
        if (Number(responseBody?.code) !== 200) {
            batchImportValidationFailed.value = true;
            ElMessage.warning('校验失败，整张表禁止导入');
        }
        else {
            ElMessage.success('校验通过，请确认导入');
        }
        options.onSuccess(responseBody);
    }
    catch (error) {
        batchImportValidationFailed.value = true;
        ElMessage.error(error?.response?.data?.msg || '导入校验请求失败');
        options.onError(error);
    }
    finally {
        batchImportLoading.value = false;
    }
}
async function confirmBatchImport() {
    if (!authStore.token || !batchImportFile.value || batchImportValidationFailed.value)
        return;
    batchImportConfirmLoading.value = true;
    try {
        const response = await batchImportStreams(authStore.token, batchImportFile.value, false);
        if (Number(response.data?.code) !== 200) {
            ElMessage.error(response.data?.msg || '批量导入失败，已全部回滚');
            return;
        }
        ElMessage.success(`批量导入成功，共 ${response.data?.return?.successCount ?? batchImportPreview.value.total} 个数据池`);
        batchImportVisible.value = false;
        streamPage.value = 1;
        await loadStreamRows();
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '批量导入失败，已全部回滚');
    }
    finally {
        batchImportConfirmLoading.value = false;
    }
}
function extractGatewayModules(root) {
    const data = root?.data ?? root;
    const direct = data?.gatewayModules ?? root?.gatewayModules;
    if (Array.isArray(direct))
        return direct;
    const rows = Array.isArray(data)
        ? data
        : (Array.isArray(data?.data)
            ? data.data
            : (data?.data ? [data.data] : (data?.list ?? data?.rows ?? [])));
    if (!Array.isArray(rows))
        return [];
    return rows.flatMap((row) => Array.isArray(row?.gatewayModules) ? row.gatewayModules : []);
}
function collectGatewayModulePoints(root, ttdcAdr = '', modelHint = '') {
    const hintModel = String(modelHint || '').trim().toUpperCase();
    const rawRows = Array.isArray(root)
        ? root
        : (Array.isArray(root?.data)
            ? root.data
            : (root?.data ? [root.data] : (root?.list ?? root?.rows ?? [])));
    const rows = Array.isArray(rawRows) ? rawRows : [];
    const out = [];
    const seen = new Set();
    const pushPoint = (p, model = '') => {
        const name = String(p?.name ?? p?.dataName ?? p?.pointName ?? p?.label ?? '').trim();
        if (!name)
            return;
        const dataId = String(p?.dataId ?? p?.id ?? '').trim();
        const key = `${model || 'UNKNOWN'}::${name}`;
        if (seen.has(key))
            return;
        seen.add(key);
        const dataType = normalizeStreamDataType(p?.dataType ?? p?.type ?? 1, p);
        out.push({ name, dataId, dataType, model });
    };
    const pushArr = (arr, model = '') => {
        if (!Array.isArray(arr))
            return;
        for (const p of arr)
            pushPoint(p, model);
    };
    const directModules = extractGatewayModules(root);
    if (directModules.length && hintModel) {
        for (const mod of directModules) {
            if (hintModel === 'TTDC') {
                const adr = String(mod?.ttdcConfig?.moduleAdr ?? '').trim();
                if (ttdcAdr && adr !== ttdcAdr)
                    continue;
            }
            pushArr(mod?.list, hintModel);
            pushArr(mod?.data, hintModel);
            pushArr(mod?.pointList, hintModel);
        }
        return out;
    }
    for (const row of rows) {
        const model = String(row?.model ?? '').trim().toUpperCase();
        if (model === 'TTDC') {
            const modules = Array.isArray(row?.gatewayModules) ? row.gatewayModules : [];
            for (const mod of modules) {
                const adr = String(mod?.ttdcConfig?.moduleAdr ?? '').trim();
                if (ttdcAdr && adr !== ttdcAdr)
                    continue;
                pushArr(mod?.list, model);
                pushArr(mod?.data, model);
                pushArr(mod?.pointList, model);
            }
            continue;
        }
        pushArr(row?.data, model);
        pushArr(row?.gatewayDatas, model);
        pushArr(row?.points, model);
        pushArr(row?.pointList, model);
        if (Array.isArray(row?.gateway)) {
            for (const g of row.gateway) {
                pushArr(g?.data, model);
                pushArr(g?.gatewayDatas, model);
                pushArr(g?.points, model);
            }
        }
    }
    return out;
}
function stripTtdcAddressPrefix(dataName, ttdcAdr) {
    const name = String(dataName || '').trim();
    const adr = String(ttdcAdr || '').trim();
    const prefix = adr ? `${adr}_` : '';
    return prefix && name.startsWith(prefix) ? name.slice(prefix.length) : name;
}
async function autoCreateStreamLabelsForNewPool(payload) {
    const { token, streamName, apiKey, streamId, equipmentId, gatewayId, gatewayName, gatewayModel, ttdcAdr } = payload;
    const gid = String(gatewayId || '').trim();
    if (!gid)
        return { ok: false, msg: '缺少绑定网关，无法自动创建标签', created: 0 };
    const { data } = await fetchGatewayModules({ token, gatewayId: gid });
    if (data?.code !== 200) {
        return { ok: false, msg: data?.msg || '读取网关配置失败，无法自动创建标签', created: 0 };
    }
    const root = data?.return ?? data;
    const points = collectGatewayModulePoints(root, ttdcAdr, gatewayModel);
    if (!points.length) {
        return { ok: true, created: 0, msg: '网关下未读取到数据点，已跳过自动创建标签' };
    }
    const labelResolvers = new Map();
    const getLabelName = (point) => {
        const model = String(point.model || '').trim().toUpperCase();
        if (!labelResolvers.has(model)) {
            labelResolvers.set(model, createDefaultLabelNameResolver(model));
        }
        const resolveLabelName = labelResolvers.get(model);
        const matchName = model === 'TTDC' ? stripTtdcAddressPrefix(point.name, ttdcAdr) : point.name;
        return resolveLabelName?.(matchName) || matchName;
    };
    const body = points.map(p => {
        return {
            streamId,
            labelId: '',
            name: getLabelName(p),
            gatewayName,
            gatewayId: gid,
            dataName: p.name,
            dataId: p.dataId,
            formula: 'x',
            unformula: 'x',
            dataType: p.dataType,
        };
    });
    const { data: labelResp } = await createStreamLabels({
        token,
        streamName,
        apiKey,
        body,
    });
    if (labelResp?.code !== 200) {
        return { ok: false, msg: labelResp?.msg || '自动创建标签失败', created: 0 };
    }
    const created = body.length;
    const sync = await syncEquipmentDataDetailsFromStreamLabels({
        token,
        equipmentId,
        streamName,
        labels: body,
    });
    if (!sync.ok) {
        return { ok: false, created, msg: `已自动创建 ${created} 个标签，${sync.msg}` };
    }
    return { ok: true, created, msg: `已自动创建 ${created} 个标签，${sync.msg}` };
}
async function submitStreamForm() {
    if (!authStore.token) {
        ElMessage.error('未登录或登录已失效');
        return;
    }
    if (!streamForm.value.name.trim()) {
        ElMessage.warning('请输入数据池名');
        return;
    }
    if (streamFormMode.value === 'add' && !streamForm.value.gatewayId) {
        ElMessage.warning('请选择绑定网关');
        return;
    }
    if (!streamForm.value.streamCode) {
        ElMessage.warning('请选择电池id');
        return;
    }
    const selectedGateway = selectedStreamGateway.value;
    const selectedGatewayModel = String(selectedGateway?.model ?? '').trim().toUpperCase();
    if (streamFormMode.value === 'add' && selectedGatewayModel === 'TTDC' && !String(streamForm.value.ttdcAdr || '').trim()) {
        ElMessage.warning('请选择TTDC地址');
        return;
    }
    const admin = String(authStore.username || '').trim();
    if (!admin) {
        ElMessage.warning('当前用户名称为空，无法提交');
        return;
    }
    streamFormLoading.value = true;
    try {
        const batteryId = String(streamForm.value.streamCode).trim();
        const { data: equipFindResp } = await findEquipmentPage({
            token: authStore.token,
            pageNum: 1,
            pageSize: 100000,
            filterFunc: 'BinC',
        });
        const equipPayload = equipFindResp?.return ?? {};
        const equipList = equipPayload?.data ?? equipPayload?.list ?? equipPayload?.rows ?? [];
        const target = Array.isArray(equipList)
            ? equipList.find((item) => String(item?.equipmentCode ?? '').trim() === batteryId)
            : null;
        if (!target) {
            ElMessage.error(`未找到电池ID为 ${batteryId} 的设备，不能创建数据池`);
            return;
        }
        const isEdit = streamFormMode.value === 'edit';
        const requestApiKey = isEdit ? streamEditApiKey.value : '';
        if (isEdit && !requestApiKey) {
            ElMessage.error('缺少原始ApiKey，无法修改数据池');
            return;
        }
        const { data } = isEdit
            ? await updateStream({
                token: authStore.token,
                apiKey: requestApiKey,
                bodyApiKey: String(streamForm.value.apiKey || '').trim() || requestApiKey,
                admin,
                name: streamForm.value.name.trim(),
                streamCode: batteryId,
            })
            : await createStream({
                token: authStore.token,
                admin,
                name: streamForm.value.name.trim(),
                streamCode: batteryId,
            });
        if (data?.code === 200) {
            const returned = data?.return?.data ?? data?.return ?? {};
            const effectiveApiKey = String(returned?.apiKey ??
                returned?.apikey ??
                streamForm.value.apiKey ??
                requestApiKey).trim();
            const effectiveStreamIdRaw = returned?.id ??
                returned?.streamId ??
                returned?.streamID ??
                streamForm.value.streamId ??
                streamForm.value.id;
            const effectiveStreamId = Number(effectiveStreamIdRaw);
            if (!(effectiveApiKey && Number.isFinite(effectiveStreamId) && effectiveStreamId > 0)) {
                ElMessage.error(isEdit ? '数据池修改返回数据不完整，设备未同步' : '数据池创建返回数据不完整，设备未绑定');
                return;
            }
            const body = {
                ...target,
                apiKey: effectiveApiKey,
                equipmentCode: batteryId,
                equipmentDetails: {
                    ...(target?.equipmentDetails ?? {}),
                    streamId: effectiveStreamId,
                },
            };
            const { data: equipUpdateResp } = await updateEquipment({ token: authStore.token, body });
            if (equipUpdateResp?.code !== 200) {
                ElMessage.error(equipUpdateResp?.msg || (isEdit ? '设备同步失败，修改不生效' : '设备绑定更新失败，创建不生效'));
                return;
            }
            if (!isEdit) {
                const gatewayName = String(selectedGateway?.name ?? '').trim();
                const auto = await autoCreateStreamLabelsForNewPool({
                    token: authStore.token,
                    streamName: streamForm.value.name.trim(),
                    apiKey: effectiveApiKey,
                    streamId: effectiveStreamId,
                    equipmentId: Number(target?.id || 0),
                    gatewayId: String(streamForm.value.gatewayId || '').trim(),
                    gatewayName: gatewayName || String(streamForm.value.gatewayId || '').trim(),
                    gatewayModel: selectedGatewayModel,
                    ttdcAdr: String(streamForm.value.ttdcAdr || '').trim(),
                });
                if (!auto.ok) {
                    ElMessage.warning(`${data?.msg || '添加数据池成功，设备绑定已完成'}；${auto.msg}`);
                }
                else {
                    ElMessage.success(`${data?.msg || '添加数据池成功，设备绑定已完成'}；${auto.msg}`);
                }
            }
            else {
                ElMessage.success(data?.msg || '修改数据池成功，设备已同步');
            }
            closeStreamDrawer();
            streamPage.value = 1;
            void loadStreamRows();
            return;
        }
        ElMessage.error(data?.msg || '添加数据池失败');
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '添加数据池请求失败');
    }
    finally {
        streamFormLoading.value = false;
    }
}
async function copyApiKey(apiKey) {
    if (!apiKey) {
        ElMessage.warning('该条数据无 apiKey');
        return;
    }
    try {
        if (navigator.clipboard && window.isSecureContext) {
            await navigator.clipboard.writeText(apiKey);
        }
        else {
            const textarea = document.createElement('textarea');
            textarea.value = apiKey;
            textarea.style.position = 'fixed';
            textarea.style.left = '-9999px';
            document.body.appendChild(textarea);
            textarea.focus();
            textarea.select();
            document.execCommand('copy');
            document.body.removeChild(textarea);
        }
        ElMessage.success('apiKey 已复制');
    }
    catch {
        ElMessage.error('复制失败');
    }
}
function normalizeGatewayRow(item) {
    const gid = item?.gatewayId ??
        item?.GatewayID ??
        item?.gatewayID ??
        item?.gateway_id ??
        item?.serialNo ??
        item?.serial ??
        item?.sn ??
        '';
    return {
        id: item?.id ?? '',
        name: String(item?.name ?? ''),
        gatewayId: String(gid ?? '').trim(),
        model: normalizeGatewayModelString(pickGatewayModelField(item)),
        card: String(item?.card ?? ''),
    };
}
async function loadGatewayRows() {
    if (!authStore.token)
        return;
    gatewayLoading.value = true;
    try {
        const req = {
            token: authStore.token,
            pageNum: gatewayPage.value,
            pageSize: gatewayPageSize.value,
            name: gatewayName.value.trim(),
        };
        const { data } = await fetchGatewayPage(req);
        if (data?.code === 400) {
            ElMessage.error(data?.msg || '获取网关数据失败');
            return;
        }
        const payload = data?.return ?? {};
        const records = payload?.data ?? payload?.records ?? payload?.list ?? payload?.rows ?? payload?.items ?? payload;
        gatewayRows.value = Array.isArray(records) ? records.map(normalizeGatewayRow) : [];
        gatewayTotal.value = Number(payload?.total ?? payload?.count ?? gatewayRows.value.length);
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '网关数据请求失败');
    }
    finally {
        gatewayLoading.value = false;
    }
}
async function loadGatewayOptions() {
    if (!authStore.token)
        return;
    gatewayOptionsLoading.value = true;
    try {
        const { data } = await fetchGatewayPage({
            token: authStore.token,
            pageNum: 1,
            pageSize: 5000,
            name: '',
        });
        if (data?.code === 400) {
            return;
        }
        const payload = data?.return ?? {};
        const records = payload?.data ?? payload?.records ?? payload?.list ?? payload?.rows ?? payload?.items ?? payload;
        gatewayOptions.value = Array.isArray(records) ? records.map(normalizeGatewayRow) : [];
    }
    catch {
        // 不阻断主页面，抽屉打开时会看到空选项
    }
    finally {
        gatewayOptionsLoading.value = false;
    }
}
function onGatewaySelectVisibleChange(visible) {
    if (visible) {
        void loadGatewayOptions();
    }
}
async function loadTtdcAddressOptions(gatewayId) {
    const gid = String(gatewayId || '').trim();
    ttdcAddressOptions.value = [];
    streamForm.value.ttdcAdr = '';
    if (!authStore.token || !gid)
        return;
    ttdcAddressOptionsLoading.value = true;
    try {
        const { data } = await fetchGatewayModules({ token: authStore.token, gatewayId: gid });
        if (data?.code !== 200) {
            ElMessage.warning(data?.msg || '读取TTDC地址失败');
            return;
        }
        const root = data?.return ?? data;
        const modules = extractGatewayModules(root);
        const set = new Set();
        for (const item of modules) {
            const adr = String(item?.ttdcConfig?.moduleAdr ?? '').trim();
            if (adr)
                set.add(adr);
        }
        ttdcAddressOptions.value = Array.from(set);
        if (!ttdcAddressOptions.value.length) {
            ElMessage.warning('当前网关未读取到TTDC地址，无法自动创建标签');
        }
    }
    finally {
        ttdcAddressOptionsLoading.value = false;
    }
}
function onStreamGatewayChange(gatewayId) {
    streamForm.value.ttdcAdr = '';
    ttdcAddressOptions.value = [];
    const hit = gatewayOptions.value.find(item => String(item.gatewayId || '').trim() === String(gatewayId || '').trim());
    if (String(hit?.model ?? '').trim().toUpperCase() === 'TTDC') {
        void loadTtdcAddressOptions(gatewayId);
    }
}
async function loadEquipmentCodeOptions() {
    if (!authStore.token)
        return;
    equipmentOptionsLoading.value = true;
    try {
        const { data } = await findEquipmentPage({
            token: authStore.token,
            pageNum: 1,
            pageSize: 100000,
            filterFunc: 'BinC',
        });
        if (data?.code === 400)
            return;
        const payload = data?.return ?? {};
        const rows = payload?.data ?? payload?.list ?? payload?.rows ?? [];
        const set = new Set();
        if (Array.isArray(rows)) {
            for (const item of rows) {
                const code = String(item?.equipmentCode ?? '').trim();
                if (code)
                    set.add(code);
            }
        }
        equipmentCodeOptions.value = Array.from(set);
    }
    finally {
        equipmentOptionsLoading.value = false;
    }
}
function onEquipmentCodeVisibleChange(visible) {
    if (visible) {
        void loadEquipmentCodeOptions();
    }
}
function normalizeStreamRow(item) {
    const getValue = (source, keys) => {
        const dict = {};
        Object.keys(source || {}).forEach((k) => {
            dict[k.toLowerCase()] = source[k];
        });
        for (const key of keys) {
            const hit = dict[key.toLowerCase()];
            if (hit !== undefined && hit !== null)
                return hit;
        }
        return '';
    };
    const streamId = getValue(item || {}, [
        'streamCode',
        'stream_code',
        'batteryId',
        'batteryID',
        'battery_id',
        '电池id',
        '电池ID',
    ]);
    return {
        id: Number(getValue(item || {}, ['id', 'streamId', 'streamID']) || 0),
        name: String(getValue(item || {}, ['name', 'poolName', 'streamName', 'pool_name'])),
        admin: String(getValue(item || {}, ['admin', 'adminName', 'manager', 'owner'])),
        streamCode: String(streamId),
        streamId: Number(getValue(item || {}, ['id', 'streamId', 'streamID']) || 0),
        batteryName: '',
        apiKey: String(getValue(item || {}, ['apiKey', 'apikey', 'api_key'])),
    };
}
async function loadStreamRows() {
    if (!authStore.token)
        return;
    streamLoading.value = true;
    try {
        const { data } = await fetchStreamPage({
            token: authStore.token,
            page: streamPage.value,
            size: streamPageSize.value,
            search: streamKeyword.value.trim(),
        });
        if (data?.code === 400) {
            ElMessage.error(data?.msg || '获取数据池数据失败');
            return;
        }
        const payload = data?.return ?? {};
        const records = payload?.stream ?? payload?.data ?? payload?.list ?? payload?.rows ?? [];
        const rows = Array.isArray(records) ? records.map(normalizeStreamRow) : [];
        try {
            const { data: equipResp } = await findEquipmentPage({
                token: authStore.token,
                pageNum: 1,
                pageSize: 100000,
                filterFunc: 'BinC',
            });
            const equipPayload = equipResp?.return ?? {};
            const equipList = equipPayload?.data ?? equipPayload?.list ?? equipPayload?.rows ?? [];
            const streamCodeToEquipmentCode = new Map();
            if (Array.isArray(equipList)) {
                for (const item of equipList) {
                    const equipmentCode = String(item?.equipmentCode ?? item?.streamCode ?? '').trim();
                    if (equipmentCode && !streamCodeToEquipmentCode.has(equipmentCode)) {
                        streamCodeToEquipmentCode.set(equipmentCode, equipmentCode);
                    }
                }
            }
            for (const row of rows) {
                const streamCode = String(row.streamCode ?? '').trim();
                row.batteryName = (streamCode && streamCodeToEquipmentCode.get(streamCode)) || row.streamCode;
            }
        }
        catch {
            for (const row of rows)
                row.batteryName = row.streamCode;
        }
        streamRows.value = rows;
        streamTotal.value = Number(payload?.total ?? payload?.count ?? streamRows.value.length);
    }
    catch (error) {
        ElMessage.error(error?.response?.data?.msg || '数据池数据请求失败');
    }
    finally {
        streamLoading.value = false;
    }
}
watch(() => route.query, () => {
    if (route.name !== 'collection-home')
        return;
    applyStateFromQuery();
    if (activeModule.value === 'gateway') {
        void loadGatewayRows();
    }
    else {
        void loadStreamRows();
    }
}, { deep: true });
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
    if (route.name === 'collection-home' && route.query.module == null) {
        void router.replace({ path: '/collection', query: { module: 'gateway' } });
    }
    applyStateFromQuery();
    if (activeModule.value === 'gateway') {
        void loadGatewayRows();
    }
    else {
        void loadStreamRows();
    }
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
  <div class="workbench" :class="{ 'is-dark': isDarkTheme }">
        <div class="card" :class="{ 'is-dark': isDarkTheme }">
          <div v-if="activeModule === 'gateway'" class="toolbar">
            <div class="tool-left">
              <span class="toolbar-title">网关总量</span>
              <span class="toolbar-count">{{ gatewayTotal }}</span>
              <el-input
                v-model="gatewayName"
                class="tool keyword-input"
                placeholder="请输入网关名称"
                clearable
                @clear="onGatewaySearch"
                @keyup.enter="onGatewaySearch"
              />
              <el-button type="primary" @click="onGatewaySearch">查询</el-button>
              <el-button @click="onGatewayReset">重置</el-button>
            </div>
            <div class="tool-right">
              <el-button type="danger" :disabled="!selectedGatewayIds.length" :loading="gatewayBatchDeleting" @click="onBatchDeleteGateways">批量删除</el-button>
              <el-button type="primary" @click="openGatewayDrawer">添加网关</el-button>
            </div>
          </div>

          <div v-if="activeModule === 'stream'" class="toolbar">
            <div class="tool-left">
              <span class="toolbar-title">数据池总量</span>
              <span class="toolbar-count">{{ streamTotal }}</span>
              <el-input
                v-model="streamKeyword"
                class="tool keyword-input"
                placeholder="请输入数据池名/管理员"
                clearable
                @clear="onStreamSearch"
                @keyup.enter="onStreamSearch"
              />
              <el-button type="primary" @click="onStreamSearch">查询</el-button>
              <el-button @click="onStreamReset">重置</el-button>
            </div>
            <div class="tool-right">
              <el-button
                type="danger"
                :disabled="!selectedStreamApiKeys.length"
                :loading="streamBatchDeleting"
                @click="onBatchDeleteStreams"
              >批量删除</el-button>
              <el-button @click="downloadBatchImportTemplate">下载模板</el-button>
              <el-button type="primary" plain @click="openBatchImport">批量导入</el-button>
              <el-button type="primary" @click="openStreamDrawer">添加数据池</el-button>
            </div>
          </div>

          <div class="table-area">
            <el-table
              v-if="activeModule === 'gateway'"
              :data="gatewayRows"
              :loading="gatewayLoading || gatewayBatchDeleting"
              stripe
              fit
              style="width: 100%"
              @selection-change="onGatewaySelectionChange"
            >
              <el-table-column type="selection" width="46" />
              <el-table-column prop="name" label="网关名称" min-width="250" />
              <el-table-column prop="gatewayId" label="序列号" min-width="190" />
              <el-table-column prop="model" label="机型" min-width="170" />
              <el-table-column prop="card" label="物联卡号" min-width="240" />
              <el-table-column label="操作" min-width="390">
                <template #default="{ row }">
                  <el-button size="small" plain @click="goGatewayDetail(row)">详情</el-button>
                <el-button size="small" type="primary" @click="openGatewayEditDrawer(row)">修改</el-button>
                  <el-button size="small" type="info" @click="openGatewayCopyDrawer(row)">复制</el-button>
                  <el-button size="small" type="danger" @click="onDeleteGateway(row.gatewayId)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>

            <el-table
              v-if="activeModule === 'stream'"
              :data="filteredStreamRows"
              :loading="streamLoading || streamBatchDeleting"
              stripe
              fit
              style="width: 100%"
              @selection-change="onStreamSelectionChange"
            >
              <el-table-column type="selection" width="46" />
              <el-table-column prop="name" label="数据池名称" min-width="240" />
              <el-table-column prop="admin" label="管理员" min-width="140" />
              <el-table-column prop="streamCode" label="电池ID" min-width="210" />
              <el-table-column prop="batteryName" label="电池名称" min-width="210" />
              <el-table-column label="操作" min-width="390">
                <template #default="{ row }">
                  <el-button size="small" plain @click="goStreamDetail(row)">详情</el-button>
                  <el-button size="small" type="warning" @click="copyApiKey(row.apiKey)">复制apiKey</el-button>
                  <el-button size="small" type="primary" @click="openStreamEditDrawer(row)">修改</el-button>
                  <el-button size="small" type="danger" @click="onDeleteStream(row.apiKey)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>

            <div class="pager">
              <el-pagination
                v-if="activeModule === 'gateway'"
                v-model:current-page="gatewayPage"
                v-model:page-size="gatewayPageSize"
                background
                layout="total, sizes, prev, pager, next, jumper"
                :page-sizes="[10, 20, 50, 100]"
                :total="gatewayTotal"
                @size-change="loadGatewayRows"
                @current-change="loadGatewayRows"
              />
              <el-pagination
                v-else
                v-model:current-page="streamPage"
                v-model:page-size="streamPageSize"
                background
                layout="total, sizes, prev, pager, next, jumper"
                :page-sizes="[10, 20, 50, 100]"
                :total="streamTotal"
                @size-change="loadStreamRows"
                @current-change="loadStreamRows"
              />
            </div>
          </div>
        </div>
        <el-drawer
          v-model="gatewayDrawerVisible"
          :title="gatewayFormMode === 'edit' ? '修改网关' : gatewayFormMode === 'copy' ? '复制网关' : '添加网关'"
          direction="rtl"
          size="420px"
          :destroy-on-close="false"
        >
          <el-form label-width="90px" label-position="right" class="gateway-drawer-form">
            <el-form-item label="网关名" required>
              <el-input v-model="gatewayForm.name" placeholder="请输入网关名" />
            </el-form-item>
            <el-form-item label="序列号" required>
              <el-input v-model="gatewayForm.gatewayId" :disabled="gatewayFormMode === 'edit'" placeholder="请输入序列号" />
            </el-form-item>
            <el-form-item label="机型">
              <el-select
                v-model="gatewayForm.model"
                class="drawer-full"
                :disabled="gatewayFormMode === 'edit' || gatewayFormMode === 'copy'"
                :teleported="false"
              >
                <el-option label="TTDC" value="TTDC" />
              </el-select>
            </el-form-item>
            <el-form-item label="物联卡号">
              <el-input v-model="gatewayForm.card" placeholder="请输入物联卡号" />
            </el-form-item>
          </el-form>
          <template #footer>
            <div class="drawer-footer">
              <el-button @click="closeGatewayDrawer">取消</el-button>
              <el-button type="primary" :loading="gatewayFormLoading" @click="submitGatewayForm">确定</el-button>
            </div>
          </template>
        </el-drawer>
        <el-dialog v-model="batchImportVisible" title="批量导入数据池" width="680px" destroy-on-close>
          <div class="batch-import-content">
            <el-upload
              :show-file-list="false"
              accept=".xlsx"
              :http-request="uploadBatchImport"
              :disabled="batchImportLoading || batchImportConfirmLoading"
            >
              <el-button :loading="batchImportLoading">选择Excel并校验</el-button>
            </el-upload>
            <div v-if="batchImportFile" class="batch-import-file">当前文件：{{ batchImportFile.name }}</div>
            <el-alert
              v-if="batchImportValidationFailed"
              class="batch-import-result"
              title="校验失败，整张表禁止导入"
              type="error"
              :closable="false"
              show-icon
            />
            <el-alert
              v-else-if="batchImportPreview.total"
              class="batch-import-result"
              :title="`校验通过，共 ${batchImportPreview.total} 行，可确认导入`"
              type="success"
              :closable="false"
              show-icon
            />
          </div>
          <template #footer>
            <el-button @click="batchImportVisible = false">取消</el-button>
            <el-button
              type="primary"
              :loading="batchImportConfirmLoading"
              :disabled="!batchImportFile || !batchImportPreview.total || batchImportValidationFailed"
              @click="confirmBatchImport"
            >确认导入</el-button>
          </template>
        </el-dialog>
        <el-drawer
          v-model="streamDrawerVisible"
          :title="streamFormMode === 'edit' ? '修改数据池' : '添加数据池'"
          direction="rtl"
          size="420px"
          :destroy-on-close="false"
        >
          <el-form label-width="90px" label-position="right" class="gateway-drawer-form">
            <el-form-item label="数据池名" required>
              <el-input v-model="streamForm.name" placeholder="请输入数据池名" />
            </el-form-item>
            <el-form-item v-if="streamFormMode === 'add'" label="绑定网关">
              <el-select
                v-model="streamForm.gatewayId"
                class="drawer-full"
                placeholder="请选择"
                filterable
                clearable
                :teleported="false"
                :loading="gatewayOptionsLoading"
                @visible-change="onGatewaySelectVisibleChange"
                @change="onStreamGatewayChange"
              >
                <el-option
                  v-for="item in gatewayOptions"
                  :key="`gw-${item.gatewayId}`"
                  :label="item.name"
                  :value="item.gatewayId"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="电池id">
              <el-select
                v-model="streamForm.streamCode"
                class="drawer-full"
                placeholder="请选择"
                filterable
                clearable
                :teleported="false"
                :loading="equipmentOptionsLoading"
                @visible-change="onEquipmentCodeVisibleChange"
              >
                <el-option
                  v-for="code in equipmentCodeOptions"
                  :key="`eq-${code}`"
                  :label="code"
                  :value="code"
                />
              </el-select>
            </el-form-item>
            <el-form-item v-if="streamFormMode === 'add' && isSelectedStreamGatewayTtdc" label="TTDC地址" required>
              <el-select
                v-model="streamForm.ttdcAdr"
                class="drawer-full"
                placeholder="请选择TTDC地址"
                filterable
                clearable
                :teleported="false"
                :loading="ttdcAddressOptionsLoading"
              >
                <el-option
                  v-for="adr in ttdcAddressOptions"
                  :key="`ttdc-adr-${adr}`"
                  :label="adr"
                  :value="adr"
                />
              </el-select>
            </el-form-item>
          </el-form>
          <template #footer>
            <div class="drawer-footer">
              <el-button @click="closeStreamDrawer">取消</el-button>
              <el-button type="primary" :loading="streamFormLoading" @click="submitStreamForm">确定</el-button>
            </div>
          </template>
        </el-drawer>
  </div>
</template>

<style scoped>
.workbench {
  width: 100%;
  max-width: 100%;
  /* 按正文实际可用宽度适配，侧栏展开和浏览器缩放也能触发。 */
  container-type: inline-size;
  padding: 16px 20px;
  box-sizing: border-box;
}

.workbench.is-dark {
  background: transparent;
}

.card {
  --content-x: 24px;
  background: #fff;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid #e4e7ed;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.card.is-dark {
  background: linear-gradient(180deg, #13253d 0%, #0f1d31 100%);
  border-color: #294362;
  box-shadow:
    inset 0 1px 0 rgba(148, 163, 184, 0.1),
    0 10px 22px rgba(2, 6, 23, 0.34);
}

.toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  width: 100%;
  box-sizing: border-box;
  padding: 16px var(--content-x);
  border-bottom: 1px solid #f0f0f0;
  background: #fafafa;
}

.tool-left {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  flex: 1;
  min-width: 0;
}

.tool-right {
  display: flex;
  align-items: center;
  gap: 6px;
  justify-self: end;
  white-space: nowrap;
}

.toolbar-title {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.65);
  margin-right: 2px;
}

.toolbar-count {
  min-width: 22px;
  padding: 0 8px;
  line-height: 22px;
  border-radius: 4px;
  border: 1px solid #dcdfe6;
  background: #fff;
  color: #1890ff;
  text-align: center;
  font-size: 13px;
  font-weight: 600;
  margin-right: 8px;
}

.tool {
  width: 260px;
}

.toolbar :deep(.el-select) {
  width: 140px;
}

.keyword-input {
  flex: 0 1 360px;
  min-width: 0;
  width: 360px;
}

.keyword-input :deep(.el-input) {
  width: 100%;
}

.gateway-drawer-form {
  padding-right: 8px;
}

.drawer-full {
  width: 100%;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.table-area {
  width: 100%;
  box-sizing: border-box;
  padding: 0 var(--content-x);
}

.table-area :deep(.el-table) {
  border-radius: 0;
  width: 100%;
}

.table-area :deep(.el-table__header-wrapper th) {
  background: #fafafa;
  color: rgba(0, 0, 0, 0.85);
  font-weight: 600;
  font-size: 14px;
  height: 54px;
}

.table-area :deep(.el-table__row td) {
  padding-top: 15px;
  padding-bottom: 15px;
}

.table-area :deep(.el-table__body td) {
  font-size: 15px;
}

.table-area :deep(.el-table .cell) {
  padding-left: 14px;
  padding-right: 14px;
  box-sizing: border-box;
  text-align: center;
}

.table-area :deep(.el-button--small) {
  padding: 9px 13px;
  font-size: 14px;
}

.pager {
  margin-top: 0;
  width: 100%;
  box-sizing: border-box;
  padding: 12px var(--content-x);
  border-top: 1px solid #f0f0f0;
  background: #fafafa;
  display: flex;
  justify-content: flex-start;
}

.card.is-dark .toolbar {
  background: rgba(15, 29, 48, 0.95);
  border-bottom-color: #294362;
}

.card.is-dark .toolbar-title {
  color: #9fb2ca;
}

.card.is-dark .toolbar-count {
  color: #d8e9ff;
  background: rgba(20, 42, 66, 0.75);
  border-color: rgba(88, 175, 255, 0.3);
}

.card.is-dark .toolbar :deep(.el-input__wrapper) {
  background: rgba(15, 29, 48, 0.95) !important;
  box-shadow: 0 0 0 1px rgba(71, 95, 124, 0.85) inset !important;
}

.card.is-dark .toolbar :deep(.el-input__wrapper:hover),
.card.is-dark .toolbar :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px rgba(96, 165, 250, 0.65) inset !important;
}

.card.is-dark .toolbar :deep(.el-input__inner) {
  color: #e6eef9 !important;
}

.card.is-dark .toolbar :deep(.el-input__inner::placeholder) {
  color: rgba(159, 178, 202, 0.78) !important;
}

.card.is-dark .table-area :deep(.el-table) {
  --el-table-bg-color: #0f1d31;
  --el-table-tr-bg-color: #0f1d31;
  --el-table-header-bg-color: #13253d;
  --el-table-row-hover-bg-color: #17304f;
  --el-table-border-color: #2d4663;
  --el-table-text-color: #d7e3f5;
  --el-table-header-text-color: #eef5ff;
}

.card.is-dark .table-area :deep(.el-table th.el-table__cell) {
  background: var(--el-table-header-bg-color) !important;
  color: var(--el-table-header-text-color) !important;
}

.card.is-dark .table-area :deep(.el-table td.el-table__cell) {
  background: var(--el-table-tr-bg-color) !important;
  color: var(--el-table-text-color) !important;
}

.card.is-dark .pager {
  background: rgba(15, 29, 48, 0.95);
  border-top-color: #294362;
}

.card.is-dark .pager :deep(.el-select__wrapper) {
  background: rgba(15, 29, 48, 0.95) !important;
  box-shadow: 0 0 0 1px rgba(71, 95, 124, 0.85) inset !important;
}

.card.is-dark .pager :deep(.el-select__wrapper:hover),
.card.is-dark .pager :deep(.el-select__wrapper.is-focused) {
  box-shadow: 0 0 0 1px rgba(96, 165, 250, 0.65) inset !important;
}

.card.is-dark .pager :deep(.el-select__selected-item),
.card.is-dark .pager :deep(.el-select__placeholder) {
  color: #dbe9ff !important;
}

.card.is-dark .pager :deep(.el-select__caret) {
  color: #a7bfdc !important;
}

.card.is-dark .pager :deep(.el-input__wrapper) {
  background: rgba(15, 29, 48, 0.95) !important;
  box-shadow: 0 0 0 1px rgba(71, 95, 124, 0.85) inset !important;
}

.card.is-dark .pager :deep(.el-input__wrapper:hover),
.card.is-dark .pager :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px rgba(96, 165, 250, 0.65) inset !important;
}

.card.is-dark .pager :deep(.el-input__inner) {
  color: #dbe9ff !important;
}

.card.is-dark .table-area :deep(.el-checkbox__inner) {
  background: rgba(15, 29, 48, 0.95) !important;
  border-color: rgba(71, 95, 124, 0.85) !important;
}

.card.is-dark .table-area :deep(.el-checkbox__input:hover .el-checkbox__inner) {
  border-color: rgba(96, 165, 250, 0.75) !important;
}

.card.is-dark .table-area :deep(.el-checkbox__input.is-checked .el-checkbox__inner),
.card.is-dark .table-area :deep(.el-checkbox__input.is-indeterminate .el-checkbox__inner) {
  background: linear-gradient(180deg, #225b8e 0%, #1a4f80 100%) !important;
  border-color: #5fa9e6 !important;
}

.card.is-dark .table-area :deep(.el-checkbox__input.is-disabled .el-checkbox__inner) {
  background: rgba(17, 33, 53, 0.78) !important;
  border-color: rgba(71, 95, 124, 0.55) !important;
}

.workbench.is-dark :deep(.el-overlay.is-drawer .el-drawer) {
  background: linear-gradient(180deg, #13253d 0%, #0f1d31 100%) !important;
  border-left: 1px solid #294362 !important;
}

.workbench.is-dark :deep(.el-overlay.is-drawer .el-drawer__header) {
  color: #e6eef9 !important;
  border-bottom: 1px solid #294362;
  margin-bottom: 0;
  padding-bottom: 16px;
}

.workbench.is-dark :deep(.el-overlay.is-drawer .el-drawer__body) {
  color: #dbe9ff !important;
}

.workbench.is-dark :deep(.el-overlay.is-drawer .el-form-item__label) {
  color: #9fb2ca !important;
}

.workbench.is-dark :deep(.el-overlay.is-drawer .el-input__wrapper),
.workbench.is-dark :deep(.el-overlay.is-drawer .el-select__wrapper) {
  background: rgba(15, 29, 48, 0.95) !important;
  box-shadow: 0 0 0 1px rgba(71, 95, 124, 0.85) inset !important;
}

.workbench.is-dark :deep(.el-overlay.is-drawer .el-input__inner),
.workbench.is-dark :deep(.el-overlay.is-drawer .el-select__selected-item),
.workbench.is-dark :deep(.el-overlay.is-drawer .el-select__placeholder) {
  color: #dbe9ff !important;
}

.workbench.is-dark :deep(.el-overlay.is-drawer .el-drawer__footer) {
  border-top: 1px solid #294362;
  background: rgba(15, 29, 48, 0.95);
}

.card.is-dark .pager :deep(.btn-prev),
.card.is-dark .pager :deep(.btn-next),
.card.is-dark .pager :deep(.el-pager li) {
  background: rgba(15, 29, 48, 0.95) !important;
  border: 1px solid rgba(71, 95, 124, 0.72) !important;
  color: #cfe1f8 !important;
}

.card.is-dark .pager :deep(.btn-prev:hover),
.card.is-dark .pager :deep(.btn-next:hover),
.card.is-dark .pager :deep(.el-pager li:hover) {
  color: #ffffff !important;
  border-color: rgba(96, 165, 250, 0.75) !important;
}

.card.is-dark .pager :deep(.el-pager li.is-active) {
  background: linear-gradient(180deg, #225b8e 0%, #1a4f80 100%) !important;
  border-color: #5fa9e6 !important;
  color: #ffffff !important;
}

/* 深色主题：Home 工作区整体改为深色 */
:global(html[data-theme='dark']) .workbench,
:global(html.dark) .workbench,
:global(body.dark) .workbench,
:global([data-theme='dark']) .workbench,
:global(.theme-dark) .workbench,
:global(.dark-theme) .workbench {
  background: transparent;
}

:global(html[data-theme='dark']) .workbench .card,
:global(html.dark) .workbench .card,
:global(body.dark) .workbench .card,
:global([data-theme='dark']) .workbench .card,
:global(.theme-dark) .workbench .card,
:global(.dark-theme) .workbench .card {
  background: linear-gradient(180deg, #13253d 0%, #0f1d31 100%);
  border-color: #294362;
  box-shadow:
    inset 0 1px 0 rgba(148, 163, 184, 0.1),
    0 10px 22px rgba(2, 6, 23, 0.34);
}

:global(html[data-theme='dark']) .workbench .toolbar,
:global(html.dark) .workbench .toolbar,
:global(body.dark) .workbench .toolbar,
:global([data-theme='dark']) .workbench .toolbar,
:global(.theme-dark) .workbench .toolbar,
:global(.dark-theme) .workbench .toolbar {
  background: rgba(15, 29, 48, 0.95);
  border-bottom-color: #294362;
}

:global(html[data-theme='dark']) .workbench .toolbar-title,
:global(html.dark) .workbench .toolbar-title,
:global(body.dark) .workbench .toolbar-title,
:global([data-theme='dark']) .workbench .toolbar-title,
:global(.theme-dark) .workbench .toolbar-title,
:global(.dark-theme) .workbench .toolbar-title {
  color: #9fb2ca;
}

:global(html[data-theme='dark']) .workbench .toolbar-count,
:global(html.dark) .workbench .toolbar-count,
:global(body.dark) .workbench .toolbar-count,
:global([data-theme='dark']) .workbench .toolbar-count,
:global(.theme-dark) .workbench .toolbar-count,
:global(.dark-theme) .workbench .toolbar-count {
  color: #d8e9ff;
  background: rgba(20, 42, 66, 0.75);
  border-color: rgba(88, 175, 255, 0.3);
}

:global(html[data-theme='dark']) .workbench .table-area .el-table,
:global(html.dark) .workbench .table-area .el-table,
:global(body.dark) .workbench .table-area .el-table,
:global([data-theme='dark']) .workbench .table-area .el-table,
:global(.theme-dark) .workbench .table-area .el-table,
:global(.dark-theme) .workbench .table-area .el-table {
  --el-table-bg-color: #0f1d31;
  --el-table-tr-bg-color: #0f1d31;
  --el-table-header-bg-color: #13253d;
  --el-table-row-hover-bg-color: #17304f;
  --el-table-border-color: #2d4663;
  --el-table-text-color: #d7e3f5;
  --el-table-header-text-color: #eef5ff;
}

:global(html[data-theme='dark']) .workbench .table-area .el-table th.el-table__cell,
:global(html.dark) .workbench .table-area .el-table th.el-table__cell,
:global(body.dark) .workbench .table-area .el-table th.el-table__cell,
:global([data-theme='dark']) .workbench .table-area .el-table th.el-table__cell,
:global(.theme-dark) .workbench .table-area .el-table th.el-table__cell,
:global(.dark-theme) .workbench .table-area .el-table th.el-table__cell {
  background: var(--el-table-header-bg-color) !important;
  color: var(--el-table-header-text-color) !important;
}

:global(html[data-theme='dark']) .workbench .table-area .el-table td.el-table__cell,
:global(html.dark) .workbench .table-area .el-table td.el-table__cell,
:global(body.dark) .workbench .table-area .el-table td.el-table__cell,
:global([data-theme='dark']) .workbench .table-area .el-table td.el-table__cell,
:global(.theme-dark) .workbench .table-area .el-table td.el-table__cell,
:global(.dark-theme) .workbench .table-area .el-table td.el-table__cell {
  background: var(--el-table-tr-bg-color) !important;
  color: var(--el-table-text-color) !important;
}

:global(html[data-theme='dark']) .workbench .pager,
:global(html.dark) .workbench .pager,
:global(body.dark) .workbench .pager,
:global([data-theme='dark']) .workbench .pager,
:global(.theme-dark) .workbench .pager,
:global(.dark-theme) .workbench .pager {
  background: rgba(15, 29, 48, 0.95);
  border-top-color: #294362;
}

@media (max-width: 980px) {
  .workbench {
    width: 100%;
  }
  .tool {
    width: 100%;
  }
  .toolbar :deep(.el-select) {
    width: 100%;
  }
}

/* 仅空间不足时把操作区排到下一行，宽屏保留原来的左右排列。 */
@container (max-width: 760px) {
  .toolbar {
    grid-template-columns: minmax(0, 1fr);
  }

  .tool-right {
    justify-self: start;
    flex-wrap: wrap;
    min-width: 0;
    max-width: 100%;
  }
}

/* 不支持容器查询的旧浏览器，仍按视口宽度提供窄屏兼容。 */
@supports not (container-type: inline-size) {
  @media (max-width: 1060px) {
    .toolbar {
      grid-template-columns: minmax(0, 1fr);
    }

    .tool-right {
      justify-self: start;
      flex-wrap: wrap;
      min-width: 0;
      max-width: 100%;
    }
  }
}
</style>
