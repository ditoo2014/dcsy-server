import axios from 'axios';
import { createAuthHeaders } from '../utils/request';
import { attachAuthFailureRedirectInterceptor } from './authFailureRedirect';
const streamHttp = axios.create({
    baseURL: import.meta.env.VITE_AUTH_BASE_URL,
    timeout: Number(import.meta.env.VITE_API_TIMEOUT ?? 10000),
});
attachAuthFailureRedirectInterceptor(streamHttp);
export function fetchStreamPage(payload) {
    const { token, ...body } = payload;
    return streamHttp.post('/tolink/stream/findpage', body, {
        headers: createAuthHeaders(token),
    });
}
export function createStream(payload) {
    const { token, ...body } = payload;
    return streamHttp.post('/tolink/stream/create', body, {
        headers: createAuthHeaders(token),
    });
}
export function deleteStream(payload) {
    const { token, apiKey } = payload;
    return streamHttp.post(`/tolink/stream/delete?ApiKey=${encodeURIComponent(apiKey)}`, {}, {
        headers: createAuthHeaders(token),
    });
}
export function updateStream(payload) {
    const { token, apiKey, bodyApiKey, ...rest } = payload;
    return streamHttp.post(`/tolink/stream/update?ApiKey=${encodeURIComponent(apiKey)}`, {
        ...rest,
        apiKey: bodyApiKey ?? apiKey,
    }, {
        headers: createAuthHeaders(token),
    });
}
export function fetchStreamLabelsIntact(payload) {
    const { token, streamName } = payload;
    return streamHttp.post(`/tolink/stream/getlabel?Name=${encodeURIComponent(streamName)}&Function=intact`, {}, {
        headers: createAuthHeaders(token),
    });
}
export function createStreamLabels(payload) {
    const { token, streamName, apiKey, body, functionName = 'create' } = payload;
    const normalizedBody = Array.isArray(body)
        ? body.map(item => {
            const rawType = item?.dataType;
            const normalizedType = typeof rawType === 'string' && rawType.trim()
                ? rawType.trim()
                : (Number.isFinite(Number(rawType)) ? String(Math.trunc(Number(rawType))) : '1');
            return {
                ...item,
                dataType: normalizedType,
            };
        })
        : [];
    return streamHttp.post(`/tolink/stream/label?Name=${encodeURIComponent(streamName)}&ApiKey=${encodeURIComponent(apiKey)}&Function=${encodeURIComponent(functionName)}`, normalizedBody, {
        headers: createAuthHeaders(token),
    });
}
export function downloadStreamBatchImportTemplate(token) {
    return streamHttp.post('/tolink/stream/batchimport/template', {}, {
        headers: createAuthHeaders(token),
        responseType: 'blob',
        timeout: 30000,
    });
}
export function batchImportStreams(token, file, dryRun = true) {
    const form = new FormData();
    form.append('file', file);
    return streamHttp.post(`/tolink/stream/batchimport?dryRun=${dryRun ? 'true' : 'false'}`, form, {
        headers: { ...createAuthHeaders(token), 'Content-Type': 'multipart/form-data' },
        responseType: dryRun ? 'blob' : 'json',
        timeout: 120000,
    });
}
