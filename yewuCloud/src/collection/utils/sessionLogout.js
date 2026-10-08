export function notifyBackendLogout(token) {
    if (!token || typeof window === 'undefined' || typeof window.fetch !== 'function') {
        return;
    }
    void window.fetch('/auth/logout', {
        method: 'POST',
        headers: {
            Authorization: `Bearer ${token}`,
        },
        credentials: 'same-origin',
        keepalive: true,
    }).catch(() => {
        // Local logout must continue even when the server is unavailable.
    });
}
