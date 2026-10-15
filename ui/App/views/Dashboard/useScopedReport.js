import {useCallback, useEffect, useRef, useState} from "react";

// Polls one companion scope. The key is what drives it: when the surface or the
// window changes the key changes and the fetch restarts, and otherwise the same
// fetch repeats on a timer so the page stays live without a reload. The fetcher
// is held in a ref so a new closure on each render cannot restart the interval.
// There is no client caching: every poll goes to the manager, whose short cache
// decides whether the game gets asked. If a fetch cannot be served right now the
// poll still answers — with slightly older data — and the next 5s will bring it up.
export const useScopedReport = (key, fetcher, intervalMs = 5000) => {

    const [payload, setPayload] = useState(null);
    const [isLoading, setIsLoading] = useState(true);
    const [updatedAt, setUpdatedAt] = useState(null);
    const fetcherRef = useRef(fetcher);
    fetcherRef.current = fetcher;

    const load = useCallback(async () => {
        try {
            setPayload(await fetcherRef.current());
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

    return {payload, isLoading, updatedAt, refresh: load};
};
