import axios from 'axios';
import { createAuthHeaders } from '../utils/request';
import { attachAuthFailureRedirectInterceptor } from './authFailureRedirect';
const dataDetailsHttp = axios.create({
    baseURL: import.meta.env.VITE_AUTH_BASE_URL,
    timeout: Number(import.meta.env.VITE_API_TIMEOUT ?? 10000),
});
attachAuthFailureRedirectInterceptor(dataDetailsHttp);
export function fetchDataDetails(payload) {
    const { token, equipmentId } = payload;
    return dataDetailsHttp.post('/business/datadetails/find', { equipmentId }, {
        headers: createAuthHeaders(token),
    });
}
export function batchAddDataDetails(payload) {
    const { token, body } = payload;
    return dataDetailsHttp.post('/business/datadetails/batchadd', body, {
        headers: createAuthHeaders(token),
    });
}
export function deleteDataDetails(payload) {
    const { token, id } = payload;
    return dataDetailsHttp.post('/business/datadetails/delete', { id }, {
        headers: createAuthHeaders(token),
    });
}
