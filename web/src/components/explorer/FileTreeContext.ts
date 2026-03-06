import { createContext, useContext } from "react";
import type { BrowseResponse } from "@/types/browse";

export interface FileTreeContextValue {
    expandedNodes: Set<string>;
    cache: Map<string, BrowseResponse>;
    loadingNodes: Set<string>;
    indexedPaths: Set<string>;
    filterQuery: string;
    toggleNode: (path: string) => void;
    onIndex: (path: string) => void;
    onSelectIndex: (path: string) => void;
}

export const FileTreeContext = createContext<FileTreeContextValue | null>(null);

export function useFileTreeContext(): FileTreeContextValue {
    const ctx = useContext(FileTreeContext);
    if (!ctx) throw new Error("useFileTreeContext used outside FileTreeContext.Provider");
    return ctx;
}
