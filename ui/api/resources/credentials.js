import client from "../client";

export default {
    status: async () => {
        const response = await client.get('/api/credentials/status');
        return response.data;
    },
    login: async (username, password) => {
        const response = await client.post('/api/credentials/login', {username, password});
        return response.data;
    },
    loginWithToken: async (username, token) => {
        const response = await client.post('/api/credentials/token', {username, token});
        return response.data;
    },
    logout: async () => {
        const response = await client.get('/api/credentials/logout');
        return response.data;
    }
}
