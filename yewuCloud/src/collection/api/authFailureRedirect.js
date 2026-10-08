import { redirectToMainLogin } from '../utils/mainLoginRedirect';
const AUTH_INVALID_KEYWORDS = [
    '未启用',
    'token过期',
    '令牌过期',
    'token验证失败',
    '令牌验证失败',
    '登录超时',
    '登录已失效',
    '会话已失效',
    '请重新登录',
    'token expired',
    'jwt expired',
    'invalid token',
    'token invalid',
    'token verification failed',
];
export function isAuthSessionInvalidMessage(msg) {
    if (typeof msg !== 'string')
        return false;
    const text = msg.trim().toLowerCase();
    if (!text)
        return false;
    return AUTH_INVALID_KEYWORDS.some(keyword => text.includes(keyword.toLowerCase()));
}
function isAuthFailureBody(data) {
    if (!data || typeof data !== 'object')
        return false;
    const body = data;
    if (isAuthSessionInvalidMessage(body.msg))
        return true;
    return body.code === 401 || body.code === '401';
}
function shouldHandleAuthFailure(config) {
    const url = String(config?.url ?? '');
    if (url.includes('/auth/login'))
        return false;
    return true;
}
let redirecting = false;
export function redirectToLoginOnAuthFailure() {
    if (redirecting)
        return;
    redirecting = true;
    redirectToMainLogin({ broadcast: true });
}
export function attachAuthFailureRedirectInterceptor(client) {
    client.interceptors.response.use((response) => {
        if (shouldHandleAuthFailure(response.config) && isAuthFailureBody(response.data)) {
            redirectToLoginOnAuthFailure();
        }
        return response;
    }, error => {
        const config = error.config;
        if (!shouldHandleAuthFailure(config)) {
            return Promise.reject(error);
        }
        const status = error.response?.status;
        if (status === 401 || isAuthFailureBody(error.response?.data)) {
            redirectToLoginOnAuthFailure();
        }
        return Promise.reject(error);
    });
}
