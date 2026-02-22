import type { SearchResult, SearchParams } from "@/types/search";

interface SearchResponse {
    results: SearchResult[];
    query: string;
    cacheStatus: string | null;
}

export async function search(params: SearchParams): Promise<SearchResponse> {
    const urlParams = new URLSearchParams({
        q: params.query,
        index: params.index,
        strategy: params.strategy,
        top: params.topK.toString(),
    });

    if (params.contentFilter && params.contentFilter !== "all") {
        urlParams.append("content_type", params.contentFilter);
    }

    const response = await fetch("/api/search?" + urlParams.toString());
    const data = await response.json();
    const cacheStatus = response.headers.get("X-Cache-Status");

    return {
        results: data.results || [],
        query: params.query,
        cacheStatus,
    };
}
