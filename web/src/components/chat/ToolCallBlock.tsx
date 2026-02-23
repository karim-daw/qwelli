import { useState } from "react";
import { ChevronRight, ChevronDown } from "lucide-react";
import type { ToolCall } from "@/types/chat";

const TOOL_META: Record<
    string,
    { icon: string; color: string; label: string }
> = {
    search: { icon: "\u2315", color: "#60a5fa", label: "Search" },
    status: { icon: "\u25C8", color: "#fbbf24", label: "Index Status" },
    read_file: { icon: "\u25A1", color: "#38bdf8", label: "Read File" },
    list_dir: { icon: "\u2261", color: "#94a3b8", label: "List Directory" },
    index_update: { icon: "\u21BB", color: "#4ade80", label: "Update Index" },
    get_file_chunks: {
        icon: "\u229E",
        color: "#a78bfa",
        label: "Read Chunks",
    },
    get_file_info: { icon: "\u2139", color: "#22d3ee", label: "File Info" },
    find_files: { icon: "\u229F", color: "#818cf8", label: "Find Files" },
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
        if (name === "find_files" && Array.isArray(data)) {
            return `Found ${data.length} files`;
        }
        if (name === "status" && typeof data === "object" && data !== null) {
            return `${data.total ?? "?"} files indexed`;
        }
        if (name === "list_dir" && Array.isArray(data)) {
            return `${data.length} entries`;
        }
    } catch {
        // not JSON
    }

    if (result.length > 200) {
        return result.slice(0, 197) + "...";
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

function PulsingDot({ color }: { color: string }) {
    return (
        <span
            className="inline-block w-1.5 h-1.5 rounded-full animate-pulse"
            style={{ backgroundColor: color }}
        />
    );
}

export function ToolCallBlock({ toolCall }: { toolCall: ToolCall }) {
    const [expanded, setExpanded] = useState(false);
    const meta = TOOL_META[toolCall.name] || {
        icon: "\u2022",
        color: "#9ca3af",
        label: toolCall.name,
    };
    const args = formatArgs(toolCall.input);
    const hasResult = toolCall.result !== undefined;
    const canExpand = hasResult && !toolCall.isRunning;

    return (
        <div className="flex flex-col gap-1">
            <button
                type="button"
                onClick={() => canExpand && setExpanded(!expanded)}
                disabled={!canExpand}
                className="flex items-center gap-2.5 px-3 py-1.5 rounded-lg bg-muted/50 border border-border text-left w-full hover:bg-muted/80 transition-colors disabled:hover:bg-muted/50 disabled:cursor-default"
            >
                {canExpand ? (
                    expanded ? (
                        <ChevronDown className="w-3.5 h-3.5 text-muted-foreground flex-shrink-0" />
                    ) : (
                        <ChevronRight className="w-3.5 h-3.5 text-muted-foreground flex-shrink-0" />
                    )
                ) : (
                    <span className="w-3.5 flex-shrink-0" />
                )}
                <span
                    className="text-sm flex-shrink-0"
                    style={{ color: meta.color }}
                >
                    {meta.icon}
                </span>
                <span className="text-xs font-mono text-foreground font-medium flex-shrink-0">
                    {meta.label}
                </span>
                {args && (
                    <span className="text-xs font-mono text-muted-foreground truncate">
                        {args}
                    </span>
                )}
                <span className="ml-auto flex items-center gap-2 flex-shrink-0">
                    {toolCall.isRunning && <PulsingDot color={meta.color} />}
                    {hasResult && !toolCall.isRunning && (
                        <span
                            className={`text-xs font-mono ${
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
                </span>
            </button>

            {expanded && hasResult && (
                <div className="ml-9 rounded-lg border border-border bg-muted/30 overflow-hidden">
                    <pre className="text-xs font-mono p-3 overflow-x-auto max-h-80 overflow-y-auto text-muted-foreground whitespace-pre-wrap break-words">
                        {formatResultFull(toolCall.result!)}
                    </pre>
                </div>
            )}
        </div>
    );
}
