import {useCallback, useEffect, useRef, useState} from "react";

// Polls one companion scope. The key is what drives it: when the surface or the
// window changes the key changes and the fetch restarts, and otherwise the same
// fetch repeats on a timer so the page stays live without a reload. The fetcher
// is held in a ref so a new closure on each render cannot restart the interval.
// There is no client caching: every poll goes to the manager, which answers from
// its own cache (kept in memory for the life of the process) and refreshes behind
// the request. The payload carries the age of that cached data, which is the number
// worth showing: the browser's own clock only says when it last heard something.
export const useScopedReport = (key, fetcher, intervalMs = 5000) => {

    const [payload, setPayload] = useState(null);
    const [isLoading, setIsLoading] = useState(true);
    const [updatedAt, setUpdatedAt] = useState(null);
    const [dataAge, setDataAge] = useState(null);
    const fetcherRef = useRef(fetcher);
    fetcherRef.current = fetcher;

    const load = useCallback(async () => {
        try {
            const next = await fetcherRef.current();
            setPayload(next);
            setDataAge(typeof next?.age_seconds === 'number' && next.age_seconds >= 0 ? next.age_seconds : null);
            setUpdatedAt(Date.now());
        } catch (e) {
            // keep the last good payload; the panel shows its age instead
        } finally {
            setIsLoading(false);
        }
    }, []);

    useEffect(() => {
        setIsLoading(true);
        load();

        const timer = setInterval(load, intervalMs);
        return () => clearInterval(timer);
    }, [key, load, intervalMs]);

    return {payload, isLoading, updatedAt, dataAge, refresh: load};
};
