export function createAuthHeaders(token) {
    return {
        Authorization: `Bearer ${token}`,
        AccessToken: token,
    };
}
