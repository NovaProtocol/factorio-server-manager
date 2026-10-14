import client from "./client";

/**
 * Upload files one request at a time, reporting progress as it goes.
 *
 * Generic on purpose, so any upload in the manager can drive a ProgressBar from
 * it without repeating the bookkeeping.
 *
 * @param files       a FileList or an array of File
 * @param url         endpoint that accepts the multipart body
 * @param field       form field name the endpoint expects for each file
 * @param onProgress  called with {overall, current}, where overall is 0 to 100
 *                    across every file and current is {name, index, count, progress}
 * @param fields      extra form fields sent with each request
 * @returns the response body of each request, in order
 */
export const uploadFiles = async (files, {url, field, onProgress, fields = {}}) => {
    const list = Array.from(files);
    const results = [];

    for (let index = 0; index < list.length; index++) {
        const file = list[index];
        const body = new FormData();
        body.append(field, file);
        Object.entries(fields).forEach(([key, value]) => body.append(key, value));

        const response = await client.post(url, body, {
            headers: {
                "Content-Type": "multipart/form-data"
            },
            onUploadProgress: event => {
                const ratio = event.total ? event.loaded / event.total : 0;

                if (onProgress) {
                    onProgress({
                        overall: ((index + ratio) * 100) / list.length,
                        current: {
                            name: file.name,
                            index,
                            count: list.length,
                            progress: ratio * 100
                        }
                    });
                }
            }
        });

        results.push(response.data);
    }

    return results;
}

export default uploadFiles;
