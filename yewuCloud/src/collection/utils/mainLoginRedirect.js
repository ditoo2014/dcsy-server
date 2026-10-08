import { useAuthStore } from '../stores/auth';
import { useTagsViewStore } from '../stores/tagsView.js';
import { notifyBackendLogout } from './sessionLogout';
export const LAST_ACTIVITY_KEY = 'dicloud:last-user-activity-at';
export const SESSION_TIMEOUT_EVENT_KEY = 'dicloud:session-timeout-at';
let redirecting = false;
function removeCookie(name) {
    if (typeof document === 'undefined')
        return;
    document.cookie = `${name}=; Max-Age=0; path=/`;
    document.cookie = `${name}=; Max-Age=0`;
}
export function clearLocalAuthState() {
    try {
        useAuthStore().logout();
    }
    catch {
        if (typeof localStorage !== 'undefined') {
            localStorage.removeItem('access_token');
            localStorage.removeItem('login_username');
            localStorage.removeItem('login_account');
        }
    }
    removeCookie('access_token');
    // 退出时同时清理采集云自己的标签页，避免下次登录保留旧页面。
    useTagsViewStore().clearVisitedViews();
}
export function redirectToMainLogin(options = {}) {
    if (redirecting)
        return;
    redirecting = true;
    if (options.broadcast && typeof localStorage !== 'undefined') {
        localStorage.setItem(SESSION_TIMEOUT_EVENT_KEY, String(Date.now()));
    }
    if (typeof localStorage !== 'undefined') {
        localStorage.removeItem(LAST_ACTIVITY_KEY);
    }
    if (options.notifyBackend && typeof localStorage !== 'undefined') {
        notifyBackendLogout(localStorage.getItem('access_token') || '');
    }
    clearLocalAuthState();
    if (typeof window !== 'undefined') {
        const redirect = encodeURIComponent('/collection');
        window.location.replace(`${window.location.origin}${window.location.pathname}#/login?redirect=${redirect}`);
    }
}
