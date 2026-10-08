import axios from 'axios';
import { createAuthHeaders } from '../utils/request';
import { attachAuthFailureRedirectInterceptor } from './authFailureRedirect';
const equipmentHttp = axios.create({
    baseURL: import.meta.env.VITE_AUTH_BASE_URL,
    timeout: Number(import.meta.env.VITE_API_TIMEOUT ?? 10000),
});
attachAuthFailureRedirectInterceptor(equipmentHttp);
export function findEquipmentPage(payload) {
    const { token, pageNum, pageSize, filterFunc = 'BinC' } = payload;
    return equipmentHttp.post(`/business/equipment/find?pageNum=${pageNum}&pageSize=${pageSize}&filterFunc=${encodeURIComponent(filterFunc)}`, {}, {
        headers: createAuthHeaders(token),
    });
}
export function updateEquipment(payload) {
    const { token, body } = payload;
    return equipmentHttp.post('/business/equipment/update', body, {
        headers: createAuthHeaders(token),
    });
}
