import { useState } from "react";
import {
    Search,
    Activity,
    FileText,
    FolderOpen,
    RefreshCw,
    Layers,
    Info,
    Files,
    ChevronRight,
    type LucideIcon,
} from "lucide-react";
import type { ToolCall } from "@/types/chat";

interface ToolMeta {
    icon: LucideIcon;
    color: string;
    bgClass: string;
    dotColor: string;
    label: string;
}

const TOOL_META: Record<string, ToolMeta> = {
    search: {
        icon: Search,
        color: "text-blue-500",
        bgClass: "bg-blue-500/15",
        dotColor: "bg-blue-500",
        label: "Search",
    },
    status: {
        icon: Activity,
        color: "text-amber-500",
        bgClass: "bg-amber-500/15",
        dotColor: "bg-amber-500",
        label: "Index Status",
    },
    read_file: {
        icon: FileText,
        color: "text-sky-500",
        bgClass: "bg-sky-500/15",
        dotColor: "bg-sky-500",
        label: "Read File",
    },
    list_dir: {
        icon: FolderOpen,
        color: "text-slate-400",
        bgClass: "bg-slate-400/15",
        dotColor: "bg-slate-400",
        label: "List Directory",
    },
    index_update: {
        icon: RefreshCw,
        color: "text-green-500",
        bgClass: "bg-green-500/15",
        dotColor: "bg-green-500",
        label: "Update Index",
    },
    get_file_chunks: {
        icon: Layers,
        color: "text-violet-500",
        bgClass: "bg-violet-500/15",
        dotColor: "bg-violet-500",
        label: "Read Chunks",
    },
    get_file_info: {
        icon: Info,
        color: "text-cyan-500",
        bgClass: "bg-cyan-500/15",
        dotColor: "bg-cyan-500",
        label: "File Info",
    },
    find_files: {
        icon: Files,
        color: "text-indigo-500",
        bgClass: "bg-indigo-500/15",
        dotColor: "bg-indigo-500",
        label: "Find Files",
    },
};

const DEFAULT_META: ToolMeta = {
    icon: Activity,
    color: "text-muted-foreground",
    bgClass: "bg-muted",
    dotColor: "bg-muted-foreground",
    label: "",
};

function formatArgs(input: Record<string, unknown>): string {
    const parts: string[] = [];
    for (const [key, value] of Object.entries(input)) {
        if (typeof value === "string") {
            const display =
                value.length > 60 ? value.slice(0, 57) + "..." : value;
            parts.push(`${key}="${display}"`);
        } else if (value !== undefined && value !== null) {
            parts.push(`${key}=${JSON.stringify(value)}`);
        }
    }
    return parts.join(" ");
}

function formatResultSummary(name: string, result: string): string {
    try {
        const data = JSON.parse(result);

        if (name === "search" && Array.isArray(data)) {
            const files = new Set(
                data.map((r: { file_path?: string }) => r.file_path),
            );
            return `Found ${data.length} results across ${files.size} files`;
        }
        if (
            name === "find_files" &&
            typeof data === "object" &&
            data !== null
        ) {
            const count = data.total_matches ?? 0;
            return `Found ${count} files`;
        }
        if (name === "status" && typeof data === "object" && data !== null) {
            return `${data.total ?? "?"} files indexed`;
        }
        if (name === "list_dir" && Array.isArray(data)) {
            return `${data.length} entries`;
        }
        if (
            name === "get_file_chunks" &&
            typeof data === "object" &&
            data !== null
        ) {
            const chunks = Array.isArray(data.chunks)
                ? data.chunks.length
                : (data.chunk_count ?? "?");
            return `${chunks} chunks retrieved`;
        }
        if (
            name === "get_file_info" &&
            typeof data === "object" &&
            data !== null
        ) {
            const fileName = data.file_name ?? "file";
            const chunkCount = data.chunk_count ?? "?";
            return `${fileName} (${chunkCount} chunks)`;
        }
    } catch {
        // not JSON — handle text-based results below
    }

    if (name === "read_file") {
        const lines = result.split("\n").length;
        return `${lines} lines read`;
    }

    if (name === "index_update") {
        if (result.toLowerCase().includes("up to date")) {
            return "Index up to date";
        }
        const match = result.match(/processed\s+(\d+)\s+files?/i);
        if (match) {
            return `Processed ${match[1]} files`;
        }
        return "Update complete";
    }

    // Fallback: show size
    if (result.length > 100) {
        const kb = (result.length / 1024).toFixed(1);
        return `${kb}kb of content`;
    }
    return result;
}

function formatResultFull(result: string): string {
    try {
        const data = JSON.parse(result);
        return JSON.stringify(data, null, 2);
    } catch {
        return result;
    }
}

export function ToolCallBlock({ toolCall }: { toolCall: ToolCall }) {
    const [expanded, setExpanded] = useState(false);
    const meta = TOOL_META[toolCall.name] || {
        ...DEFAULT_META,
        label: toolCall.name,
    };
    const Icon = meta.icon;
    const args = formatArgs(toolCall.input);
    const hasResult = toolCall.result !== undefined;
    const canExpand = hasResult && !toolCall.isRunning;

    return (
        <div className="flex flex-col">
            <button
                type="button"
                onClick={() => canExpand && setExpanded(!expanded)}
                disabled={!canExpand}
                className="flex items-center gap-2.5 px-3 py-1.5 rounded-lg bg-muted/50 border border-border text-left w-full min-w-0 hover:bg-muted/80 transition-colors disabled:hover:bg-muted/50 disabled:cursor-default"
            >
                {/* Chevron / running dot */}
                <span className="w-4 flex-shrink-0 flex items-center justify-center">
                    {toolCall.isRunning ? (
                        <span
                            className={`w-2 h-2 rounded-full animate-pulse ${meta.dotColor}`}
                        />
                    ) : canExpand ? (
                        <ChevronRight
                            className={`w-3.5 h-3.5 text-muted-foreground transition-transform duration-200 ${expanded ? "rotate-90" : ""}`}
                        />
                    ) : (
                        <span className="w-3.5" />
                    )}
                </span>

                {/* Icon pill */}
                <span
                    className={`w-6 h-6 rounded-md flex items-center justify-center flex-shrink-0 ${meta.bgClass}`}
                >
                    <Icon className={`w-3.5 h-3.5 ${meta.color}`} />
                </span>

                {/* Tool label */}
                <span className="text-xs font-mono text-foreground font-medium flex-shrink-0">
                    {meta.label}
                </span>

                {/* Args */}
                {args && (
                    <span className="text-xs font-mono text-muted-foreground min-w-0 truncate">
                        {args}
                    </span>
                )}

                {/* Result summary */}
                {hasResult && !toolCall.isRunning && (
                    <span
                        className={`ml-auto text-xs font-mono min-w-0 truncate ${
                            toolCall.isError
                                ? "text-red-500 dark:text-red-400"
                                : "text-emerald-600 dark:text-emerald-400"
                        }`}
                    >
                        {toolCall.isError ? "\u2715" : "\u2713"}{" "}
                        {!expanded &&
                            formatResultSummary(
                                toolCall.name,
                                toolCall.result!,
                            )}
                    </span>
                )}
            </button>

            {/* Expandable result area with CSS grid animation */}
            <div
                className="grid transition-[grid-template-rows] duration-200 ease-out"
                style={{
                    gridTemplateRows: expanded && hasResult ? "1fr" : "0fr",
                }}
            >
                <div className="overflow-hidden">
                    <div className="ml-12 border-t border-border/50 mt-1">
                        <pre className="text-xs font-mono p-3 max-h-80 overflow-y-auto text-muted-foreground whitespace-pre-wrap break-words">
                            {hasResult && formatResultFull(toolCall.result!)}
                        </pre>
                    </div>
                </div>
            </div>
        </div>
    );
}
