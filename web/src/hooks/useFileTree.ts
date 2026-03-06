import { useState, useCallback, useMemo, useRef, useEffect } from "react";
import { useAppContext } from "@/contexts/AppContext";
import * as browseApi from "@/api/browse";
import type { BrowseResponse } from "@/types/browse";

const STORAGE_ROOT_KEY = "qwelli_tree_root";
const STORAGE_EXPANDED_KEY = "qwelli_tree_expanded";

function loadStoredRoot(): string | null {
    try {
        return localStorage.getItem(STORAGE_ROOT_KEY);
    } catch {
        return null;
    }
}

function loadStoredExpanded(): Set<string> {
    try {
        const raw = localStorage.getItem(STORAGE_EXPANDED_KEY);
        if (raw) return new Set(JSON.parse(raw) as string[]);
    } catch {}
    return new Set();
}

function normalizePath(p: string): string {
    return p.toLowerCase().replace(/\\/g, "/");
}

export function useFileTree() {
    const { indexes } = useAppContext();

    const [rootPath, setRootPathState] = useState<string | null>(loadStoredRoot);
    const [expandedNodes, setExpandedNodes] = useState<Set<string>>(loadStoredExpanded);
    const [cache, setCache] = useState<Map<string, BrowseResponse>>(new Map());
    const [loadingNodes, setLoadingNodes] = useState<Set<string>>(new Set());
    const [filterQuery, setFilterQueryState] = useState("");

    // Use refs for values needed inside callbacks without causing stale closures
    const cacheRef = useRef(cache);
    cacheRef.current = cache;

    const abortControllers = useRef<Map<string, AbortController>>(new Map());

    // Cancel all in-flight requests on unmount
    useEffect(() => {
        return () => {
            for (const ctrl of abortControllers.current.values()) {
                ctrl.abort();
            }
        };
    }, []);

    // Derive indexed paths from AppContext.indexes — updates automatically
    // after fetchIndexes() runs (e.g. after indexing completes)
    const indexedPaths = useMemo(() => {
        const set = new Set<string>();
        for (const idx of indexes) {
            set.add(normalizePath(idx.path));
        }
        return set;
    }, [indexes]);

    const fetchNode = useCallback(async (path: string) => {
        // Cancel any prior in-flight request for this path
        abortControllers.current.get(path)?.abort();
        const ctrl = new AbortController();
        abortControllers.current.set(path, ctrl);

        setLoadingNodes((prev) => new Set(prev).add(path));

        try {
            const resp = await browseApi.listDirectory(path, ctrl.signal);
            setCache((prev) => new Map(prev).set(path, resp));
        } catch (err: unknown) {
            if (err instanceof Error && err.name === "AbortError") return;
            // Store empty response so a failed dir doesn't retry on every render
            setCache((prev) =>
                new Map(prev).set(path, { entries: [], parent: "", total: 0 }),
            );
        } finally {
            abortControllers.current.delete(path);
            setLoadingNodes((prev) => {
                const next = new Set(prev);
                next.delete(path);
                return next;
            });
        }
    }, []);

    // Fetch root on mount if persisted root exists but isn't cached yet
    useEffect(() => {
        if (rootPath && !cacheRef.current.has(rootPath)) {
            fetchNode(rootPath);
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const setRootPath = useCallback(
        (path: string | null) => {
            // Cancel all in-flight requests before switching root
            for (const ctrl of abortControllers.current.values()) {
                ctrl.abort();
            }
            abortControllers.current.clear();

            setRootPathState(path);
            setExpandedNodes(new Set());
            setCache(new Map());
            setLoadingNodes(new Set());

            try {
                if (path) localStorage.setItem(STORAGE_ROOT_KEY, path);
                else localStorage.removeItem(STORAGE_ROOT_KEY);
                localStorage.removeItem(STORAGE_EXPANDED_KEY);
            } catch {}

            if (path) fetchNode(path);
        },
        [fetchNode],
    );

    const toggleNode = useCallback(
        (path: string) => {
            setExpandedNodes((prev) => {
                const next = new Set(prev);
                if (next.has(path)) {
                    next.delete(path);
                } else {
                    next.add(path);
                    // Fetch children if not already in cache
                    if (!cacheRef.current.has(path)) {
                        fetchNode(path);
                    }
                }
                try {
                    localStorage.setItem(
                        STORAGE_EXPANDED_KEY,
                        JSON.stringify([...next]),
                    );
                } catch {}
                return next;
            });
        },
        [fetchNode],
    );

    const setFilterQuery = useCallback((q: string) => {
        setFilterQueryState(q);
    }, []);

    const refreshNode = useCallback(
        (path: string) => {
            fetchNode(path);
        },
        [fetchNode],
    );

    return {
        rootPath,
        setRootPath,
        rootResponse: rootPath ? cache.get(rootPath) : undefined,
        rootLoading: rootPath ? loadingNodes.has(rootPath) : false,
        expandedNodes,
        cache,
        loadingNodes,
        filterQuery,
        setFilterQuery,
        toggleNode,
        refreshNode,
        indexedPaths,
    };
}
