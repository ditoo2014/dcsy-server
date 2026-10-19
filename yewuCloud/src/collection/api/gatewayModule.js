import axios from 'axios';
import { createAuthHeaders } from '../utils/request';
import { attachAuthFailureRedirectInterceptor } from './authFailureRedirect';
const gatewayModuleHttp = axios.create({
    baseURL: import.meta.env.VITE_AUTH_BASE_URL,
    timeout: Number(import.meta.env.VITE_API_TIMEOUT ?? 10000),
});
attachAuthFailureRedirectInterceptor(gatewayModuleHttp);
export function fetchGatewayOne(payload) {
    const { token, gatewayId } = payload;
    return gatewayModuleHttp.post('/tolink/gateway/findone', {}, {
        params: {
            GatewayID: gatewayId,
        },
        headers: createAuthHeaders(token),
    });
}
export function fetchGatewayModules(payload) {
    const { token, gatewayId } = payload;
    return gatewayModuleHttp.post('/tolink/gatewaymodule/find', {}, {
        params: {
            GatewayID: gatewayId,
        },
        headers: createAuthHeaders(token),
    });
}
export function addGatewayModule(payload) {
    const { token, body } = payload;
    return gatewayModuleHttp.post('/tolink/gatewaymodule/add', body, {
        params: {
            GatewayID: String(body?.gatewayId ?? ''),
            Module: 'drive',
        },
        headers: createAuthHeaders(token),
    });
}
export function updateGatewayModule(payload) {
    const { token, body } = payload;
    return gatewayModuleHttp.post('/tolink/gatewaymodule/update', body, {
        params: {
            GatewayID: String(body?.gatewayId ?? ''),
            Module: 'drive',
        },
        headers: createAuthHeaders(token),
    });
}
export function deleteGatewayPoint(payload) {
    const { token, gatewayId, body } = payload;
    return gatewayModuleHttp.post('/tolink/gatewaymodule/update', body, {
        params: {
            GatewayID: gatewayId,
            Function: 'deletepoint',
        },
        headers: createAuthHeaders(token),
    });
}
export function deleteGatewayModule(payload) {
    const { token, gatewayId, body } = payload;
    return gatewayModuleHttp.post('/tolink/gatewaymodule/delete', body, {
        params: {
            GatewayID: gatewayId,
        },
        headers: createAuthHeaders(token),
    });
}
