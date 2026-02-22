import { useCallback } from "react";
import { useSearchContext } from "@/contexts/SearchContext";
import { useAppContext } from "@/contexts/AppContext";
import * as searchApi from "@/api/search";
import type { RecentSearch } from "@/types/search";

export function useSearch() {
    const search = useSearchContext();
    const { selectedIndex } = useAppContext();

    const executeSearch = useCallback(async () => {
        if (!search.query.trim() || !selectedIndex) return;

        search.setLoading(true);
        try {
            const response = await searchApi.search({
                query: search.query,
                index: selectedIndex,
                strategy: search.strategy,
                topK: search.topK,
                contentFilter: search.contentFilter,
            });

            search.setCacheStatus(response.cacheStatus);
            search.setResults(response.results);

            search.saveRecentSearch(selectedIndex, {
                query: search.query,
                strategy: search.strategy,
                contentFilter: search.contentFilter,
                topK: search.topK,
                timestamp: Date.now(),
                resultCount: response.results.length,
                results: response.results,
                cacheStatus: response.cacheStatus || undefined,
            });
        } catch (error) {
            console.error("Search failed:", error);
        } finally {
            search.setLoading(false);
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
