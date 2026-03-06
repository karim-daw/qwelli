import { useState, useCallback, useMemo } from "react";
import {
    Search,
    X,
    Folder,
    Activity,
    ChevronDown,
    ChevronRight,
    FolderOpen,
    ArrowUp,
    Loader2,
} from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useAppContext } from "@/contexts/AppContext";
import { useSearchContext } from "@/contexts/SearchContext";
import { useIndexes } from "@/hooks/useIndexes";
import { useFileTree } from "@/hooks/useFileTree";
import * as filesApi from "@/api/files";
import { toast } from "sonner";
import { FileTreeContext } from "./FileTreeContext";
import { FileTreeNode } from "./FileTreeNode";

interface FileTreeProps {
    width: number;
    onResizeStart: () => void;
    onCreateIndex: (path: string, contentType: string) => void;
}

export function FileTree({ width, onResizeStart, onCreateIndex }: FileTreeProps) {
    const { setSelectedIndex, setViewMode } = useAppContext();
    const { reset: resetSearch } = useSearchContext();
    const { indexes, deleteIndex } = useIndexes();

    const tree = useFileTree();

    const [rootInput, setRootInput] = useState("");
    const [showIndexed, setShowIndexed] = useState(true);
    const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

    // ── Indexed strip actions ──────────────────────────────────────────────

    const handleSelectIndex = useCallback(
        (path: string) => {
            setSelectedIndex(path);
            setViewMode("search");
            resetSearch();
        },
        [setSelectedIndex, setViewMode, resetSearch],
    );

    const handleViewStatus = useCallback(
        (path: string) => {
            setSelectedIndex(path);
            setViewMode("status");
        },
        [setSelectedIndex, setViewMode],
    );

    const handleOpenFolder = useCallback(async (path: string) => {
        try {
            await filesApi.openFolder(path);
        } catch {
            toast.error("Failed to open folder");
        }
    }, []);

    const handleConfirmDelete = useCallback(async () => {
        if (deleteConfirm) {
            await deleteIndex(deleteConfirm);
            resetSearch();
            setDeleteConfirm(null);
        }
    }, [deleteConfirm, deleteIndex, resetSearch]);

    // ── Root picker ────────────────────────────────────────────────────────

    const handleSetRoot = useCallback(() => {
        const trimmed = rootInput.trim();
        if (!trimmed) return;
        tree.setRootPath(trimmed);
        setRootInput("");
    }, [rootInput, tree]);

    const handleClearRoot = useCallback(() => {
        tree.setRootPath(null);
    }, [tree]);

    const handleNavigateUp = useCallback(() => {
        const parent = tree.rootResponse?.parent;
        if (parent) tree.setRootPath(parent);
    }, [tree]);

    // ── Tree context callbacks ─────────────────────────────────────────────

    const onIndex = useCallback(
        (path: string) => {
            onCreateIndex(path, "all");
        },
        [onCreateIndex],
    );

    const onSelectIndexFromTree = useCallback(
        (path: string) => {
            handleSelectIndex(path);
        },
        [handleSelectIndex],
    );

    // ── Context value (memoized to avoid cascading re-renders) ─────────────

    const contextValue = useMemo(
        () => ({
            expandedNodes: tree.expandedNodes,
            cache: tree.cache,
            loadingNodes: tree.loadingNodes,
            indexedPaths: tree.indexedPaths,
            filterQuery: tree.filterQuery,
            toggleNode: tree.toggleNode,
            onIndex,
            onSelectIndex: onSelectIndexFromTree,
        }),
        [tree.expandedNodes, tree.cache, tree.loadingNodes, tree.indexedPaths,
         tree.filterQuery, tree.toggleNode, onIndex, onSelectIndexFromTree],
    );

    // ── Root path display (last two path segments for brevity) ────────────
    const rootLabel = useMemo(() => {
        if (!tree.rootPath) return null;
        const parts = tree.rootPath.replace(/\\/g, "/").split("/").filter(Boolean);
        if (parts.length <= 2) return tree.rootPath;
        return "…/" + parts.slice(-2).join("/");
    }, [tree.rootPath]);

    return (
        <div style={{ width }} className="flex-shrink-0 border-r relative flex flex-col h-full">

            {/* ── Header ────────────────────────────────────────────────── */}
            <div className="p-3 border-b flex-shrink-0">
                <div className="flex items-center justify-between mb-2">
                    <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                        Explorer
                    </span>
                </div>

                {/* Filter input (only shown when a root is set) */}
                {tree.rootPath && (
                    <div className="relative">
                        <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3 h-3 text-muted-foreground pointer-events-none" />
                        <Input
                            placeholder="Filter…"
                            value={tree.filterQuery}
                            onChange={(e) => tree.setFilterQuery(e.target.value)}
                            className="h-7 pl-6 pr-6 text-xs font-mono"
                        />
                        {tree.filterQuery && (
                            <button
                                className="absolute right-1.5 top-1/2 -translate-y-1/2 p-0.5 rounded hover:bg-muted"
                                onClick={() => tree.setFilterQuery("")}
                            >
                                <X className="w-3 h-3 text-muted-foreground" />
                            </button>
                        )}
                    </div>
                )}
            </div>

            {/* ── Indexed folders strip ─────────────────────────────────── */}
            {indexes.length > 0 && (
                <div className="flex-shrink-0 border-b">
                    <button
                        className="w-full flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wider hover:bg-muted/50 transition-colors"
                        onClick={() => setShowIndexed((v) => !v)}
                    >
                        {showIndexed ? (
                            <ChevronDown className="w-3 h-3" />
                        ) : (
                            <ChevronRight className="w-3 h-3" />
                        )}
                        Indexed ({indexes.length})
                    </button>

                    {showIndexed && (
                        <div>
                            {indexes.map((index) => (
                                <div
                                    key={index.path}
                                    onClick={() => handleSelectIndex(index.path)}
                                    className="group w-full text-left px-3 py-1.5 hover:bg-accent transition-colors flex items-center gap-2 cursor-pointer"
                                >
                                    <button
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            handleOpenFolder(index.path);
                                        }}
                                        className="p-0.5 hover:bg-accent rounded"
                                        title="Open in Explorer"
                                    >
                                        <Folder className="w-3.5 h-3.5 text-amber-500 dark:text-amber-400 flex-shrink-0" />
                                    </button>
                                    <span className="text-xs truncate flex-1 font-mono">
                                        {index.name}
                                    </span>
                                    <span className="text-[10px] text-muted-foreground flex-shrink-0">
                                        {index.documentCount}
                                    </span>
                                    <button
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            handleViewStatus(index.path);
                                        }}
                                        className="opacity-0 group-hover:opacity-100 p-0.5 hover:bg-accent rounded transition-opacity"
                                        title="View status"
                                    >
                                        <Activity className="w-3 h-3" />
                                    </button>
                                    <button
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            setDeleteConfirm(index.path);
                                        }}
                                        className="opacity-0 group-hover:opacity-100 p-0.5 hover:bg-red-500/20 rounded transition-opacity text-red-600 dark:text-red-400"
                                        title="Delete index"
                                    >
                                        <X className="w-3 h-3" />
                                    </button>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            )}

            {/* ── Tree body ─────────────────────────────────────────────── */}
            <div className="flex-1 overflow-y-auto custom-scrollbar min-h-0">

                {!tree.rootPath ? (
                    /* Root picker */
                    <div className="flex flex-col items-center justify-center gap-3 px-4 py-8">
                        <FolderOpen className="w-8 h-8 text-muted-foreground/30" />
                        <p className="text-xs text-muted-foreground text-center leading-relaxed">
                            Set a root folder to browse your files
                        </p>
                        <div className="w-full flex gap-1.5">
                            <Input
                                placeholder="\\server\share or C:\Users\..."
                                value={rootInput}
                                onChange={(e) => setRootInput(e.target.value)}
                                onKeyDown={(e) => e.key === "Enter" && handleSetRoot()}
                                className="h-7 text-xs font-mono"
                            />
                            <Button
                                size="sm"
                                variant="secondary"
                                className="h-7 px-2 text-xs"
                                onClick={handleSetRoot}
                                disabled={!rootInput.trim()}
                            >
                                Go
                            </Button>
                        </div>
                    </div>
                ) : (
                    <>
                        {/* Root path breadcrumb */}
                        <div className="flex items-center gap-1 px-2 py-1.5 border-b bg-muted/30 flex-shrink-0">
                            {tree.rootResponse?.parent && (
                                <button
                                    onClick={handleNavigateUp}
                                    className="p-0.5 rounded hover:bg-muted transition-colors flex-shrink-0"
                                    title="Go up"
                                >
                                    <ArrowUp className="w-3 h-3 text-muted-foreground" />
                                </button>
                            )}
                            <span
                                className="text-[10px] font-mono text-muted-foreground truncate flex-1"
                                title={tree.rootPath}
                            >
                                {rootLabel}
                            </span>
                            {tree.rootLoading && (
                                <Loader2 className="w-3 h-3 animate-spin text-muted-foreground flex-shrink-0" />
                            )}
                            <button
                                onClick={handleClearRoot}
                                className="p-0.5 rounded hover:bg-muted transition-colors flex-shrink-0"
                                title="Change root folder"
                            >
                                <X className="w-3 h-3 text-muted-foreground" />
                            </button>
                        </div>

                        {/* Tree entries */}
                        <FileTreeContext.Provider value={contextValue}>
                            {(tree.rootResponse?.entries ?? []).map((entry) => (
                                <FileTreeNode
                                    key={entry.path}
                                    entry={entry}
                                    depth={0}
                                />
                            ))}

                            {/* Root truncation notice */}
                            {tree.rootResponse?.truncated && !tree.filterQuery && (
                                <div className="text-[10px] text-muted-foreground/60 italic px-4 py-2">
                                    Showing 500 of {tree.rootResponse.total} —
                                    use filter to narrow
                                </div>
                            )}

                            {/* Root-level filter with no matches */}
                            {tree.filterQuery &&
                                (tree.rootResponse?.entries ?? []).length > 0 &&
                                (tree.rootResponse?.entries ?? []).filter((e) =>
                                    e.name
                                        .toLowerCase()
                                        .includes(tree.filterQuery.toLowerCase()),
                                ).length === 0 && (
                                    <div className="text-[10px] text-muted-foreground/60 italic px-4 py-2">
                                        No matches in root
                                    </div>
                                )}
                        </FileTreeContext.Provider>
                    </>
                )}
            </div>

            {/* ── Resize handle ─────────────────────────────────────────── */}
            <div
                className="absolute top-0 right-0 w-1 h-full cursor-col-resize hover:bg-accent transition-colors z-10"
                onMouseDown={onResizeStart}
            />

            {/* ── Delete confirmation dialog ────────────────────────────── */}
            <AlertDialog
                open={!!deleteConfirm}
                onOpenChange={() => setDeleteConfirm(null)}
            >
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete Index</AlertDialogTitle>
                        <AlertDialogDescription>
                            Are you sure you want to delete this index? This
                            cannot be undone.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction onClick={handleConfirmDelete}>
                            Delete
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}
