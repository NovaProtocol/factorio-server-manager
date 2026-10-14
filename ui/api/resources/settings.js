import client from "../client";

export default {
    server: {
        list: async () => {
            const response = await client.get('/api/settings')
            return response.data;
        },
        update: async data => {
            const response = await client.post('/api/settings/update', data)
            return response.data;
        },
        // The values the game ships in server-settings.example.json, used to mark
        // a field that still holds its default.
        defaults: async () => {
            const response = await client.get('/api/settings/defaults')
            return response.data;
        }
    },
    game: {
        list: async () => {
            const response = await client.get('/api/config');
            return response.data;
        },
        modSettings: async () => {
            const response = await client.get('/api/game-settings');
            return response.data;
        },
        saveModSettings: async changes => {
            const response = await client.post('/api/game-settings', {changes});
            return response.data;
        },
        // Runs the game with --dump-data: the game evaluates every enabled mod's
        // settings.lua itself, and writes mod-settings.dat if it is missing.
        dumpModSettings: async () => {
            const response = await client.post('/api/game-settings/dump');
            return response.data;
        }
    }
}