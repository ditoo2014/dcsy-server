import { batchAddDataDetails, deleteDataDetails, fetchDataDetails } from '../api/data-details';
import { fetchStreamLabelsIntact } from '../api/stream';
function getList(root) {
    const payload = root?.return ?? root ?? {};
    const list = payload?.list ?? payload?.data ?? payload?.rows ?? payload?.items ?? [];
    return Array.isArray(list) ? list : [];
}
function pickText(...values) {
    for (const value of values) {
        const text = String(value ?? '').trim();
        if (text)
            return text;
    }
    return '';
}
function buildLabelMergeKey(item) {
    return [
        pickText(item?.dataId, item?.id),
        pickText(item?.gatewayId),
        pickText(item?.dataName, item?.name, item?.tagName),
    ].join('::');
}
function mergeLabels(primary, supplement) {
    const merged = primary.map(item => ({ ...item }));
    const indexMap = new Map();
    merged.forEach((item, index) => {
        indexMap.set(buildLabelMergeKey(item), index);
    });
    for (const item of supplement) {
        const key = buildLabelMergeKey(item);
        const index = indexMap.get(key);
        if (index == null) {
            merged.push({ ...item });
            indexMap.set(key, merged.length - 1);
            continue;
        }
        merged[index] = {
            ...item,
            ...merged[index],
            labelId: pickText(merged[index]?.labelId, item?.labelId, item?.id),
            id: pickText(merged[index]?.id, item?.labelId, item?.id),
        };
    }
    return merged;
}
function normalizeLabel(item, index) {
    const labelId = pickText(item?.labelId, item?.id);
    const labelName = pickText(item?.name, item?.tagName, item?.labelName, item?.dataName);
    const dataName = pickText(item?.dataName, item?.name, item?.tagName, item?.labelName);
    if (!labelId || !labelName || !dataName)
        return null;
    return {
        equipmentId: 0,
        name: labelName,
        labelId,
        labelName,
        contentDataDetails: {},
        sortNum: index + 1,
    };
}
export async function syncEquipmentDataDetailsFromStreamLabels(payload) {
    const equipmentId = Number(payload.equipmentId);
    if (!Number.isFinite(equipmentId) || equipmentId <= 0) {
        return { ok: false, added: 0, deleted: 0, msg: '设备ID为空，业务数据点未同步' };
    }
    let labels = Array.isArray(payload.labels) ? payload.labels : [];
    const needRemoteLabels = !labels.length || labels.some(item => !pickText(item?.labelId, item?.id));
    if (needRemoteLabels) {
        try {
            const { data } = await fetchStreamLabelsIntact({
                token: payload.token,
                streamName: payload.streamName,
            });
            if (Number(data?.code) !== 200) {
                return { ok: false, added: 0, deleted: 0, msg: data?.msg || '读取标签列表失败，业务数据点未同步' };
            }
            labels = labels.length ? mergeLabels(labels, getList(data)) : getList(data);
        }
        catch (error) {
            return { ok: false, added: 0, deleted: 0, msg: error?.response?.data?.msg || '读取标签列表失败，业务数据点未同步' };
        }
    }
    const normalized = labels
        .map(normalizeLabel)
        .filter((item) => !!item)
        .map(item => ({ ...item, equipmentId }));
    if (!normalized.length) {
        return { ok: true, added: 0, deleted: 0, msg: '没有可同步的业务数据点' };
    }
    const { data: detailResp } = await fetchDataDetails({
        token: payload.token,
        equipmentId,
    });
    if (Number(detailResp?.code) !== 200) {
        return { ok: false, added: 0, deleted: 0, msg: detailResp?.msg || '读取业务数据点失败' };
    }
    const existed = getList(detailResp);
    let deleted = 0;
    for (const item of existed) {
        const id = Number(item?.id);
        if (!Number.isFinite(id) || id <= 0)
            continue;
        const { data: deleteResp } = await deleteDataDetails({
            token: payload.token,
            id,
        });
        if (Number(deleteResp?.code) !== 200) {
            return { ok: false, added: 0, deleted, msg: deleteResp?.msg || '清理旧业务数据点失败' };
        }
        deleted += 1;
    }
    const { data: addResp } = await batchAddDataDetails({
        token: payload.token,
        body: normalized,
    });
    if (Number(addResp?.code) !== 200) {
        return { ok: false, added: 0, deleted, msg: addResp?.msg || '业务数据点同步失败' };
    }
    return {
        ok: true,
        added: normalized.length,
        deleted,
        msg: deleted > 0
            ? `已清理 ${deleted} 条旧数据，并同步 ${normalized.length} 个业务数据点`
            : `已同步 ${normalized.length} 个业务数据点`,
    };
}
