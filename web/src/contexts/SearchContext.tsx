import { createContext, useContext, useState, useCallback, useMemo, type ReactNode } from "react";
import type { SearchResult, RecentSearch } from "@/types/search";

const STORAGE_KEY = "qwelli_recent_searches";
const MAX_RECENT_SEARCHES = 10;

function readFromStorage(indexPath: string): RecentSearch[] {
    try {
        const stored = localStorage.getItem(STORAGE_KEY);
        if (!stored) return [];
        const all: Record<string, RecentSearch[]> = JSON.parse(stored);
        return all[indexPath] || [];
    } catch {
        return [];
    }
}

function writeToStorage(indexPath: string, searches: RecentSearch[]) {
    try {
        const stored = localStorage.getItem(STORAGE_KEY);
        const all: Record<string, RecentSearch[]> = stored ? JSON.parse(stored) : {};
        all[indexPath] = searches;
        localStorage.setItem(STORAGE_KEY, JSON.stringify(all));
    } catch {
        // ignore storage errors
    }
}

interface SearchContextValue {
    query: string;
    strategy: string;
    topK: number;
    contentFilter: string;
    results: SearchResult[];
    loading: boolean;
    cacheStatus: string | null;
    recentSearches: RecentSearch[];
    setQuery: (q: string) => void;
    setStrategy: (s: string) => void;
    setTopK: (k: number) => void;
    setContentFilter: (f: string) => void;
    setResults: (r: SearchResult[]) => void;
    setLoading: (l: boolean) => void;
    setCacheStatus: (c: string | null) => void;
    reset: () => void;
    loadRecentSearches: (indexPath: string) => void;
    saveRecentSearch: (indexPath: string, search: RecentSearch) => void;
    clearRecentSearches: (indexPath: string) => void;
}

const SearchContext = createContext<SearchContextValue | null>(null);

export function SearchProvider({ children }: { children: ReactNode }) {
    const [query, setQuery] = useState("");
    const [strategy, setStrategy] = useState("semantic");
    const [topK, setTopK] = useState(10);
    const [contentFilter, setContentFilter] = useState("all");
    const [results, setResults] = useState<SearchResult[]>([]);
    const [loading, setLoading] = useState(false);
    const [cacheStatus, setCacheStatus] = useState<string | null>(null);
    const [recentSearches, setRecentSearches] = useState<RecentSearch[]>([]);

    const reset = useCallback(() => {
        setQuery("");
        setStrategy("semantic");
        setTopK(10);
        setContentFilter("all");
        setResults([]);
        setLoading(false);
        setCacheStatus(null);
    }, []);

    const loadRecentSearches = useCallback((indexPath: string) => {
        setRecentSearches(readFromStorage(indexPath));
    }, []);

    const saveRecentSearch = useCallback((indexPath: string, search: RecentSearch) => {
        setRecentSearches((prev) => {
            const filtered = prev.filter(
                (s) =>
                    !(
                        s.query === search.query &&
                        s.strategy === search.strategy &&
                        s.contentFilter === search.contentFilter &&
                        s.topK === search.topK
                    ),
            );
            const updated = [search, ...filtered].slice(0, MAX_RECENT_SEARCHES);
            writeToStorage(indexPath, updated);
            return updated;
        });
    }, []);

    const clearRecentSearches = useCallback((indexPath: string) => {
        writeToStorage(indexPath, []);
        setRecentSearches([]);
    }, []);

    const value = useMemo(
        () => ({
            query,
            strategy,
            topK,
            contentFilter,
            results,
            loading,
            cacheStatus,
            recentSearches,
            setQuery,
            setStrategy,
            setTopK,
            setContentFilter,
            setResults,
            setLoading,
            setCacheStatus,
            reset,
            loadRecentSearches,
            saveRecentSearch,
            clearRecentSearches,
        }),
        [
            query,
            strategy,
            topK,
            contentFilter,
            results,
            loading,
            cacheStatus,
            recentSearches,
            reset,
            loadRecentSearches,
            saveRecentSearch,
            clearRecentSearches,
        ],
    );

    return (
        <SearchContext.Provider value={value}>
            {children}
        </SearchContext.Provider>
    );
}

export function useSearchContext() {
    const context = useContext(SearchContext);
    if (!context) {
        throw new Error(
            "useSearchContext must be used within a SearchProvider",
        );
    }
    return context;
}
