export interface BrowseEntry {
    name: string;
    path: string;
    isDir: boolean;
    size?: number;
    modifiedAt?: string;
    isIndexed?: boolean;
}

export interface BrowseResponse {
    entries: BrowseEntry[];
    parent: string;
    total: number;
    truncated?: boolean;
}
