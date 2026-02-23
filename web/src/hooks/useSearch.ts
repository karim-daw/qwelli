import { useCallback, useRef, useEffect } from "react";
import { useSearchContext } from "@/contexts/SearchContext";
import { useAppContext } from "@/contexts/AppContext";
import * as searchApi from "@/api/search";
import type { RecentSearch } from "@/types/search";

export function useSearch() {
    const search = useSearchContext();
    const { selectedIndex } = useAppContext();
    const abortRef = useRef<AbortController | null>(null);

    // Abort in-flight search on unmount
    useEffect(() => {
        return () => {
            abortRef.current?.abort();
        };
    }, []);

    const executeSearch = useCallback(async () => {
        if (!search.query.trim() || !selectedIndex) return;

        // Abort any previous in-flight search
        abortRef.current?.abort();
        const controller = new AbortController();
        abortRef.current = controller;

        search.setLoading(true);
        try {
            const response = await searchApi.search(
                {
                    query: search.query,
                    index: selectedIndex,
                    strategy: search.strategy,
                    topK: search.topK,
                    contentFilter: search.contentFilter,
                },
                controller.signal,
            );

            search.setCacheStatus(response.cacheStatus);
            search.setResults(response.results);

            // Strip large fields (imageData, content) to keep recent searches lightweight
            const lightResults = response.results.map(
                ({ imageData, content, ...rest }) => rest,
            );

            search.saveRecentSearch(selectedIndex, {
                query: search.query,
                strategy: search.strategy,
                contentFilter: search.contentFilter,
                topK: search.topK,
                timestamp: Date.now(),
                resultCount: response.results.length,
                results: lightResults as typeof response.results,
                cacheStatus: response.cacheStatus || undefined,
            });
        } catch (error) {
            if ((error as Error).name === "AbortError") return;
            console.error("Search failed:", error);
        } finally {
            if (!controller.signal.aborted) {
                search.setLoading(false);
            }
        }
    }, [search, selectedIndex]);

    const loadCachedSearch = useCallback(
        (cached: RecentSearch) => {
            search.setQuery(cached.query);
            search.setStrategy(cached.strategy);
            search.setContentFilter(cached.contentFilter);
            search.setTopK(cached.topK);
            search.setResults(cached.results);
            search.setCacheStatus("LOCAL");
        },
        [search],
    );

    return { executeSearch, loadCachedSearch };
}
