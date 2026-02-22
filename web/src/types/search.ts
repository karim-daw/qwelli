export interface SearchResult {
    chunkId: string;
    content: string;
    filePath: string;
    fileName: string;
    contentType: string;
    similarity: number;
    pageNumbers?: number[];
    chunkIndex?: number;
    totalChunks?: number;
    fileSize?: number;
    modifiedDate?: string;
    imageData?: string;
    metadata?: Record<string, unknown>;
}

export interface RecentSearch {
    query: string;
    strategy: string;
    contentFilter: string;
    topK: number;
    timestamp: number;
    resultCount: number;
    results: SearchResult[];
    cacheStatus?: string;
}

export type SearchStrategy = "semantic" | "keyword" | "hybrid";
export type ContentFilter = "all" | "text" | "image";

export interface SearchParams {
    query: string;
    index: string;
    strategy: string;
    topK: number;
    contentFilter?: string;
}
