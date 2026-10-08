import axios from 'axios';
import { createAuthHeaders } from '../utils/request';
import { attachAuthFailureRedirectInterceptor } from './authFailureRedirect';
const gatewayHttp = axios.create({
    baseURL: import.meta.env.VITE_AUTH_BASE_URL,
    timeout: Number(import.meta.env.VITE_API_TIMEOUT ?? 10000),
});
attachAuthFailureRedirectInterceptor(gatewayHttp);
export function fetchGatewayPage(payload) {
    const { token, pageNum, pageSize, name } = payload;
    return gatewayHttp.post(`/tolink/gateway/findpage?pageNum=${pageNum}&pageSize=${pageSize}`, {
        ...(name ? { name } : {}),
    }, {
        headers: createAuthHeaders(token),
    });
}
export function addGateway(payload) {
    const { token, ...body } = payload;
    return gatewayHttp.post('/tolink/gateway/add', body, {
        headers: createAuthHeaders(token),
    });
}
export function updateGateway(payload) {
    const { token, ...body } = payload;
    return gatewayHttp.post('/tolink/gateway/update', body, {
        headers: createAuthHeaders(token),
    });
}
export function deleteGateway(payload) {
    const { token, gatewayId } = payload;
    return gatewayHttp.post('/tolink/gateway/delete', { gatewayId }, {
        headers: createAuthHeaders(token),
    });
}
