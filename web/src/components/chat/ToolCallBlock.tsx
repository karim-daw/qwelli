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
    ExternalLink,
    Folder,
    type LucideIcon,
} from "lucide-react";
import { toast } from "sonner";
import type { ToolCall } from "@/types/chat";
import type { SearchResult } from "@/types/search";
import { ResultCard } from "@/components/search/ResultCard";
import * as filesApi from "@/api/files";

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

interface ToolSearchItem {
    file_path: string;
    file_name: string;
    distance: number;
    preview: string;
    page_numbers?: number[];
}

interface FindFilesItem {
    path: string;
    file_name: string;
    file_type: string;
    size_bytes: number;
    modified_at: string;
}

interface StatusData {
    total: number;
    needs_update?: boolean;
    added?: string[];
    modified?: string[];
    deleted?: string[];
}

interface ListDirItem {
    name: string;
    is_dir: boolean;
    size?: number;
    modified?: string;
}

interface FileChunk {
    content: string;
    page?: number;
}

interface GetFileChunksData {
    file_name: string;
    chunks: FileChunk[];
}

interface GetFileInfoData {
    file_name: string;
    file_type: string;
    chunk_count: number;
    size_bytes: number;
    modified_at?: string;
}

function toolSearchItemToResult(item: ToolSearchItem, index: number): SearchResult {
    return {
        chunkId: `${item.file_path}-${index}`,
        filePath: item.file_path,
        fileName: item.file_name,
        content: item.preview,
        contentType: "text",
        similarity: Math.max(0, 1 - item.distance),
        pageNumbers: item.page_numbers,
    };
}

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

function formatFileSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatDate(iso: string): string {
    try {
        return new Date(iso).toLocaleDateString();
    } catch {
        return iso;
    }
}

interface ToolCallBlockProps {
    toolCall: ToolCall;
    onOpenPDF: (result: SearchResult) => void;
    onViewFullText: (result: SearchResult) => void;
}

export function ToolCallBlock({ toolCall, onOpenPDF, onViewFullText }: ToolCallBlockProps) {
    const [expanded, setExpanded] = useState(false);
    const meta = TOOL_META[toolCall.name] || {
        ...DEFAULT_META,
        label: toolCall.name,
    };
    const Icon = meta.icon;
    const args = formatArgs(toolCall.input);
    const hasResult = toolCall.result !== undefined;
    const canExpand = hasResult && !toolCall.isRunning;

    // Parse result for rich rendering
    let searchResults: SearchResult[] | null = null;
    let findFilesItems: FindFilesItem[] | null = null;
    let statusData: StatusData | null = null;
    let listDirItems: ListDirItem[] | null = null;
    let fileChunksData: GetFileChunksData | null = null;
    let fileInfoData: GetFileInfoData | null = null;
    let readFileContent: string | null = null;
    let indexUpdateMessage: string | null = null;

    if (hasResult && !toolCall.isError) {
        try {
            const data = JSON.parse(toolCall.result!);
            if (toolCall.name === "search" && Array.isArray(data)) {
                searchResults = (data as ToolSearchItem[]).map(toolSearchItemToResult);
            } else if (
                toolCall.name === "find_files" &&
                typeof data === "object" &&
                data !== null &&
                Array.isArray(data.files)
            ) {
                findFilesItems = data.files as FindFilesItem[];
            } else if (toolCall.name === "status" && typeof data === "object" && data !== null) {
                statusData = data as StatusData;
            } else if (toolCall.name === "list_dir" && Array.isArray(data)) {
                listDirItems = data as ListDirItem[];
            } else if (
                toolCall.name === "get_file_chunks" &&
                typeof data === "object" &&
                data !== null
            ) {
                fileChunksData = data as GetFileChunksData;
            } else if (
                toolCall.name === "get_file_info" &&
                typeof data === "object" &&
                data !== null
            ) {
                fileInfoData = data as GetFileInfoData;
            }
        } catch {
            // Not JSON - check for text-based tools
            if (toolCall.name === "read_file") {
                readFileContent = toolCall.result!;
            } else if (toolCall.name === "index_update") {
                indexUpdateMessage = toolCall.result!;
            }
        }
    }

    const useRichRender =
        searchResults !== null ||
        findFilesItems !== null ||
        statusData !== null ||
        listDirItems !== null ||
        fileChunksData !== null ||
        fileInfoData !== null ||
        readFileContent !== null ||
        indexUpdateMessage !== null;

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
                    {useRichRender ? (
                        <div className="pt-2 max-h-96 overflow-y-auto custom-scrollbar">
                            <div className="flex flex-col gap-2">
                                {searchResults && searchResults.map((result, i) => (
                                    <ResultCard
                                        key={result.chunkId}
                                        result={result}
                                        index={i}
                                        onViewFullText={onViewFullText}
                                        onOpenPDF={onOpenPDF}
                                    />
                                ))}
                                {findFilesItems && findFilesItems.map((item, i) => (
                                    <FindFileRow key={`${item.path}-${i}`} item={item} />
                                ))}
                                {statusData && <StatusCard data={statusData} />}
                                {listDirItems && <ListDirCard items={listDirItems} />}
                                {fileChunksData && <FileChunksCard data={fileChunksData} />}
                                {fileInfoData && <FileInfoCard data={fileInfoData} />}
                                {readFileContent && <ReadFileCard content={readFileContent} />}
                                {indexUpdateMessage && <IndexUpdateCard message={indexUpdateMessage} />}
                            </div>
                        </div>
                    ) : (
                        <div className="ml-12 border-t border-border/50 mt-1">
                            <pre className="text-xs font-mono p-3 max-h-80 overflow-y-auto custom-scrollbar text-muted-foreground whitespace-pre-wrap break-words">
                                {hasResult && formatResultFull(toolCall.result!)}
                            </pre>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}

function FindFileRow({ item }: { item: FindFilesItem }) {
    const handleOpen = () => {
        window.open(filesApi.getFileUrl(item.path), "_blank");
    };

    const handleShowInExplorer = async () => {
        try {
            await filesApi.openFileLocation(item.path);
        } catch (error) {
            toast.error("Failed to open file location: " + String(error));
        }
    };

    return (
        <div className="flex items-center gap-3 px-3 py-2 rounded hover:bg-muted/50 text-sm">
            <FileText className="w-4 h-4 flex-shrink-0 text-muted-foreground" />
            <span className="font-mono text-xs text-foreground/80 flex-1 min-w-0 truncate">
                {item.file_name}
            </span>
            <span className="text-xs text-muted-foreground px-1.5 py-0.5 rounded bg-muted flex-shrink-0">
                {item.file_type}
            </span>
            {item.size_bytes > 0 && (
                <span className="text-xs text-muted-foreground flex-shrink-0">
                    {formatFileSize(item.size_bytes)}
                </span>
            )}
            {item.modified_at && (
                <span className="text-xs text-muted-foreground flex-shrink-0">
                    {formatDate(item.modified_at)}
                </span>
            )}
            <button
                type="button"
                onClick={handleOpen}
                className="flex-shrink-0 text-muted-foreground hover:text-foreground transition-colors"
                title="Open file"
            >
                <ExternalLink className="w-3.5 h-3.5" />
            </button>
            <button
                type="button"
                onClick={handleShowInExplorer}
                className="flex-shrink-0 text-muted-foreground hover:text-foreground transition-colors"
                title="Show in explorer"
            >
                <Folder className="w-3.5 h-3.5" />
            </button>
        </div>
    );
}

function StatusCard({ data }: { data: StatusData }) {
    const hasChanges = (data.added?.length ?? 0) + (data.modified?.length ?? 0) + (data.deleted?.length ?? 0) > 0;

    return (
        <div className="rounded-lg border border-border bg-card p-4">
            <div className="grid grid-cols-2 gap-4 mb-3">
                <div>
                    <div className="text-xs text-muted-foreground">Total Files</div>
                    <div className="text-lg font-semibold">{data.total}</div>
                </div>
                <div>
                    <div className="text-xs text-muted-foreground">Status</div>
                    <div className={`text-lg font-semibold ${data.needs_update ? 'text-amber-500' : 'text-emerald-500'}`}>
                        {data.needs_update ? 'Needs Update' : 'Up to date'}
                    </div>
                </div>
            </div>
            {hasChanges && (
                <div className="space-y-2 text-xs">
                    {data.added && data.added.length > 0 && (
                        <div>
                            <span className="text-emerald-500 font-medium">+{data.added.length} added</span>
                        </div>
                    )}
                    {data.modified && data.modified.length > 0 && (
                        <div>
                            <span className="text-amber-500 font-medium">~{data.modified.length} modified</span>
                        </div>
                    )}
                    {data.deleted && data.deleted.length > 0 && (
                        <div>
                            <span className="text-red-500 font-medium">-{data.deleted.length} deleted</span>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}

function ListDirCard({ items }: { items: ListDirItem[] }) {
    return (
        <div className="space-y-1">
            {items.map((item, i) => (
                <div key={i} className="flex items-center gap-2 px-3 py-1.5 rounded hover:bg-muted/50 text-sm">
                    {item.is_dir ? (
                        <FolderOpen className="w-4 h-4 text-blue-500 flex-shrink-0" />
                    ) : (
                        <FileText className="w-4 h-4 text-muted-foreground flex-shrink-0" />
                    )}
                    <span className="font-mono text-xs flex-1 min-w-0 truncate">
                        {item.name}
                    </span>
                    {!item.is_dir && item.size && (
                        <span className="text-xs text-muted-foreground">
                            {formatFileSize(item.size)}
                        </span>
                    )}
                </div>
            ))}
        </div>
    );
}

function FileChunksCard({ data }: { data: GetFileChunksData }) {
    return (
        <div className="rounded-lg border border-border bg-card">
            <div className="px-4 py-2 border-b border-border">
                <div className="text-sm font-medium">{data.file_name}</div>
                <div className="text-xs text-muted-foreground">{data.chunks.length} chunks</div>
            </div>
            <div className="divide-y divide-border max-h-80 overflow-y-auto custom-scrollbar">
                {data.chunks.map((chunk, i) => (
                    <div key={i} className="p-3">
                        {chunk.page !== undefined && (
                            <div className="text-xs text-blue-500 mb-1">Page {chunk.page}</div>
                        )}
                        <div className="text-xs text-foreground/80 whitespace-pre-wrap break-words">
                            {chunk.content.length > 300 ? chunk.content.slice(0, 300) + "..." : chunk.content}
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}

function FileInfoCard({ data }: { data: GetFileInfoData }) {
    return (
        <div className="rounded-lg border border-border bg-card p-4">
            <div className="text-sm font-medium mb-3">{data.file_name}</div>
            <div className="grid grid-cols-2 gap-3 text-xs">
                <div>
                    <div className="text-muted-foreground">Type</div>
                    <div className="font-medium">{data.file_type}</div>
                </div>
                <div>
                    <div className="text-muted-foreground">Chunks</div>
                    <div className="font-medium">{data.chunk_count}</div>
                </div>
                <div>
                    <div className="text-muted-foreground">Size</div>
                    <div className="font-medium">{formatFileSize(data.size_bytes)}</div>
                </div>
                {data.modified_at && (
                    <div>
                        <div className="text-muted-foreground">Modified</div>
                        <div className="font-medium">{formatDate(data.modified_at)}</div>
                    </div>
                )}
            </div>
        </div>
    );
}

function ReadFileCard({ content }: { content: string }) {
    return (
        <div className="rounded-lg border border-border bg-card">
            <pre className="text-xs font-mono p-4 max-h-80 overflow-y-auto custom-scrollbar whitespace-pre-wrap break-words">
                {content}
            </pre>
        </div>
    );
}

function IndexUpdateCard({ message }: { message: string }) {
    const isUpToDate = message.toLowerCase().includes("up to date");

    return (
        <div className={`rounded-lg border p-4 ${
            isUpToDate
                ? 'border-emerald-500/50 bg-emerald-500/10'
                : 'border-blue-500/50 bg-blue-500/10'
        }`}>
            <div className="text-sm">{message}</div>
        </div>
    );
}
