<script setup>
import { computed, onMounted, ref } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { useAuthStore } from '../stores/auth';
import { TTDC_TEMPLATE_PERMISSIONS } from '../constants/permissions';
import { deleteTtdcGroup, deleteTtdcAlarmRule, deleteTtdcTemplate, exportTtdcTemplate, fetchTtdcGroupDetail, fetchTtdcGroupPage, fetchTtdcAlarmRuleDetail, fetchTtdcAlarmRulePage, fetchTtdcTemplateDetail, importTtdcTemplate, saveTtdcGroup, saveTtdcAlarmRule, saveTtdcTemplate, setTtdcGroupState, setTtdcAlarmRuleEnabled, setTtdcTemplateManualEnabled, } from '../api/ttdc-template';
const auth = useAuthStore();
const loading = ref(false);
const saving = ref(false);
const keyword = ref('');
const groups = ref([]);
const total = ref(0);
const pageNum = ref(1);
const pageSize = ref(20);
const selectedGroupId = ref();
const groupDialogVisible = ref(false);
const templateDialogVisible = ref(false);
const alarmRuleDialogVisible = ref(false);
const alarmRuleLoading = ref(false);
const alarmRuleSaving = ref(false);
const alarmRules = ref([]);
const alarmRuleTotal = ref(0);
const alarmRulePage = ref(1);
const alarmRulePageSize = ref(20);
const alarmRuleKeyword = ref('');
const pointPage = ref(1);
const pointPageSize = 25;
const canAddGroup = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.addGroup));
const canUpdateGroup = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.updateGroup));
const canChangeGroupState = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.changeGroupState, TTDC_TEMPLATE_PERMISSIONS.updateGroup));
const canDeleteGroup = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.deleteGroup));
const canAddTemplate = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.addTemplate));
const canUpdateTemplate = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.updateTemplate));
const canChangeTemplateState = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.changeTemplateState, TTDC_TEMPLATE_PERMISSIONS.updateTemplate));
const canDeleteTemplate = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.deleteTemplate));
const canImportTemplate = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.importTemplate));
const canExportTemplate = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.exportTemplate));
const canViewAlarmRule = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.viewAlarmRule, TTDC_TEMPLATE_PERMISSIONS.view));
const canAddAlarmRule = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.addAlarmRule));
const canUpdateAlarmRule = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.updateAlarmRule));
const canDeleteAlarmRule = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.deleteAlarmRule));
const canChangeAlarmRuleState = computed(() => auth.hasAnyPermission(TTDC_TEMPLATE_PERMISSIONS.changeAlarmRuleState, TTDC_TEMPLATE_PERMISSIONS.updateAlarmRule));
const canSaveAlarmRule = computed(() => alarmRuleForm.value.id ? canUpdateAlarmRule.value : canAddAlarmRule.value);
const canSaveGroup = computed(() => groupForm.value.id ? canUpdateGroup.value : canAddGroup.value);
const canSaveTemplate = computed(() => templateForm.value.id ? canUpdateTemplate.value : canAddTemplate.value);
const hasGroupOperations = computed(() => canUpdateGroup.value || canDeleteGroup.value);
const hasTemplateOperations = computed(() => canUpdateTemplate.value || canExportTemplate.value || canDeleteTemplate.value);
const hasAlarmRuleOperations = computed(() => canUpdateAlarmRule.value || canDeleteAlarmRule.value);
const emptyGroup = () => ({
    groupCode: '', groupName: '', topicSegment: '', enabled: true,
    runtimeMatchEnabled: false, autoCreateEnabled: false, version: 1,
    description: '', templates: [],
});
const groupForm = ref(emptyGroup());
const emptyPoint = (role, sortNo = 1) => ({
    sortNo, baseDataId: '', name: '', dataType: 1,
    address: role === '42_SOC' ? null : 0,
    byteCount: role === '42_SOC' ? null : 1,
    convertType: role === '42_SOC' ? null : 3,
    bitIndex: null, algorithmKey: null, transport: role === '42_SOC' ? '-' : '0 1 3',
    formula: 'x', unformula: 'x', enabled: true, description: '',
});
const emptyTemplate = (groupId = 0, role = '42') => ({
    groupId, templateCode: '', templateName: '', cid2Type: role,
    protocolCode: '', machineCid2: role === '42_SOC' ? '42_SOC' : role,
    parserProfile: role === '42_SOC' ? 'derived_algorithm' : 'raw_fields',
    enabled: true, autoCreateEnabled: false, version: 1, description: '',
    points: [emptyPoint(role)],
});
const templateForm = ref(emptyTemplate());
const emptyAlarmLogic = () => ({ logic: '<', triggerValue: 0 });
const alarmStageOptions = [
    { label: '一级', value: 1 },
    { label: '二级', value: 2 },
    { label: '三级', value: 3 },
];
const emptyAlarmRule = (groupId = 0) => ({
    templateGroupId: groupId, templateId: 0, templatePointId: 0, baseDataId: '', ruleName: '', ruleCode: '',
    alarmType: '0', alarmStage: 1, formula: 'x', cycle: 0, sustainTime: 0, enabled: true, version: 1,
    description: '', logics: [emptyAlarmLogic()],
});
const alarmRuleForm = ref(emptyAlarmRule());
const selectedGroup = computed(() => groups.value.find(item => item.id === selectedGroupId.value));
const sortedTemplates = computed(() => [...(selectedGroup.value?.templates ?? [])].sort((a, b) => roleOrder(a.cid2Type) - roleOrder(b.cid2Type)));
const pagedPoints = computed(() => {
    const start = (pointPage.value - 1) * pointPageSize;
    return templateForm.value.points.slice(start, start + pointPageSize);
});
const pointTotal = computed(() => templateForm.value.points.length);
const alarmPointOptions = computed(() => (selectedGroup.value?.templates ?? [])
    // Public rules use the data-pool label as their cross-battery key. Include
    // every numeric channel point; the selected point is only a management-time
    // reference for the label used by the runtime synchronizer.
    .filter(template => template.enabled !== false)
    .sort((a, b) => roleOrder(a.cid2Type) - roleOrder(b.cid2Type))
    .flatMap(template => (template.points ?? []).filter(point => point.enabled !== false && point.dataType === 1 && point.id && point.baseDataId.trim()).map(point => ({
    templateId: template.id ?? 0, templateName: template.templateName, cid2Type: template.cid2Type,
    pointId: point.id ?? 0, baseDataId: point.baseDataId, pointName: point.name, dataType: point.dataType,
}))));
const groupDialogTitle = computed(() => groupForm.value.id ? '编辑电池模板组' : '新增电池模板组');
const templateDialogTitle = computed(() => templateForm.value.id ? '编辑通道模板' : '新增通道模板');
function roleOrder(role) { return role === '42' ? 1 : role === '42_SOC' ? 2 : 3; }
function channelName(role) {
    return role === '42' ? '遥测' : role === '42_SOC' ? '遥测SOC' : '遥信';
}
function returnData(response) { return response?.data?.return; }
function requestError(error, fallback) {
    const responseMessage = String(error?.response?.data?.msg ?? '').trim();
    if (responseMessage)
        return responseMessage;
    const errorMessage = String(error?.message ?? '').trim();
    if (error?.code === 'ECONNABORTED' || /timeout/i.test(errorMessage))
        return '请求超时，请稍后重试';
    if (errorMessage === 'Network Error')
        return '网络请求失败，请检查网络连接';
    if (/[㐀-鿿]/.test(errorMessage))
        return errorMessage;
    return fallback;
}
async function loadGroups() {
    if (!auth.token)
        return;
    loading.value = true;
    try {
        const response = await fetchTtdcGroupPage({ token: auth.token, pageNum: pageNum.value, pageSize: pageSize.value, keyword: keyword.value.trim() });
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '查询失败');
        const payload = returnData(response) ?? { data: [], total: 0 };
        groups.value = Array.isArray(payload.data) ? payload.data : [];
        total.value = Number(payload.total ?? 0);
        if (!groups.value.some(item => item.id === selectedGroupId.value))
            selectedGroupId.value = groups.value[0]?.id;
        await refreshSelectedGroupData();
        await loadAlarmRules();
    }
    catch (error) {
        ElMessage.error(requestError(error, '模板组查询失败'));
    }
    finally {
        loading.value = false;
    }
}
async function loadAlarmRules() {
    if (!canViewAlarmRule.value || !auth.token) {
        alarmRules.value = [];
        alarmRuleTotal.value = 0;
        return;
    }
    alarmRuleLoading.value = true;
    try {
        const response = await fetchTtdcAlarmRulePage({ token: auth.token, pageNum: alarmRulePage.value, pageSize: alarmRulePageSize.value, keyword: alarmRuleKeyword.value.trim() });
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '告警规则查询失败');
        const payload = returnData(response) ?? { data: [], total: 0 };
        alarmRules.value = Array.isArray(payload.data) ? payload.data : [];
        alarmRuleTotal.value = Number(payload.total ?? 0);
    }
    catch (error) {
        ElMessage.error(requestError(error, '告警规则查询失败'));
    }
    finally {
        alarmRuleLoading.value = false;
    }
}
async function refreshSelectedGroupData() {
    if (!auth.token || !selectedGroupId.value)
        return;
    try {
        const response = await fetchTtdcGroupDetail(auth.token, selectedGroupId.value);
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '模板组详情加载失败');
        const detail = returnData(response)?.data;
        const index = groups.value.findIndex(item => item.id === selectedGroupId.value);
        if (detail && index >= 0)
            groups.value[index] = detail;
    }
    catch (error) {
        ElMessage.error(requestError(error, '模板组详情加载失败'));
    }
}
function selectGroup(row) {
    if (!row?.id || row.id === selectedGroupId.value)
        return;
    selectedGroupId.value = row.id;
    alarmRulePage.value = 1;
    void refreshSelectedGroupData().then(() => loadAlarmRules());
}
function openAddGroup() {
    if (!canAddGroup.value)
        return;
    groupForm.value = emptyGroup();
    groupDialogVisible.value = true;
}
async function openEditGroup(row) {
    if (!canUpdateGroup.value || !auth.token || !row.id)
        return;
    try {
        const response = await fetchTtdcGroupDetail(auth.token, row.id);
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '模板组详情查询失败');
        const detail = returnData(response)?.data;
        if (!detail)
            throw new Error('模板组详情数据为空');
        groupForm.value = { ...detail, templates: [...(detail.templates ?? [])] };
        groupDialogVisible.value = true;
    }
    catch (error) {
        ElMessage.error(requestError(error, '模板组详情查询失败'));
    }
}
async function submitGroup() {
    if (!canSaveGroup.value || !auth.token)
        return;
    if (!groupForm.value.groupCode.trim() || !groupForm.value.groupName.trim() || !groupForm.value.topicSegment.trim()) {
        ElMessage.warning('请填写模板组编码、名称和主题区分段');
        return;
    }
    saving.value = true;
    try {
        const response = await saveTtdcGroup(auth.token, groupForm.value);
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '保存失败');
        ElMessage.success('模板组保存成功');
        groupDialogVisible.value = false;
        await loadGroups();
    }
    catch (error) {
        ElMessage.error(requestError(error, '模板组保存失败'));
    }
    finally {
        saving.value = false;
    }
}
async function removeGroup(row) {
    if (!canDeleteGroup.value || !auth.token || !row.id)
        return;
    try {
        await ElMessageBox.confirm(`确定删除模板组“${row.groupName}”吗？`, '删除模板组', { type: 'warning' });
    }
    catch {
        return;
    }
    try {
        const response = await deleteTtdcGroup(auth.token, row.id);
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '模板组删除失败');
        ElMessage.success('删除成功');
        await loadGroups();
    }
    catch (error) {
        ElMessage.error(requestError(error, '模板组删除失败'));
    }
}
async function changeGroupState(row, field, value) {
    if (!canChangeGroupState.value || !auth.token || !row.id)
        return;
    let confirmed = false;
    if (field === 'runtime_match_enabled' && !value) {
        try {
            await ElMessageBox.confirm(`关闭后将停止接收 /DCBMS4G/${row.topicSegment}/PUB/<网关序列号>，确定继续吗？`, '影响在线采集', { type: 'warning' });
            confirmed = true;
        }
        catch {
            row[fieldToProperty(field)] = true;
            return;
        }
    }
    try {
        const response = await setTtdcGroupState(auth.token, row.id, field, value, confirmed);
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '模板组状态更新失败');
        await loadGroups();
    }
    catch (error) {
        ;
        row[fieldToProperty(field)] = !value;
        ElMessage.error(requestError(error, '模板组状态更新失败'));
    }
}
function fieldToProperty(field) { return field === 'enabled' ? 'enabled' : field === 'runtime_match_enabled' ? 'runtimeMatchEnabled' : 'autoCreateEnabled'; }
function firstMissingRole(group) {
    const used = new Set((group.templates ?? []).map(item => item.cid2Type));
    return ['42', '42_SOC', '44'].find(role => !used.has(role)) ?? '42';
}
function openAddTemplate(group) {
    if (!canAddTemplate.value || !group.id)
        return;
    const role = firstMissingRole(group);
    templateForm.value = emptyTemplate(group.id, role);
    pointPage.value = 1;
    templateDialogVisible.value = true;
}
async function openEditTemplate(row) {
    if (!canUpdateTemplate.value || !auth.token || !row.id)
        return;
    try {
        const response = await fetchTtdcTemplateDetail(auth.token, row.id);
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '模板详情查询失败');
        const detail = returnData(response)?.data;
        if (!detail)
            throw new Error('模板详情数据为空');
        templateForm.value = { ...detail, points: [...(detail.points ?? [])] };
        pointPage.value = 1;
        templateDialogVisible.value = true;
    }
    catch (error) {
        ElMessage.error(requestError(error, '模板详情查询失败'));
    }
}
function onRoleChange(role) {
    templateForm.value.parserProfile = role === '42_SOC' ? 'derived_algorithm' : 'raw_fields';
    templateForm.value.machineCid2 = role === '42_SOC' ? '42_SOC' : role;
    templateForm.value.points = [emptyPoint(role)];
    pointPage.value = 1;
}
function addPoint() {
    templateForm.value.points.push(emptyPoint(templateForm.value.cid2Type, templateForm.value.points.length + 1));
    pointPage.value = Math.ceil(templateForm.value.points.length / pointPageSize);
}
function removePoint(point) {
    const index = templateForm.value.points.indexOf(point);
    if (index >= 0)
        templateForm.value.points.splice(index, 1);
    const max = Math.max(1, Math.ceil(templateForm.value.points.length / pointPageSize));
    if (pointPage.value > max)
        pointPage.value = max;
}
function onSocPointIDChange(point) {
    const key = point.baseDataId.trim();
    if (/^alg_cell(?:0[1-9]|[1-9][0-9]{1,2})_soc$/.test(key) || ['alg_fangdianshichang', 'alg_soc', 'alg_soh', 'ditoo_wencha', 'ditoo_yacha'].includes(key)) {
        point.dataType = 1;
    }
    else if (key === 'alg_rekongfengxianstage') {
        point.dataType = 2;
    }
}
function onConvertTypeChange(point, value) {
    point.convertType = value;
    point.bitIndex = value === 2 ? (point.bitIndex ?? 0) : null;
    if (value === 2) {
        point.formula = 'x';
        point.unformula = 'x';
    }
}
function onByteCountChange(point) {
    const byteCount = Number(point.byteCount ?? 0);
    const bitIndex = Number(point.bitIndex ?? 0);
    if (point.convertType === 2 && byteCount > 0 && bitIndex >= byteCount * 8)
        point.bitIndex = 0;
}
function bitOptions(point) {
    const byteCount = Number(point.byteCount ?? 0);
    const count = Number.isFinite(byteCount) && byteCount > 0 ? Math.min(byteCount * 8, 256) : 0;
    return Array.from({ length: count }, (_, index) => index);
}
async function submitTemplate() {
    if (!canSaveTemplate.value || !auth.token)
        return;
    const item = templateForm.value;
    if (!item.groupId || !item.templateName.trim() || !item.cid2Type) {
        ElMessage.warning('请填写完整模板基本信息');
        return;
    }
    if (!item.points.length) {
        ElMessage.warning('数据点列表不能为空');
        return;
    }
    item.points.forEach((point, index) => {
        point.sortNo = index + 1;
        if (item.cid2Type === '42_SOC') {
            point.address = null;
            point.byteCount = null;
            point.convertType = null;
            point.bitIndex = null;
            point.transport = '-';
            point.formula = 'x';
            point.unformula = 'x';
        }
        else {
            point.algorithmKey = null;
            point.dataType = 1;
            if (item.cid2Type === '42')
                point.bitIndex = null;
        }
    });
    saving.value = true;
    try {
        // Pagination only changes rendering; the full in-memory point array is submitted.
        const response = await saveTtdcTemplate(auth.token, { ...item, points: item.points });
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '保存失败');
        ElMessage.success('模板保存成功');
        templateDialogVisible.value = false;
        await loadGroups();
    }
    catch (error) {
        ElMessage.error(requestError(error, '模板保存失败'));
    }
    finally {
        saving.value = false;
    }
}
async function removeTemplate(row) {
    if (!canDeleteTemplate.value || !auth.token || !row.id)
        return;
    try {
        await ElMessageBox.confirm(`确定删除模板“${row.templateName}”吗？`, '删除模板', { type: 'warning' });
    }
    catch {
        return;
    }
    try {
        const response = await deleteTtdcTemplate(auth.token, row.id);
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '模板删除失败');
        ElMessage.success('删除成功');
        await loadGroups();
    }
    catch (error) {
        ElMessage.error(requestError(error, '模板删除失败'));
    }
}
async function toggleTemplate(row, value) {
    if (!canChangeTemplateState.value || !auth.token || !row.id)
        return;
    try {
        const response = await setTtdcTemplateManualEnabled(auth.token, row.id, value);
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '模板状态更新失败');
    }
    catch (error) {
        row.enabled = !value;
        ElMessage.error(requestError(error, '模板状态更新失败'));
    }
}
async function uploadTemplate(options) {
    if (!canImportTemplate.value || !auth.token || !selectedGroup.value?.id)
        return;
    try {
        const response = await importTtdcTemplate(auth.token, options.file, selectedGroup.value.id);
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '导入失败');
        ElMessage.success('导入成功');
        await loadGroups();
        options.onSuccess(response.data);
    }
    catch (error) {
        ElMessage.error(requestError(error, '导入失败'));
        options.onError(error);
    }
}
async function downloadTemplate(row) {
    if (!canExportTemplate.value || !auth.token || !row.id)
        return;
    try {
        const response = await exportTtdcTemplate(auth.token, row.id);
        if (String(response.headers['content-type'] ?? '').includes('application/json')) {
            const payload = JSON.parse(await response.data.text());
            throw new Error(payload?.msg || '模板导出失败');
        }
        const url = URL.createObjectURL(response.data);
        const anchor = document.createElement('a');
        anchor.href = url;
        anchor.download = `${row.templateCode}.xlsx`;
        anchor.click();
        URL.revokeObjectURL(url);
    }
    catch (error) {
        ElMessage.error(requestError(error, '模板导出失败'));
    }
}
function alarmRoleName(role) { return channelName(role); }
function alarmPointLabel(option) {
    return `${option.pointName || '-'} / ${alarmRoleName(option.cid2Type)}`;
}
function setAlarmPoint(pointId) {
    const option = alarmPointOptions.value.find(item => item.pointId === pointId);
    if (!option)
        return;
    alarmRuleForm.value.templateId = option.templateId;
    alarmRuleForm.value.templatePointId = option.pointId;
    alarmRuleForm.value.baseDataId = option.baseDataId;
}
function alarmPointValue() {
    return alarmRuleForm.value.templatePointId || undefined;
}
function alarmTypeName(value) { return value === '1' ? '预警' : '告警'; }
function alarmTriggerModelName() { return '数字'; }
function alarmLogicText(logic) { return `${logic.logic} ${logic.triggerValue}`; }
function openAddAlarmRule() {
    if (!canAddAlarmRule.value || !selectedGroup.value?.id)
        return;
    const first = alarmPointOptions.value[0];
    if (!first) {
        ElMessage.warning('请先在当前模板组中配置启用的数据点');
        return;
    }
    alarmRuleForm.value = emptyAlarmRule(selectedGroup.value.id);
    setAlarmPoint(first.pointId);
    alarmRuleDialogVisible.value = true;
}
async function openEditAlarmRule(row) {
    if (!canUpdateAlarmRule.value || !auth.token || !row.id)
        return;
    try {
        const response = await fetchTtdcAlarmRuleDetail(auth.token, row.id);
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '告警规则详情查询失败');
        const detail = returnData(response)?.data;
        if (!detail)
            throw new Error('告警规则详情数据为空');
        alarmRuleForm.value = { ...detail, cycle: 0, logics: [...(detail.logics ?? [])] };
        alarmRuleDialogVisible.value = true;
    }
    catch (error) {
        ElMessage.error(requestError(error, '告警规则详情查询失败'));
    }
}
function addAlarmLogic() {
    alarmRuleForm.value.logics.push(emptyAlarmLogic());
}
function removeAlarmLogic(index) {
    if (alarmRuleForm.value.logics.length <= 1)
        return;
    alarmRuleForm.value.logics.splice(index, 1);
}
async function submitAlarmRule() {
    if (!canSaveAlarmRule.value || !auth.token)
        return;
    const item = alarmRuleForm.value;
    if (!item.templateGroupId || !item.templateId || !item.templatePointId || !item.ruleName.trim() || !item.ruleCode.trim()) {
        ElMessage.warning('请填写规则名称、告警代码并选择通道数据点');
        return;
    }
    if (![1, 2, 3].includes(Number(item.alarmStage))) {
        ElMessage.warning('告警等级只能选择一级、二级或三级');
        return;
    }
    if (!item.logics.length || item.logics.some(logic => !logic.logic || !Number.isFinite(Number(logic.triggerValue)))) {
        ElMessage.warning('请填写完整的判断条件');
        return;
    }
    alarmRuleSaving.value = true;
    try {
        const response = await saveTtdcAlarmRule(auth.token, {
            id: item.id,
            templateGroupId: item.templateGroupId,
            templateId: item.templateId,
            templatePointId: item.templatePointId,
            baseDataId: item.baseDataId,
            ruleName: item.ruleName,
            ruleCode: item.ruleCode,
            alarmType: item.alarmType,
            alarmStage: item.alarmStage,
            formula: item.formula,
            // TTDC public alarm rules always use the single-trigger mode.
            cycle: 0,
            sustainTime: item.sustainTime,
            enabled: item.enabled,
            version: item.version,
            description: item.description,
            logics: item.logics.map(logic => ({ logic: logic.logic, triggerValue: Number(logic.triggerValue) })),
        });
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '告警规则保存失败');
        ElMessage.success('告警规则保存成功');
        alarmRuleDialogVisible.value = false;
        await loadAlarmRules();
    }
    catch (error) {
        ElMessage.error(requestError(error, '告警规则保存失败'));
    }
    finally {
        alarmRuleSaving.value = false;
    }
}
async function toggleAlarmRule(row, value) {
    if (!canChangeAlarmRuleState.value || !auth.token || !row.id)
        return;
    try {
        const response = await setTtdcAlarmRuleEnabled(auth.token, row.id, value);
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '告警规则状态更新失败');
    }
    catch (error) {
        row.enabled = !value;
        ElMessage.error(requestError(error, '告警规则状态更新失败'));
    }
}
async function removeAlarmRule(row) {
    if (!canDeleteAlarmRule.value || !auth.token || !row.id)
        return;
    try {
        await ElMessageBox.confirm(`确定删除规则“${row.ruleName}”吗？历史告警记录不会删除。`, '删除告警规则', { type: 'warning' });
    }
    catch {
        return;
    }
    try {
        const response = await deleteTtdcAlarmRule(auth.token, row.id);
        if (response.data?.code !== 200)
            throw new Error(response.data?.msg || '告警规则删除失败');
        ElMessage.success('告警规则删除成功');
        await loadAlarmRules();
    }
    catch (error) {
        ElMessage.error(requestError(error, '告警规则删除失败'));
    }
}
onMounted(() => { void loadGroups(); });
</script>

<template>
  <main class="template-page">
    <header class="page-toolbar">
      <h1>TTDC电池通道模板</h1>
      <div class="toolbar-actions">
        <el-input v-model="keyword" clearable placeholder="组名称、编码或主题段" @keyup.enter="loadGroups" />
        <el-button @click="loadGroups">查询</el-button>
        <el-button v-if="canAddGroup" type="primary" :icon="Plus" @click="openAddGroup">新增模板组</el-button>
      </div>
    </header>

    <section class="group-section">
      <el-table v-loading="loading" :data="groups" row-key="id" highlight-current-row @current-change="selectGroup">
        <el-table-column label="模板组" min-width="180">
          <template #default="{ row }"><strong>{{ row.groupName }}</strong><div class="machine-code">{{ row.groupCode }}</div></template>
        </el-table-column>
        <el-table-column label="主题区分段" min-width="180"><template #default="{ row }"><code>{{ row.topicSegment }}</code></template></el-table-column>
        <el-table-column label="完整状态" width="110" align="center"><template #default="{ row }"><el-tag :type="row.complete ? 'success' : 'warning'">{{ row.complete ? '三通道完整' : '待补齐' }}</el-tag></template></el-table-column>
        <el-table-column label="手动可选" width="100" align="center"><template #default="{ row }"><el-switch v-model="row.enabled" :disabled="!canChangeGroupState" @change="changeGroupState(row, 'enabled', Boolean($event))" /></template></el-table-column>
        <el-table-column label="运行时接收" width="110" align="center"><template #default="{ row }"><el-switch v-model="row.runtimeMatchEnabled" :disabled="!canChangeGroupState" @change="changeGroupState(row, 'runtime_match_enabled', Boolean($event))" /></template></el-table-column>
        <el-table-column label="自动创建" width="100" align="center"><template #default="{ row }"><el-switch v-model="row.autoCreateEnabled" :disabled="!canChangeGroupState" @change="changeGroupState(row, 'auto_create_enabled', Boolean($event))" /></template></el-table-column>
        <el-table-column v-if="hasGroupOperations" label="操作" width="126" fixed="right" align="center">
          <template #default="{ row }"><el-button v-if="canUpdateGroup" link :icon="Edit" title="编辑模板组" @click.stop="openEditGroup(row)" /><el-button v-if="canDeleteGroup" link type="danger" :icon="Delete" title="删除模板组" @click.stop="removeGroup(row)" /></template>
        </el-table-column>
      </el-table>
      <el-pagination v-model:current-page="pageNum" v-model:page-size="pageSize" :total="total" layout="total, sizes, prev, pager, next" @change="loadGroups" />
    </section>

    <section v-if="selectedGroup" class="template-section">
      <div class="section-head"><div><h2>{{ selectedGroup.groupName }} / 通道模板</h2><span class="topic-preview">/DCBMS4G/{{ selectedGroup.topicSegment }}/PUB/&lt;网关序列号&gt;</span></div><div class="section-actions"><el-upload v-if="canImportTemplate" :show-file-list="false" accept=".xlsx" :http-request="uploadTemplate"><el-button :icon="Upload">导入</el-button></el-upload><el-button v-if="canAddTemplate" type="primary" :icon="Plus" @click="openAddTemplate(selectedGroup)">新增模板</el-button></div></div>
      <el-table :data="sortedTemplates" row-key="id" empty-text="请分别创建遥测、遥测SOC和遥信通道模板">
        <el-table-column label="通道" width="100"><template #default="{ row }"><el-tag effect="plain">{{ channelName(row.cid2Type) }}</el-tag></template></el-table-column>
        <el-table-column label="模板名称" min-width="220"><template #default="{ row }"><strong>{{ row.templateName }}</strong></template></el-table-column>
        <el-table-column label="数据点" width="90" align="right"><template #default="{ row }">{{ row.pointCount ?? 0 }}</template></el-table-column>
        <el-table-column label="手动可选" width="100" align="center"><template #default="{ row }"><el-switch v-model="row.enabled" :disabled="!canChangeTemplateState" @change="toggleTemplate(row, Boolean($event))" /></template></el-table-column>
        <el-table-column v-if="hasTemplateOperations" label="操作" width="156" fixed="right" align="center"><template #default="{ row }"><el-button v-if="canUpdateTemplate" link :icon="Edit" title="编辑" @click="openEditTemplate(row)" /><el-button v-if="canExportTemplate" link :icon="Download" title="导出" @click="downloadTemplate(row)" /><el-button v-if="canDeleteTemplate" link type="danger" :icon="Delete" title="删除" @click="removeTemplate(row)" /></template></el-table-column>
      </el-table>
    </section>

    <section v-if="canViewAlarmRule" class="alarm-section">
      <div class="section-head">
        <div>
          <h2>公共告警和预警配置</h2>
          <span class="topic-preview">规则按数据池标签名自动关联所有TTDC电池设备</span>
        </div>
        <div class="section-actions">
          <el-input v-model="alarmRuleKeyword" clearable placeholder="规则名称、代码或数据点" @keyup.enter="loadAlarmRules" />
          <el-button @click="loadAlarmRules">查询</el-button>
          <el-button v-if="canAddAlarmRule" type="primary" :icon="Plus" @click="openAddAlarmRule">新增规则</el-button>
        </div>
      </div>
      <el-table v-loading="alarmRuleLoading" :data="alarmRules" row-key="id" empty-text="暂无公共告警或预警规则">
        <el-table-column label="规则名称" min-width="180">
          <template #default="{ row }"><strong>{{ row.ruleName }}</strong><div class="machine-code">{{ row.ruleCode }}</div></template>
        </el-table-column>
        <el-table-column label="数据池标签名" min-width="250">
          <template #default="{ row }"><span>{{ row.pointName || '-' }}</span></template>
        </el-table-column>
        <el-table-column label="类型" width="86" align="center"><template #default="{ row }"><el-tag :type="row.alarmType === '1' ? 'warning' : 'danger'">{{ alarmTypeName(row.alarmType) }}</el-tag></template></el-table-column>
        <el-table-column label="判断类型" width="96" align="center"><template #default>{{ alarmTriggerModelName() }}</template></el-table-column>
        <el-table-column label="判断条件" min-width="190"><template #default="{ row }">{{ row.logics?.map(alarmLogicText).join('；') || '-' }}</template></el-table-column>
        <el-table-column label="启用" width="82" align="center"><template #default="{ row }"><el-switch v-model="row.enabled" :disabled="!canChangeAlarmRuleState" @change="toggleAlarmRule(row, Boolean($event))" /></template></el-table-column>
        <el-table-column v-if="hasAlarmRuleOperations" label="操作" width="116" fixed="right" align="center">
          <template #default="{ row }"><el-button v-if="canUpdateAlarmRule" link :icon="Edit" title="编辑告警规则" @click="openEditAlarmRule(row)" /><el-button v-if="canDeleteAlarmRule" link type="danger" :icon="Delete" title="删除告警规则" @click="removeAlarmRule(row)" /></template>
        </el-table-column>
      </el-table>
      <el-pagination v-model:current-page="alarmRulePage" v-model:page-size="alarmRulePageSize" :total="alarmRuleTotal" layout="total, sizes, prev, pager, next" @change="loadAlarmRules" />
    </section>

    <el-dialog v-model="groupDialogVisible" :title="groupDialogTitle" width="min(680px, 94vw)" destroy-on-close>
      <el-form label-width="110px">
        <el-form-item label="模板组编码" required><el-input v-model="groupForm.groupCode" :disabled="Boolean(groupForm.id)" placeholder="例如 SODIUM_BATTERY" /></el-form-item>
        <el-form-item label="模板组名称" required><el-input v-model="groupForm.groupName" /></el-form-item>
        <el-form-item label="主题区分段" required><el-input v-model="groupForm.topicSegment" placeholder="例如 SodiumBattery" /></el-form-item>
        <el-form-item label="手动可选"><el-switch v-model="groupForm.enabled" /></el-form-item>
        <el-form-item label="说明"><el-input v-model="groupForm.description" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="groupDialogVisible = false">取消</el-button><el-button v-if="canSaveGroup" type="primary" :loading="saving" @click="submitGroup">保存</el-button></template>
    </el-dialog>

    <el-dialog v-model="templateDialogVisible" :title="templateDialogTitle" width="min(1680px, 96vw)" top="4vh" destroy-on-close>
      <el-form label-width="104px" class="template-form">
        <div class="form-grid">
          <el-form-item label="模板名称" required><el-input v-model="templateForm.templateName" /></el-form-item>
          <el-form-item label="通道" required><el-select v-model="templateForm.cid2Type" :disabled="Boolean(templateForm.id)" @change="onRoleChange"><el-option label="遥测" value="42" /><el-option label="遥测SOC" value="42_SOC" /><el-option label="遥信" value="44" /></el-select></el-form-item>
          <el-form-item label="手动可选"><el-switch v-model="templateForm.enabled" /></el-form-item>
          <el-form-item label="说明"><el-input v-model="templateForm.description" /></el-form-item>
        </div>
      </el-form>
      <div class="point-head">
        <h2>数据点列表 <span>{{ pointTotal }}</span></h2>
        <div class="point-head-actions">
          <el-button v-if="canSaveTemplate" size="small" type="primary" @click="addPoint">添加数据点</el-button>
        </div>
      </div>
      <el-table :data="pagedPoints" max-height="520" border style="width: 100%" :class="{ 'soc42-point-table': templateForm.cid2Type === '42_SOC' }">
        <template v-if="templateForm.cid2Type === '42_SOC'">
          <el-table-column label="ID" min-width="220" align="center">
            <template #default="{ row }"><el-input v-model="row.baseDataId" placeholder="请输入算法结果ID" clearable @input="onSocPointIDChange(row)" /></template>
          </el-table-column>
          <el-table-column label="名称" min-width="180">
            <template #default="{ row }"><el-input v-model="row.name" /></template>
          </el-table-column>
          <el-table-column label="数据类型" width="228" align="center">
            <template #default="{ row }">
              <el-select v-model="row.dataType" disabled style="width: 100%"><el-option label="浮点型" :value="1" /><el-option label="字符串" :value="2" /></el-select>
            </template>
          </el-table-column>
        </template>
        <template v-else>
          <el-table-column label="名称" min-width="160">
            <template #default="{ row }"><el-input v-model="row.name" /></template>
          </el-table-column>
          <el-table-column label="地址" min-width="105">
            <template #default="{ row }"><el-input-number v-model="row.address" :min="0" :step="2" :controls="false" style="width: 100%" /></template>
          </el-table-column>
          <el-table-column label="字节数" min-width="105">
            <template #default="{ row }"><el-input-number v-model="row.byteCount" :min="1" :controls="false" style="width: 100%" @change="onByteCountChange(row)" /></template>
          </el-table-column>
          <el-table-column label="进制转换类型" min-width="220">
            <template #default="{ row }">
              <el-select v-model="row.convertType" placeholder="请选择" style="width: 100%" @change="onConvertTypeChange(row, Number($event))">
                <el-option label="十六进制转有符号十进制" :value="1" />
                <el-option v-if="templateForm.cid2Type === '44'" label="十六进制转二进制" :value="2" />
                <el-option label="十六进制转无符号十进制" :value="3" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column v-if="templateForm.cid2Type === '44'" label="取位" min-width="120">
            <template #default="{ row }">
              <el-select v-if="row.convertType === 2" v-model="row.bitIndex" placeholder="请选择" style="width: 100%">
                <el-option v-for="bit in bitOptions(row)" :key="bit" :value="bit" :label="`取位${bit + 1}`" />
              </el-select>
              <el-input v-else model-value="不取位" disabled />
            </template>
          </el-table-column>
          <el-table-column label="数据类型" min-width="120">
            <template #default="{ row }"><el-select v-model="row.dataType" disabled style="width: 100%"><el-option label="浮点型" :value="1" /></el-select></template>
          </el-table-column>
          <el-table-column label="formula" min-width="120">
            <template #default="{ row }"><el-input v-model="row.formula" :disabled="row.convertType === 2" /></template>
          </el-table-column>
          <el-table-column label="unformula" min-width="120">
            <template #default="{ row }"><el-input v-model="row.unformula" :disabled="row.convertType === 2" /></template>
          </el-table-column>
        </template>
        <el-table-column v-if="canSaveTemplate" label="操作" width="76" fixed="right" align="center">
          <template #default="{ row }"><el-button size="small" type="danger" @click.stop="removePoint(row)">删</el-button></template>
        </el-table-column>
      </el-table>
      <el-pagination v-if="pointTotal > pointPageSize" v-model:current-page="pointPage" :page-size="pointPageSize" :total="pointTotal" :pager-count="5" layout="total, prev, pager, next" background />
      <template #footer><el-button @click="templateDialogVisible = false">取消</el-button><el-button v-if="canSaveTemplate" type="primary" :loading="saving" @click="submitTemplate">保存全部数据点</el-button></template>
    </el-dialog>

    <el-dialog v-model="alarmRuleDialogVisible" :title="alarmRuleForm.id ? '编辑告警规则' : '新增告警规则'" width="min(920px, 94vw)" destroy-on-close>
      <el-form label-width="106px" class="alarm-rule-form">
        <div class="form-grid">
          <el-form-item label="规则名称" required><el-input v-model="alarmRuleForm.ruleName" placeholder="例如 单体电压过低" /></el-form-item>
          <el-form-item label="告警代码" required><el-input v-model="alarmRuleForm.ruleCode" placeholder="例如 CELL_VOLTAGE_LOW" /></el-form-item>
          <el-form-item label="数据池标签名" required>
            <el-select :model-value="alarmPointValue()" filterable style="width: 100%" placeholder="请选择标签参考数据点" @change="setAlarmPoint(Number($event))">
              <el-option v-for="option in alarmPointOptions" :key="option.pointId" :label="alarmPointLabel(option)" :value="option.pointId" />
            </el-select>
          </el-form-item>
          <el-form-item label="告警类型"><el-select v-model="alarmRuleForm.alarmType" style="width: 100%"><el-option label="告警" value="0" /><el-option label="预警" value="1" /></el-select></el-form-item>
          <el-form-item label="判断类型"><el-input model-value="数字" disabled /></el-form-item>
          <el-form-item label="告警等级"><el-select v-model="alarmRuleForm.alarmStage" style="width: 100%"><el-option v-for="stage in alarmStageOptions" :key="stage.value" :label="stage.label" :value="stage.value" /></el-select></el-form-item>
          <el-form-item label="触发方式"><el-input model-value="单次触发" disabled /></el-form-item>
          <el-form-item label="持续时间"><el-input-number v-model="alarmRuleForm.sustainTime" :min="0" :controls="false" style="width: 100%" /></el-form-item>
          <el-form-item label="启用"><el-switch v-model="alarmRuleForm.enabled" /></el-form-item>
        </div>
        <el-form-item label="处理公式"><el-input v-model="alarmRuleForm.formula" placeholder="默认 x" /></el-form-item>
        <el-form-item label="说明"><el-input v-model="alarmRuleForm.description" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <div class="logic-head"><h2>判断条件 <span>{{ alarmRuleForm.logics.length }}</span></h2><el-button size="small" :icon="Plus" @click="addAlarmLogic">添加条件</el-button></div>
      <el-table :data="alarmRuleForm.logics" border class="logic-table">
        <el-table-column label="条件" width="74" align="center"><template #default="{ $index }">{{ $index + 1 }}</template></el-table-column>
        <el-table-column label="运算符" width="150">
          <template #default="{ row }">
            <el-select v-model="row.logic" style="width: 100%"><el-option v-for="operator in ['>', '>=', '<', '<=', '=']" :key="operator" :label="operator" :value="operator" /></el-select>
          </template>
        </el-table-column>
        <el-table-column label="阈值" min-width="260">
          <template #default="{ row }"><el-input-number v-model="row.triggerValue" :controls="false" style="width: 100%" /></template>
        </el-table-column>
        <el-table-column label="操作" width="78" align="center"><template #default="{ $index }"><el-button v-if="alarmRuleForm.logics.length > 1" link type="danger" :icon="Delete" title="删除条件" @click="removeAlarmLogic($index)" /></template></el-table-column>
      </el-table>
      <template #footer><el-button @click="alarmRuleDialogVisible = false">取消</el-button><el-button v-if="canSaveAlarmRule" type="primary" :loading="alarmRuleSaving" @click="submitAlarmRule">保存规则</el-button></template>
    </el-dialog>
  </main>
</template>

<style scoped>
.template-page { min-width: 0; padding: 18px 22px 30px; color: var(--el-text-color-primary); }
.page-toolbar, .section-head, .point-head { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.point-head-actions { display: flex; align-items: center; gap: 8px; }
.page-toolbar { margin-bottom: 16px; }
h1 { margin: 0; font-size: 22px; font-weight: 650; letter-spacing: 0; }
h2 { margin: 0; font-size: 16px; letter-spacing: 0; }
.toolbar-actions, .section-actions { display: flex; align-items: center; gap: 8px; }
.toolbar-actions .el-input { width: 260px; }
.group-section, .template-section, .alarm-section { border-top: 1px solid var(--el-border-color-light); padding-top: 12px; }
.template-section { margin-top: 24px; }
.alarm-section { margin-top: 24px; }
.section-head { margin-bottom: 10px; }
.topic-preview, .machine-code { color: var(--el-text-color-secondary); font-size: 12px; }
.topic-preview { display: block; margin-top: 3px; font-family: ui-monospace, SFMono-Regular, Consolas, monospace; }
.machine-code { margin-top: 2px; }
code { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; }
.el-pagination { justify-content: flex-end; margin-top: 12px; }
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 18px; }
.point-head { margin: 8px 0 10px; }
.point-head h2 span { margin-left: 6px; color: var(--el-text-color-secondary); font-weight: 400; }
.alarm-section .section-actions .el-input { width: 220px; }
.logic-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin: 6px 0 10px 106px; }
.logic-head h2 span { margin-left: 6px; color: var(--el-text-color-secondary); font-weight: 400; }
.logic-table { margin-left: 106px; width: calc(100% - 106px); }
:deep(.el-input-number) { width: 100%; }
@media (max-width: 820px) {
  .template-page { padding: 12px; }
  .page-toolbar, .section-head { align-items: flex-start; flex-direction: column; }
  .toolbar-actions { width: 100%; flex-wrap: wrap; }
  .toolbar-actions .el-input { width: min(100%, 320px); }
  .form-grid { grid-template-columns: 1fr; }
  .alarm-section .section-actions { width: 100%; flex-wrap: wrap; }
  .alarm-section .section-actions .el-input { width: min(100%, 320px); }
  .logic-head, .logic-table { margin-left: 0; width: 100%; }
}
</style>
