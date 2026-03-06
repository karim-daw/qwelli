import { ChevronRight, Folder, FolderOpen, FileText, Loader2, DatabaseZap } from "lucide-react";
import { useFileTreeContext } from "./FileTreeContext";
import type { BrowseEntry } from "@/types/browse";
import { cn } from "@/lib/utils";

interface FileTreeNodeProps {
    entry: BrowseEntry;
    depth: number;
}

export function FileTreeNode({ entry, depth }: FileTreeNodeProps) {
    const ctx = useFileTreeContext();
    const { expandedNodes, cache, loadingNodes, indexedPaths, filterQuery, toggleNode, onIndex, onSelectIndex } = ctx;

    const expanded = expandedNodes.has(entry.path);
    const loading = loadingNodes.has(entry.path);
    const childResponse = cache.get(entry.path);
    const isIndexed =
        entry.isIndexed ||
        indexedPaths.has(entry.path.toLowerCase().replace(/\\/g, "/"));

    const paddingLeft = 12 + depth * 16;

    const handleClick = () => {
        if (!entry.isDir) return;
        if (isIndexed) {
            // Selecting an indexed folder activates it for search
            onSelectIndex(entry.path);
        }
        toggleNode(entry.path);
    };

    // Filter children by query
    const allChildren = childResponse?.entries ?? [];
    const visibleChildren = filterQuery
        ? allChildren.filter((c) =>
              c.name.toLowerCase().includes(filterQuery.toLowerCase()),
          )
        : allChildren;

    return (
        <div>
            {/* Row */}
            <div
                className={cn(
                    "group flex items-center gap-1.5 py-1 pr-2 cursor-pointer select-none",
                    "hover:bg-muted/50 transition-colors text-sm",
                    !entry.isDir && "cursor-default text-muted-foreground",
                )}
                style={{ paddingLeft }}
                onClick={handleClick}
            >
                {/* Chevron / spinner (dirs only) */}
                <span className="w-3.5 flex-shrink-0 flex items-center justify-center">
                    {entry.isDir &&
                        (loading ? (
                            <Loader2 className="w-3 h-3 animate-spin text-muted-foreground" />
                        ) : (
                            <ChevronRight
                                className={cn(
                                    "w-3.5 h-3.5 text-muted-foreground transition-transform duration-150",
                                    expanded && "rotate-90",
                                )}
                            />
                        ))}
                </span>

                {/* Folder / file icon */}
                {entry.isDir ? (
                    expanded ? (
                        <FolderOpen className="w-3.5 h-3.5 flex-shrink-0 text-amber-500 dark:text-amber-400" />
                    ) : (
                        <Folder className="w-3.5 h-3.5 flex-shrink-0 text-amber-500 dark:text-amber-400" />
                    )
                ) : (
                    <FileText className="w-3.5 h-3.5 flex-shrink-0 text-muted-foreground/60" />
                )}

                {/* Name */}
                <span className="truncate flex-1 font-mono text-xs leading-none">
                    {entry.name}
                </span>

                {/* Indexed dot */}
                {isIndexed && (
                    <span
                        className="w-1.5 h-1.5 rounded-full bg-emerald-500 flex-shrink-0 ml-0.5"
                        title="Indexed"
                    />
                )}

                {/* Index action (dirs only, non-indexed) */}
                {entry.isDir && !isIndexed && (
                    <button
                        className="opacity-0 group-hover:opacity-100 flex-shrink-0 p-0.5 rounded hover:bg-accent transition-opacity"
                        title="Index this folder"
                        onClick={(e) => {
                            e.stopPropagation();
                            onIndex(entry.path);
                        }}
                    >
                        <DatabaseZap className="w-3 h-3 text-muted-foreground hover:text-foreground" />
                    </button>
                )}
            </div>

            {/* Children — animated with CSS grid expand trick */}
            {entry.isDir && (
                <div
                    className="grid transition-[grid-template-rows] duration-200 ease-out"
                    style={{ gridTemplateRows: expanded ? "1fr" : "0fr" }}
                >
                    <div className="overflow-hidden">
                        {visibleChildren.map((child) => (
                            <FileTreeNode
                                key={child.path}
                                entry={child}
                                depth={depth + 1}
                            />
                        ))}

                        {/* Truncation notice */}
                        {childResponse?.truncated && !filterQuery && (
                            <div
                                className="text-[10px] text-muted-foreground/60 italic py-1"
                                style={{ paddingLeft: paddingLeft + 20 }}
                            >
                                Showing 500 of {childResponse.total} —{" "}
                                use filter to narrow
                            </div>
                        )}

                        {/* Filter active but no matches */}
                        {filterQuery &&
                            expanded &&
                            allChildren.length > 0 &&
                            visibleChildren.length === 0 && (
                                <div
                                    className="text-[10px] text-muted-foreground/60 italic py-1"
                                    style={{ paddingLeft: paddingLeft + 20 }}
                                >
                                    No matches
                                </div>
                            )}
                    </div>
                </div>
            )}
        </div>
    );
}
