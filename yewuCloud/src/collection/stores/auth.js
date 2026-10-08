import { defineStore } from 'pinia';
import { ref } from 'vue';
import { getToken, getUsername, removeToken } from '@/utils/auth.js';
const TOKEN_KEY = 'access_token';
const USERNAME_KEY = 'login_username';
const ACCOUNT_KEY = 'login_account';
export const useAuthStore = defineStore('auth', () => {
    // 与主前端共享同一登录会话，不再维护采集平台自己的登录状态。
    const token = ref(getToken() || localStorage.getItem(TOKEN_KEY) || '');
    const username = ref(getUsername() || localStorage.getItem(USERNAME_KEY) || '');
    const account = ref(localStorage.getItem(ACCOUNT_KEY) || localStorage.getItem(USERNAME_KEY) || '');
    const isAdmin = ref(false);
    const permissionUrls = ref([]);
    const accessLoaded = ref(false);
    function setToken(value) {
        if (token.value !== value)
            clearAccess();
        token.value = value;
        localStorage.setItem(TOKEN_KEY, value);
    }
    function setUsername(value) {
        username.value = value;
        localStorage.setItem(USERNAME_KEY, value);
    }
    function setAccount(value) {
        account.value = value;
        localStorage.setItem(ACCOUNT_KEY, value);
    }
    function setAccess(admin, urls) {
        isAdmin.value = admin;
        permissionUrls.value = [...new Set(urls.map(item => item.trim()).filter(Boolean))];
        accessLoaded.value = true;
    }
    function clearAccess() {
        isAdmin.value = false;
        permissionUrls.value = [];
        accessLoaded.value = false;
    }
    function hasAnyPermission(...urls) {
        if (isAdmin.value)
            return true;
        const owned = new Set(permissionUrls.value);
        return urls.some(url => owned.has(url));
    }
    function logout() {
        token.value = '';
        removeToken();
        localStorage.removeItem(TOKEN_KEY);
        username.value = '';
        localStorage.removeItem(USERNAME_KEY);
        account.value = '';
        localStorage.removeItem(ACCOUNT_KEY);
        clearAccess();
    }
    return {
        token,
        username,
        account,
        isAdmin,
        permissionUrls,
        accessLoaded,
        setToken,
        setUsername,
        setAccount,
        setAccess,
        clearAccess,
        hasAnyPermission,
        logout,
    };
});
