import client from "../client";
import uploadFiles from "../upload";

export default {
    list: async (latest) => {
        const response = await client.get('/api/saves/list', {
            params: {
                latest
            }
        });
        return response.data;
    },
    delete: async (save) => {
        const response = await client.post(`/api/saves/rm/${save.name}`);
        return response.data;
    },
    create: async (name) => {
        const response = await client.post(`/api/saves/create/${name}`);
        return response.data;
    },
    upload: async (files, onProgress) => uploadFiles(files, {
        url: '/api/saves/upload',
        field: 'savefile',
        onProgress
    }),
    mods: async save => {
        const response = await client.post("/api/saves/mods", {
            saveFile: save
        });
        return response.data;
    },
    info: async save => {
        const response = await client.get(`/api/saves/info/${encodeURIComponent(save)}`);
        return response.data;
    }
}