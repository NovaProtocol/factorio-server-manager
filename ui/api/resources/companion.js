import client from "../client";

const query = params => Object.entries(params)
    .filter(([, value]) => value !== undefined && value !== null && value !== "")
    .map(([key, value]) => `${key}=${encodeURIComponent(value)}`)
    .join("&");

export default {
    overview: async () => {
        const response = await client.get('/api/companion/overview');
        return response.data;
    },
    power: async (surface, window) => {
        const response = await client.get(`/api/companion/power?${query({surface, window})}`);
        return response.data;
    },
    production: async (surface, window) => {
        const response = await client.get(`/api/companion/production?${query({surface, window})}`);
        return response.data;
    },
    science: async (surface, window) => {
        const response = await client.get(`/api/companion/science?${query({surface, window})}`);
        return response.data;
    },
    graph: async (surface, category, series, window) => {
        const response = await client.get(`/api/companion/graph?${query({surface, category, series, window})}`);
        return response.data;
    },
    setSpeed: async (speed) => {
        const response = await client.post('/api/companion/speed', {speed});
        return response.data;
    }
}
