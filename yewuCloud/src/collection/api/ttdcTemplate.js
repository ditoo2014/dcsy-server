import axios from 'axios';
import { createAuthHeaders } from '../utils/request';
import { attachAuthFailureRedirectInterceptor } from './authFailureRedirect';
const http = axios.create({ baseURL: import.meta.env.VITE_AUTH_BASE_URL, timeout: Number(import.meta.env.VITE_API_TIMEOUT ?? 15000) });
attachAuthFailureRedirectInterceptor(http);
const headers = (token) => ({ headers: createAuthHeaders(token) });
export function fetchTtdcGroupPage(params) {
    const { token, ...body } = params;
    return http.post('/tolink/ttdcgroup/findpage', body, headers(token));
}
export function fetchTtdcGroupDetail(token, id) {
    return http.post(`/tolink/ttdcgroup/detail?id=${id}`, {}, headers(token));
}
export function fetchTtdcGroupOptions(token) {
    return http.post('/tolink/ttdcgroup/options', {}, headers(token));
}
export function saveTtdcGroup(token, group) {
    return http.post(group.id ? '/tolink/ttdcgroup/update' : '/tolink/ttdcgroup/add', group, headers(token));
}
export function setTtdcGroupState(token, id, field, value, confirm = false) {
    return http.post('/tolink/ttdcgroup/state', { id, field, value, confirm }, headers(token));
}
export function deleteTtdcGroup(token, id) {
    return http.post(`/tolink/ttdcgroup/delete?id=${id}`, {}, headers(token));
}
export function fetchTtdcTemplatePage(params) {
    const { token, ...body } = params;
    return http.post('/tolink/ttdctemplate/findpage', body, headers(token));
}
export function fetchTtdcTemplateDetail(token, id) {
    return http.post(`/tolink/ttdctemplate/detail?id=${id}`, {}, headers(token));
}
export function fetchTtdcTemplateOptions(token, params) {
    const query = new URLSearchParams();
    if (typeof params === 'number')
        query.set('groupId', String(params));
    else if (params) {
        if (params.groupId)
            query.set('groupId', String(params.groupId));
        if (params.gatewayId)
            query.set('gatewayId', params.gatewayId);
        if (params.moduleAdr)
            query.set('moduleAdr', params.moduleAdr);
    }
    const suffix = query.size ? `?${query.toString()}` : '';
    return http.post(`/tolink/ttdctemplate/options${suffix}`, {}, headers(token));
}
export function fetchTtdcTemplateProfiles(token) {
    return http.post('/tolink/ttdctemplate/profiles', {}, headers(token));
}
export function fetchTtdcAlgorithmOptions(token) {
    return http.post('/tolink/ttdctemplate/algorithms', {}, headers(token));
}
export function saveTtdcTemplate(token, template) {
    return http.post(template.id ? '/tolink/ttdctemplate/update' : '/tolink/ttdctemplate/add', { ...template, topics: [] }, headers(token));
}
export function deleteTtdcTemplate(token, id) {
    return http.post(`/tolink/ttdctemplate/delete?id=${id}`, {}, headers(token));
}
export function setTtdcTemplateManualEnabled(token, id, value) {
    return http.post('/tolink/ttdctemplate/enable', { id, value }, headers(token));
}
export function importTtdcTemplate(token, file, groupId) {
    const form = new FormData();
    form.append('file', file);
    if (groupId)
        form.append('groupId', String(groupId));
    return http.post('/tolink/ttdctemplate/import', form, { headers: { ...createAuthHeaders(token), 'Content-Type': 'multipart/form-data' } });
}
export function exportTtdcTemplate(token, id) {
    return http.post(`/tolink/ttdctemplate/export?id=${id}`, {}, { ...headers(token), responseType: 'blob' });
}
export function fetchTtdcAlarmRulePage(params) {
    const { token, ...body } = params;
    return http.post('/tolink/ttdcalarmrule/findpage', body, headers(token));
}
export function fetchTtdcAlarmRuleDetail(token, id) {
    return http.post(`/tolink/ttdcalarmrule/detail?id=${id}`, {}, headers(token));
}
export function saveTtdcAlarmRule(token, rule) {
    return http.post(rule.id ? '/tolink/ttdcalarmrule/update' : '/tolink/ttdcalarmrule/add', rule, headers(token));
}
export function setTtdcAlarmRuleEnabled(token, id, value) {
    return http.post('/tolink/ttdcalarmrule/state', { id, value }, headers(token));
}
export function deleteTtdcAlarmRule(token, id) {
    return http.post(`/tolink/ttdcalarmrule/delete?id=${id}`, {}, headers(token));
}
