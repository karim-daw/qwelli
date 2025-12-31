import { useState, useEffect } from "react";
import {
    Search,
    Folder,
    Plus,
    ExternalLink,
    Image as ImageIcon,
    X,
    Settings2,
    RefreshCw,
    AlertCircle,
    CheckCircle,
    Menu,
    ChevronRight,
    FileText,
    Clock,
    HardDrive,
    Activity,
} from "lucide-react";

interface Index {
    name: string;
    path: string;
    documentCount: number;
    lastModified: string;
}

interface IndexStatus {
    toAdd: FileStatus[];
    toUpdate: FileStatus[];
    toDelete: FileStatus[];
    total: number;
    upToDate: number;
}

interface FileStatus {
    path: string;
    fileType: string;
    size: number;
    modifiedAt: string;
    reason: string;
}

interface SearchResult {
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
    metadata?: Record<string, any>;
}

interface IndexProgress {
    current: number;
    total: number;
    file: string;
    indexPath: string;
}

function App() {
    const [indexes, setIndexes] = useState<Index[]>([]);
    const [selectedIndex, setSelectedIndex] = useState<string>("");
    const [query, setQuery] = useState("");
    const [results, setResults] = useState<SearchResult[]>([]);
    const [loading, setLoading] = useState(false);
    const [showNewIndexDialog, setShowNewIndexDialog] = useState(false);
    const [showSettings, setShowSettings] = useState(false);
    const [showSidebar, setShowSidebar] = useState(true);
    const [viewMode, setViewMode] = useState<"search" | "status">("search");
    const [indexStatus, setIndexStatus] = useState<IndexStatus | null>(null);
    const [loadingStatus, setLoadingStatus] = useState(false);
    const [newIndexPath, setNewIndexPath] = useState("");
    const [indexing, setIndexing] = useState(false);
    const [indexProgress, setIndexProgress] = useState<IndexProgress | null>(
        null,
    );
    const [indexingComplete, setIndexingComplete] = useState(false);
    const [selectedResult, setSelectedResult] = useState<SearchResult | null>(
        null,
    );
    const [showFullTextModal, setShowFullTextModal] = useState(false);
    const [cacheStatus, setCacheStatus] = useState<string | null>(null);

    // Search settings
    const [strategy, setStrategy] = useState("semantic");
    const [topK, setTopK] = useState(10);
    const [contentFilter, setContentFilter] = useState("all");

    useEffect(() => {
        fetchIndexes();
    }, []);

    const fetchIndexes = async () => {
        try {
            const response = await fetch("/api/indexes");
            const data = await response.json();
            setIndexes(data.indexes || []);
            if (data.indexes?.length > 0 && !selectedIndex) {
                setSelectedIndex(data.indexes[0].path);
            }
        } catch (error) {
            console.error("Failed to fetch indexes:", error);
        }
    };

    const fetchIndexStatus = async (indexPath: string) => {
        setLoadingStatus(true);
        try {
            const response = await fetch(
                "/api/index/status?path=" + encodeURIComponent(indexPath),
            );
            const data = await response.json();
            setIndexStatus(data);
        } catch (error) {
            console.error("Failed to fetch status:", error);
        } finally {
            setLoadingStatus(false);
        }
    };

    const handleSearch = async (e?: React.FormEvent) => {
        e?.preventDefault();
        if (!query.trim() || !selectedIndex) return;

        setLoading(true);
        try {
            const params = new URLSearchParams({
                q: query,
                index: selectedIndex,
                strategy: strategy,
                top: topK.toString(),
            });

            if (contentFilter !== "all") {
                params.append("content_type", contentFilter);
            }

            const response = await fetch("/api/search?" + params.toString());
            const data = await response.json();

            // Check cache status from response headers
            const cacheHeader = response.headers.get("X-Cache-Status");
            setCacheStatus(cacheHeader);

            setResults(data.results || []);
        } catch (error) {
            console.error("Search failed:", error);
        } finally {
            setLoading(false);
        }
    };

    const convertWindowsPathToWSL = (path: string): string => {
        const windowsPathRegex = /^([A-Za-z]):[\/\\]/;
        const match = path.match(windowsPathRegex);

        if (match) {
            const drive = match[1].toLowerCase();
            let converted = path.replace(
                windowsPathRegex,
                "/mnt/" + drive + "/",
            );
            converted = converted.replace(/\\/g, "/");
            return converted;
        }

        return path;
    };

    const handleCreateIndex = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!newIndexPath.trim()) return;

        setIndexing(true);
        const convertedPath = convertWindowsPathToWSL(newIndexPath.trim());

        try {
            // Connect to SSE progress stream
            const eventSource = new EventSource(
                "/api/index/progress?path=" + encodeURIComponent(convertedPath),
            );

            eventSource.onmessage = (event) => {
                const data = JSON.parse(event.data);

                if (data.type === "progress") {
                    setIndexProgress({
                        current: data.current,
                        total: data.total,
                        file: data.file,
                        indexPath: convertedPath,
                    });
                } else if (data.type === "complete") {
                    eventSource.close();
                    setIndexingComplete(true);
                    setIndexing(false);
                    fetchIndexes();
                } else if (data.type === "cancelled") {
                    eventSource.close();
                    setIndexProgress(null);
                    setIndexing(false);
                    setIndexingComplete(false);
                    setShowNewIndexDialog(false);
                    setNewIndexPath("");
                } else if (data.type === "error") {
                    eventSource.close();
                    setIndexProgress(null);
                    setIndexing(false);
                    setIndexingComplete(false);
                    alert("Indexing error: " + data.message);
                }
            };

            eventSource.onerror = () => {
                eventSource.close();
                setIndexProgress(null);
                setIndexing(false);
            };

            // Start indexing
            const response = await fetch("/api/index/create", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ folderPath: convertedPath }),
            });

            if (!response.ok) {
                const data = await response.json();
                eventSource.close();
                setIndexProgress(null);
                setIndexing(false);
                alert("Error: " + data.error);
            }
        } catch (error) {
            setIndexProgress(null);
            setIndexing(false);
            alert("Failed to create index: " + error);
        }
    };

    const formatFileSize = (bytes?: number): string => {
        if (!bytes) return "Unknown";
        if (bytes < 1024) return bytes + " B";
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
        if (bytes < 1024 * 1024 * 1024)
            return (bytes / (1024 * 1024)).toFixed(1) + " MB";
        return (bytes / (1024 * 1024 * 1024)).toFixed(1) + " GB";
    };

    const handleCancelIndexing = async () => {
        if (!indexProgress) return;

        try {
            const response = await fetch("/api/index/cancel", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ indexPath: indexProgress.indexPath }),
            });

            if (!response.ok) {
                const data = await response.json();
                alert("Failed to cancel: " + data.error);
            }
        } catch (error) {
            alert("Failed to cancel indexing: " + error);
        }
    };

    const handleDeleteIndex = async (indexPath: string) => {
        if (
            !confirm(
                "Are you sure you want to delete this index? This cannot be undone.",
            )
        ) {
            return;
        }

        try {
            const response = await fetch("/api/index/delete", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ indexPath }),
            });

            if (response.ok) {
                // Refresh index list
                fetchIndexes();
                // Clear selection if deleted index was selected
                if (selectedIndex === indexPath) {
                    setSelectedIndex("");
                    setResults([]);
                }
            } else {
                const data = await response.json();
                alert("Failed to delete index: " + data.error);
            }
        } catch (error) {
            alert("Failed to delete index: " + error);
        }
    };

    const handleOpenFolder = async (folderPath: string) => {
        try {
            const response = await fetch("/api/open-folder", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ path: folderPath }),
            });

            if (!response.ok) {
                const data = await response.json();
                alert("Failed to open folder: " + data.error);
            }
        } catch (error) {
            alert("Failed to open folder: " + error);
        }
    };

    const handleCloseProgressModal = () => {
        setIndexProgress(null);
        setIndexingComplete(false);
        setShowNewIndexDialog(false);
        setNewIndexPath("");
    };

    const currentIndex = indexes.find((i) => i.path === selectedIndex);

    return (
        <div className="flex h-screen bg-black text-white overflow-hidden">
            {/* Sidebar */}
            <div
                className={`${showSidebar ? "w-64" : "w-0"} flex-shrink-0 border-r border-white/10 transition-all duration-300 overflow-hidden`}
            >
                <div className="h-full flex flex-col">
                    {/* Sidebar Header */}
                    <div className="p-4 border-b border-white/10">
                        <div className="flex items-center justify-between mb-3">
                            <h2 className="text-sm font-medium">Indexes</h2>
                            <button
                                onClick={() => setShowNewIndexDialog(true)}
                                className="p-1.5 hover:bg-white/10 rounded-md transition-colors"
                            >
                                <Plus className="w-4 h-4" />
                            </button>
                        </div>
                    </div>

                    {/* Index List */}
                    <div className="flex-1 overflow-y-auto">
                        {indexes?.map((index) => (
                            <button
                                key={index.path}
                                onClick={() => {
                                    setSelectedIndex(index.path);
                                    setViewMode("search");
                                    setResults([]);
                                }}
                                className={`w-full text-left px-4 py-3 border-b border-white/5 hover:bg-white/5 transition-colors group ${
                                    selectedIndex === index.path
                                        ? "bg-white/10"
                                        : ""
                                }`}
                            >
                                <div className="flex items-start justify-between gap-2">
                                    <div className="flex-1 min-w-0">
                                        <div className="flex items-center gap-2 mb-1">
                                            <button
                                                onClick={(e) => {
                                                    e.stopPropagation();
                                                    handleOpenFolder(
                                                        index.path,
                                                    );
                                                }}
                                                className="p-0.5 hover:bg-white/10 rounded transition-colors"
                                                title="Open folder in file explorer"
                                            >
                                                <Folder className="w-3.5 h-3.5 flex-shrink-0 text-gray-400 hover:text-white" />
                                            </button>
                                            <span className="text-sm truncate">
                                                {index.name}
                                            </span>
                                        </div>
                                        <div className="text-xs text-gray-500">
                                            {index.documentCount} docs
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-1">
                                        <button
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                setSelectedIndex(index.path);
                                                setViewMode("status");
                                                setIndexStatus(null);
                                                fetchIndexStatus(index.path);
                                            }}
                                            className="opacity-0 group-hover:opacity-100 p-1 hover:bg-white/10 rounded transition-opacity"
                                            title="View status & changes"
                                        >
                                            <Activity className="w-3.5 h-3.5" />
                                        </button>
                                        <button
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                handleDeleteIndex(index.path);
                                            }}
                                            className="opacity-0 group-hover:opacity-100 p-1 hover:bg-red-500/20 rounded transition-opacity text-red-400"
                                            title="Delete index"
                                        >
                                            <X className="w-3.5 h-3.5" />
                                        </button>
                                    </div>
                                </div>
                            </button>
                        ))}
                    </div>
                </div>
            </div>

            {/* Main Content */}
            <div className="flex-1 flex flex-col overflow-hidden">
                {/* Top Bar */}
                <div className="flex-shrink-0 border-b border-white/10 bg-black/80 backdrop-blur-xl">
                    <div className="px-6 py-4">
                        <div className="flex items-center gap-4 mb-4">
                            <button
                                onClick={() => setShowSidebar(!showSidebar)}
                                className="p-1.5 hover:bg-white/10 rounded-md transition-colors"
                            >
                                <Menu className="w-5 h-5" />
                            </button>
                            <h1 className="text-lg font-medium">Qwelli</h1>
                            {currentIndex && (
                                <div className="flex items-center gap-2 text-sm text-gray-400">
                                    <ChevronRight className="w-4 h-4" />
                                    <span>{currentIndex.name}</span>
                                </div>
                            )}
                        </div>

                        {viewMode === "search" && (
                            <form onSubmit={handleSearch} className="relative">
                                <div className="relative">
                                    <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
                                    <input
                                        type="text"
                                        value={query}
                                        onChange={(e) =>
                                            setQuery(e.target.value)
                                        }
                                        placeholder="Search knowledge base..."
                                        disabled={!selectedIndex}
                                        className="w-full pl-12 pr-12 py-3.5 text-base bg-white/5 border border-white/10 rounded-xl focus:border-white/30 outline-none transition-colors disabled:opacity-50 disabled:cursor-not-allowed placeholder:text-gray-500"
                                    />
                                    <button
                                        type="button"
                                        onClick={() =>
                                            setShowSettings(!showSettings)
                                        }
                                        className="absolute right-3 top-1/2 -translate-y-1/2 p-1.5 text-gray-400 hover:text-white transition-colors rounded-md hover:bg-white/10"
                                    >
                                        <Settings2 className="w-4 h-4" />
                                    </button>
                                </div>

                                {showSettings && (
                                    <div className="absolute top-full mt-2 right-0 bg-black border border-white/10 rounded-lg p-4 shadow-2xl w-80 z-50">
                                        <div className="space-y-4">
                                            <div>
                                                <label className="block text-xs text-gray-400 mb-2">
                                                    Search Strategy
                                                </label>
                                                <div className="grid grid-cols-3 gap-2">
                                                    {[
                                                        "semantic",
                                                        "keyword",
                                                        "hybrid",
                                                    ].map((s) => (
                                                        <button
                                                            key={s}
                                                            type="button"
                                                            onClick={() =>
                                                                setStrategy(s)
                                                            }
                                                            className={
                                                                "px-3 py-2 text-xs rounded-md transition-colors " +
                                                                (strategy === s
                                                                    ? "bg-white text-black"
                                                                    : "bg-white/5 hover:bg-white/10")
                                                            }
                                                        >
                                                            {s
                                                                .charAt(0)
                                                                .toUpperCase() +
                                                                s.slice(1)}
                                                        </button>
                                                    ))}
                                                </div>
                                            </div>

                                            <div>
                                                <label className="block text-xs text-gray-400 mb-2">
                                                    Content Type
                                                </label>
                                                <div className="grid grid-cols-3 gap-2">
                                                    {[
                                                        {
                                                            value: "all",
                                                            label: "All",
                                                        },
                                                        {
                                                            value: "text",
                                                            label: "Text",
                                                        },
                                                        {
                                                            value: "image",
                                                            label: "Images",
                                                        },
                                                    ].map((c) => (
                                                        <button
                                                            key={c.value}
                                                            type="button"
                                                            onClick={() =>
                                                                setContentFilter(
                                                                    c.value,
                                                                )
                                                            }
                                                            className={
                                                                "px-3 py-2 text-xs rounded-md transition-colors " +
                                                                (contentFilter ===
                                                                c.value
                                                                    ? "bg-white text-black"
                                                                    : "bg-white/5 hover:bg-white/10")
                                                            }
                                                        >
                                                            {c.label}
                                                        </button>
                                                    ))}
                                                </div>
                                            </div>

                                            <div>
                                                <label className="block text-xs text-gray-400 mb-2">
                                                    Results: {topK}
                                                </label>
                                                <input
                                                    type="range"
                                                    min="5"
                                                    max="50"
                                                    step="5"
                                                    value={topK}
                                                    onChange={(e) =>
                                                        setTopK(
                                                            parseInt(
                                                                e.target.value,
                                                            ),
                                                        )
                                                    }
                                                    className="w-full"
                                                />
                                            </div>

                                            <button
                                                type="button"
                                                onClick={() => {
                                                    setShowSettings(false);
                                                    handleSearch();
                                                }}
                                                className="w-full px-4 py-2 text-sm bg-white text-black rounded-md hover:opacity-90"
                                            >
                                                Apply & Search
                                            </button>
                                        </div>
                                    </div>
                                )}
                            </form>
                        )}
                    </div>
                </div>

                {/* Content Area */}
                <div className="flex-1 overflow-y-auto">
                    <div className="max-w-5xl mx-auto px-6 py-6">
                        {viewMode === "status" ? (
                            loadingStatus ? (
                                <div className="text-center text-gray-400 py-20">
                                    Loading status...
                                </div>
                            ) : indexStatus ? (
                                <div className="space-y-6">
                                    {/* Action Bar */}
                                    <div className="flex items-center justify-between">
                                        <h2 className="text-lg font-medium">
                                            Index Status
                                        </h2>
                                        {((indexStatus.toAdd?.length || 0) >
                                            0 ||
                                            (indexStatus.toUpdate?.length ||
                                                0) > 0 ||
                                            (indexStatus.toDelete?.length ||
                                                0) > 0) && (
                                            <button
                                                onClick={async () => {
                                                    if (
                                                        confirm(
                                                            "Sync all changes? This will update the index with new, modified, and deleted files.",
                                                        )
                                                    ) {
                                                        try {
                                                            const response =
                                                                await fetch(
                                                                    "/api/index/update",
                                                                    {
                                                                        method: "POST",
                                                                        headers:
                                                                            {
                                                                                "Content-Type":
                                                                                    "application/json",
                                                                            },
                                                                        body: JSON.stringify(
                                                                            {
                                                                                indexPath:
                                                                                    selectedIndex,
                                                                            },
                                                                        ),
                                                                    },
                                                                );
                                                            if (response.ok) {
                                                                alert(
                                                                    "Index sync started in background. Refresh status in a few moments.",
                                                                );
                                                                setTimeout(
                                                                    () =>
                                                                        fetchIndexStatus(
                                                                            selectedIndex,
                                                                        ),
                                                                    3000,
                                                                );
                                                            }
                                                        } catch (error) {
                                                            alert(
                                                                "Failed to sync: " +
                                                                    error,
                                                            );
                                                        }
                                                    }
                                                }}
                                                className="flex items-center gap-2 px-4 py-2 bg-white text-black rounded-md hover:opacity-90 text-sm font-medium"
                                            >
                                                <RefreshCw className="w-4 h-4" />
                                                Sync Changes
                                            </button>
                                        )}
                                    </div>

                                    {/* Status Summary */}
                                    <div className="grid grid-cols-4 gap-4">
                                        <div className="bg-white/5 rounded-lg p-4 border border-white/10">
                                            <div className="text-2xl font-semibold mb-1">
                                                {indexStatus.total}
                                            </div>
                                            <div className="text-xs text-gray-400">
                                                Total Files
                                            </div>
                                        </div>
                                        <div className="bg-green-500/10 rounded-lg p-4 border border-green-500/20">
                                            <div className="text-2xl font-semibold text-green-400 mb-1">
                                                {indexStatus.upToDate}
                                            </div>
                                            <div className="text-xs text-gray-400">
                                                Up to Date
                                            </div>
                                        </div>
                                        <div className="bg-yellow-500/10 rounded-lg p-4 border border-yellow-500/20">
                                            <div className="text-2xl font-semibold text-yellow-400 mb-1">
                                                {indexStatus.toUpdate?.length ||
                                                    0}
                                            </div>
                                            <div className="text-xs text-gray-400">
                                                Modified
                                            </div>
                                        </div>
                                        <div className="bg-blue-500/10 rounded-lg p-4 border border-blue-500/20">
                                            <div className="text-2xl font-semibold text-blue-400 mb-1">
                                                {indexStatus.toAdd?.length || 0}
                                            </div>
                                            <div className="text-xs text-gray-400">
                                                New Files
                                            </div>
                                        </div>
                                    </div>

                                    {/* Change Details */}
                                    {indexStatus.toAdd?.length > 0 && (
                                        <div>
                                            <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
                                                <Plus className="w-4 h-4 text-blue-400" />
                                                New Files (
                                                {indexStatus.toAdd.length})
                                            </h3>
                                            <div className="space-y-2">
                                                {indexStatus.toAdd
                                                    .slice(0, 10)
                                                    .map((file, i) => (
                                                        <div
                                                            key={i}
                                                            className="text-xs bg-white/5 rounded px-3 py-2 border border-white/10"
                                                        >
                                                            <div className="font-mono truncate">
                                                                {file.path}
                                                            </div>
                                                        </div>
                                                    ))}
                                            </div>
                                        </div>
                                    )}

                                    {indexStatus.toUpdate?.length > 0 && (
                                        <div>
                                            <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
                                                <AlertCircle className="w-4 h-4 text-yellow-400" />
                                                Modified Files (
                                                {indexStatus.toUpdate.length})
                                            </h3>
                                            <div className="space-y-2">
                                                {indexStatus.toUpdate
                                                    .slice(0, 10)
                                                    .map((file, i) => (
                                                        <div
                                                            key={i}
                                                            className="text-xs bg-white/5 rounded px-3 py-2 border border-white/10"
                                                        >
                                                            <div className="font-mono truncate">
                                                                {file.path}
                                                            </div>
                                                        </div>
                                                    ))}
                                            </div>
                                        </div>
                                    )}

                                    {indexStatus.toDelete?.length > 0 && (
                                        <div>
                                            <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
                                                <X className="w-4 h-4 text-red-400" />
                                                Deleted Files (
                                                {indexStatus.toDelete.length})
                                            </h3>
                                            <div className="space-y-2">
                                                {indexStatus.toDelete
                                                    .slice(0, 10)
                                                    .map((file, i) => (
                                                        <div
                                                            key={i}
                                                            className="text-xs bg-white/5 rounded px-3 py-2 border border-white/10"
                                                        >
                                                            <div className="font-mono truncate">
                                                                {file.path}
                                                            </div>
                                                        </div>
                                                    ))}
                                            </div>
                                        </div>
                                    )}

                                    {(indexStatus.toAdd?.length || 0) === 0 &&
                                        (indexStatus.toUpdate?.length || 0) ===
                                            0 &&
                                        (indexStatus.toDelete?.length || 0) ===
                                            0 && (
                                            <div className="text-center py-20">
                                                <CheckCircle className="w-12 h-12 text-green-400 mx-auto mb-3" />
                                                <p className="text-gray-400">
                                                    Index is up to date!
                                                </p>
                                            </div>
                                        )}
                                </div>
                            ) : (
                                <div className="text-center text-gray-500 py-20">
                                    Select an index to view status
                                </div>
                            )
                        ) : !selectedIndex ? (
                            <div className="text-center text-gray-500 py-20">
                                <Folder className="w-12 h-12 mx-auto mb-3 opacity-50" />
                                <p>Select an index from the sidebar to start</p>
                            </div>
                        ) : loading ? (
                            <div className="text-center text-gray-400 py-20">
                                Searching...
                            </div>
                        ) : results?.length > 0 ? (
                            <div className="space-y-4">
                                {/* Results header with cache indicator */}
                                <div className="flex items-center justify-between pb-2 border-b border-white/10">
                                    <div className="text-sm text-gray-400">
                                        Found {results.length} result
                                        {results.length !== 1 ? "s" : ""}
                                    </div>
                                    {cacheStatus === "HIT" && (
                                        <div className="flex items-center gap-1.5 text-xs text-green-400 bg-green-500/10 px-2 py-1 rounded border border-green-500/20">
                                            <svg
                                                className="w-3 h-3"
                                                fill="none"
                                                stroke="currentColor"
                                                viewBox="0 0 24 24"
                                            >
                                                <path
                                                    strokeLinecap="round"
                                                    strokeLinejoin="round"
                                                    strokeWidth={2}
                                                    d="M13 10V3L4 14h7v7l9-11h-7z"
                                                />
                                            </svg>
                                            Cached
                                        </div>
                                    )}
                                </div>
                                {results?.map((result, index) => (
                                    <div
                                        key={result.chunkId + index}
                                        className="border border-white/10 rounded-lg p-5 hover:border-white/20 transition-colors bg-white/[0.02]"
                                    >
                                        {/* Header */}
                                        <div className="flex items-start justify-between gap-4 mb-3">
                                            <div className="flex items-center gap-2 text-sm text-gray-400 flex-1 min-w-0">
                                                {result.contentType ===
                                                "image" ? (
                                                    <ImageIcon className="w-4 h-4 flex-shrink-0" />
                                                ) : (
                                                    <FileText className="w-4 h-4 flex-shrink-0" />
                                                )}
                                                <span className="font-mono text-xs truncate font-medium text-gray-300">
                                                    {result.fileName ||
                                                        result.filePath}
                                                </span>
                                                {result.pageNumbers &&
                                                    result.pageNumbers.length >
                                                        0 && (
                                                        <span className="text-xs flex-shrink-0 bg-blue-500/10 px-2 py-0.5 rounded border border-blue-500/20 text-blue-400">
                                                            Page{" "}
                                                            {
                                                                result
                                                                    .pageNumbers[0]
                                                            }
                                                        </span>
                                                    )}
                                                {result.chunkIndex !==
                                                    undefined &&
                                                    result.totalChunks && (
                                                        <span className="text-xs flex-shrink-0 text-gray-500">
                                                            Chunk{" "}
                                                            {result.chunkIndex +
                                                                1}
                                                            /
                                                            {result.totalChunks}
                                                        </span>
                                                    )}
                                            </div>
                                            <div className="flex items-center gap-2 flex-shrink-0">
                                                <span className="text-xs font-medium px-2 py-1 rounded bg-white/10">
                                                    {(
                                                        (1 -
                                                            result.similarity) *
                                                        100
                                                    ).toFixed(0)}
                                                    %
                                                </span>
                                                <button
                                                    onClick={() => {
                                                        setSelectedResult(
                                                            result,
                                                        );
                                                        setShowFullTextModal(
                                                            true,
                                                        );
                                                    }}
                                                    className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
                                                    title="View full text"
                                                >
                                                    <FileText className="w-4 h-4" />
                                                </button>
                                                <button
                                                    onClick={() =>
                                                        window.open(
                                                            "/api/file?path=" +
                                                                encodeURIComponent(
                                                                    result.filePath,
                                                                ),
                                                            "_blank",
                                                        )
                                                    }
                                                    className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
                                                    title="Open file"
                                                >
                                                    <ExternalLink className="w-4 h-4" />
                                                </button>
                                            </div>
                                        </div>

                                        {/* Content */}
                                        {result.contentType === "image" &&
                                        result.imageData ? (
                                            <div className="mb-3">
                                                <img
                                                    src={`data:image/jpeg;base64,${result.imageData}`}
                                                    alt={result.fileName}
                                                    className="max-w-full h-auto max-h-64 rounded-lg border border-white/10"
                                                />
                                                <p className="text-xs text-gray-500 mt-2 italic">
                                                    {result.content}
                                                </p>
                                            </div>
                                        ) : (
                                            <p className="text-sm leading-relaxed text-gray-300 mb-3">
                                                {result.content}
                                            </p>
                                        )}

                                        {/* Metadata Footer */}
                                        <div className="flex items-center gap-4 text-xs text-gray-500 pt-3 border-t border-white/5">
                                            {result.fileSize && (
                                                <div className="flex items-center gap-1.5">
                                                    <HardDrive className="w-3 h-3" />
                                                    <span>
                                                        {formatFileSize(
                                                            result.fileSize,
                                                        )}
                                                    </span>
                                                </div>
                                            )}
                                            {result.modifiedDate && (
                                                <div className="flex items-center gap-1.5">
                                                    <Clock className="w-3 h-3" />
                                                    <span>
                                                        {result.modifiedDate}
                                                    </span>
                                                </div>
                                            )}
                                            <div className="flex items-center gap-1.5">
                                                <Folder className="w-3 h-3" />
                                                <span className="font-mono truncate max-w-md">
                                                    {result.filePath}
                                                </span>
                                            </div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        ) : query ? (
                            <div className="text-center text-gray-500 py-20">
                                <p>No results found for "{query}"</p>
                            </div>
                        ) : null}
                    </div>
                </div>
            </div>

            {/* New Index Dialog */}
            {showNewIndexDialog && (
                <div className="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50">
                    <div className="bg-black border border-white/20 rounded-lg max-w-md w-full mx-4 p-6 shadow-2xl">
                        <div className="flex items-center justify-between mb-4">
                            <h2 className="text-lg font-medium">
                                Index a new folder
                            </h2>
                            <button
                                onClick={() => setShowNewIndexDialog(false)}
                                className="text-gray-400 hover:text-white"
                                disabled={indexing}
                            >
                                <X className="w-5 h-5" />
                            </button>
                        </div>
                        <form onSubmit={handleCreateIndex}>
                            <div className="mb-4">
                                <label className="block text-sm text-gray-400 mb-2">
                                    Folder path (absolute path)
                                </label>
                                <div className="flex gap-2">
                                    <input
                                        type="text"
                                        value={newIndexPath}
                                        onChange={(e) =>
                                            setNewIndexPath(e.target.value)
                                        }
                                        placeholder="C:\Users\karim\Documents or /home/user/Documents"
                                        className="flex-1 px-3 py-2 bg-white/5 border border-white/10 rounded-md focus:border-white/30 outline-none"
                                        disabled={indexing}
                                    />
                                    <label className="px-4 py-2 bg-white/10 hover:bg-white/20 border border-white/10 rounded-md cursor-pointer transition-colors flex items-center gap-2">
                                        <Folder className="w-4 h-4" />
                                        Browse
                                        <input
                                            type="file"
                                            // @ts-ignore - webkitdirectory is not in types but exists
                                            webkitdirectory=""
                                            directory=""
                                            multiple
                                            className="hidden"
                                            onChange={(e) => {
                                                const files = e.target.files;
                                                if (files && files.length > 0) {
                                                    // Get the path from the first file
                                                    const firstFile = files[0];
                                                    // Extract directory path from webkitRelativePath
                                                    const relativePath = (
                                                        firstFile as any
                                                    ).webkitRelativePath;
                                                    if (relativePath) {
                                                        const parts =
                                                            relativePath.split(
                                                                "/",
                                                            );
                                                        const folderName =
                                                            parts[0];
                                                        // Get current working directory hint
                                                        setNewIndexPath(
                                                            folderName,
                                                        );
                                                    }
                                                }
                                            }}
                                            disabled={indexing}
                                        />
                                    </label>
                                </div>
                                <p className="text-xs text-gray-500 mt-2">
                                    Paste full path or use Browse (then complete
                                    the path manually)
                                </p>
                                {newIndexPath &&
                                    convertWindowsPathToWSL(newIndexPath) !==
                                        newIndexPath && (
                                        <p className="text-xs text-gray-500 mt-1">
                                            →{" "}
                                            {convertWindowsPathToWSL(
                                                newIndexPath,
                                            )}
                                        </p>
                                    )}
                            </div>
                            <div className="flex gap-2">
                                <button
                                    type="button"
                                    onClick={() => setShowNewIndexDialog(false)}
                                    className="flex-1 px-4 py-2 text-sm border border-white/10 rounded-md hover:bg-white/5"
                                    disabled={indexing}
                                >
                                    Cancel
                                </button>
                                <button
                                    type="submit"
                                    className="flex-1 px-4 py-2 text-sm bg-white text-black rounded-md hover:opacity-90 disabled:opacity-50"
                                    disabled={indexing || !newIndexPath.trim()}
                                >
                                    {indexing
                                        ? "Indexing..."
                                        : "Start Indexing"}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            )}

            {/* Live Indexing Progress Modal */}
            {indexProgress && (
                <div className="fixed inset-0 bg-black/90 backdrop-blur-sm flex items-center justify-center z-50">
                    <div className="bg-black border border-white/20 rounded-lg max-w-2xl w-full mx-4 p-8 shadow-2xl">
                        <div className="mb-6">
                            <h2 className="text-xl font-medium mb-2">
                                {indexingComplete
                                    ? "Indexing Complete!"
                                    : "Indexing in progress"}
                            </h2>
                            <p className="text-sm text-gray-400 font-mono truncate">
                                {indexProgress.indexPath}
                            </p>
                        </div>

                        {/* Progress Bar */}
                        {!indexingComplete && (
                            <>
                                <div className="mb-6">
                                    <div className="flex items-center justify-between text-sm mb-2">
                                        <span className="text-gray-400">
                                            Processing files...
                                        </span>
                                        <span className="font-medium">
                                            {indexProgress.current} /{" "}
                                            {indexProgress.total}
                                        </span>
                                    </div>
                                    <div className="w-full h-2 bg-white/10 rounded-full overflow-hidden">
                                        <div
                                            className="h-full bg-white transition-all duration-300 ease-out"
                                            style={{
                                                width: `${indexProgress.total > 0 ? (indexProgress.current / indexProgress.total) * 100 : 0}%`,
                                            }}
                                        />
                                    </div>
                                    <div className="mt-2 text-xs text-gray-500">
                                        {indexProgress.total > 0
                                            ? (
                                                  (indexProgress.current /
                                                      indexProgress.total) *
                                                  100
                                              ).toFixed(1)
                                            : "0.0"}
                                        % complete
                                    </div>
                                </div>

                                {/* Current File */}
                                <div className="bg-white/5 rounded-lg p-4 border border-white/10 mb-6">
                                    <div className="flex items-start gap-3">
                                        <div className="mt-1">
                                            <div className="w-2 h-2 bg-white rounded-full animate-pulse" />
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <div className="text-xs text-gray-400 mb-1">
                                                Current file
                                            </div>
                                            <div className="font-mono text-sm truncate">
                                                {indexProgress.file}
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </>
                        )}

                        {/* Completion Message */}
                        {indexingComplete && (
                            <div className="bg-green-500/10 border border-green-500/20 rounded-lg p-6 mb-6">
                                <div className="flex items-center gap-3 mb-2">
                                    <CheckCircle className="w-6 h-6 text-green-500" />
                                    <span className="text-lg font-medium">
                                        Successfully indexed{" "}
                                        {indexProgress.total} files
                                    </span>
                                </div>
                                <p className="text-sm text-gray-400">
                                    Your files are now searchable
                                </p>
                            </div>
                        )}

                        {/* Action Buttons */}
                        <div className="flex gap-3">
                            {!indexingComplete ? (
                                <>
                                    <button
                                        onClick={handleCancelIndexing}
                                        className="flex-1 px-4 py-2.5 bg-red-500/10 hover:bg-red-500/20 border border-red-500/20 rounded-lg transition-colors text-red-400 font-medium"
                                    >
                                        Cancel Indexing
                                    </button>
                                    <div className="flex-1 text-xs text-gray-500 flex items-center justify-center">
                                        This may take a few moments depending on
                                        folder size
                                    </div>
                                </>
                            ) : (
                                <button
                                    onClick={handleCloseProgressModal}
                                    className="flex-1 px-4 py-2.5 bg-white hover:bg-gray-200 text-black rounded-lg transition-colors font-medium"
                                >
                                    Done
                                </button>
                            )}
                        </div>
                    </div>
                </div>
            )}

            {/* Full Text Modal */}
            {showFullTextModal && selectedResult && (
                <div className="fixed inset-0 bg-black/90 backdrop-blur-sm flex items-center justify-center z-50 p-4">
                    <div className="bg-black border border-white/20 rounded-lg max-w-4xl w-full max-h-[90vh] flex flex-col shadow-2xl">
                        {/* Modal Header */}
                        <div className="flex items-start justify-between p-6 border-b border-white/10">
                            <div className="flex-1 min-w-0 pr-4">
                                <h2 className="text-lg font-medium mb-2 flex items-center gap-2">
                                    {selectedResult.contentType === "image" ? (
                                        <ImageIcon className="w-5 h-5 flex-shrink-0" />
                                    ) : (
                                        <FileText className="w-5 h-5 flex-shrink-0" />
                                    )}
                                    <span className="truncate">
                                        {selectedResult.fileName ||
                                            selectedResult.filePath}
                                    </span>
                                </h2>
                                <div className="flex items-center gap-4 text-xs text-gray-400">
                                    {selectedResult.pageNumbers &&
                                        selectedResult.pageNumbers.length >
                                            0 && (
                                            <span className="bg-blue-500/10 px-2 py-1 rounded border border-blue-500/20 text-blue-400">
                                                Page{" "}
                                                {selectedResult.pageNumbers[0]}
                                            </span>
                                        )}
                                    {selectedResult.chunkIndex !== undefined &&
                                        selectedResult.totalChunks && (
                                            <span>
                                                Chunk{" "}
                                                {selectedResult.chunkIndex + 1}{" "}
                                                / {selectedResult.totalChunks}
                                            </span>
                                        )}
                                    <span className="px-2 py-1 rounded bg-white/10">
                                        {(
                                            (1 - selectedResult.similarity) *
                                            100
                                        ).toFixed(0)}
                                        % match
                                    </span>
                                </div>
                            </div>
                            <button
                                onClick={() => {
                                    setShowFullTextModal(false);
                                    setSelectedResult(null);
                                }}
                                className="text-gray-400 hover:text-white transition-colors"
                            >
                                <X className="w-5 h-5" />
                            </button>
                        </div>

                        {/* Modal Body */}
                        <div className="flex-1 overflow-y-auto p-6">
                            {selectedResult.contentType === "image" &&
                            selectedResult.imageData ? (
                                <div className="flex flex-col items-center">
                                    <img
                                        src={`data:image/jpeg;base64,${selectedResult.imageData}`}
                                        alt={selectedResult.fileName}
                                        className="max-w-full h-auto rounded-lg border border-white/10 shadow-lg"
                                    />
                                    <p className="text-sm text-gray-400 mt-4 italic text-center">
                                        {selectedResult.content}
                                    </p>
                                </div>
                            ) : (
                                <div className="prose prose-invert max-w-none">
                                    <div className="text-sm leading-relaxed text-gray-300 whitespace-pre-wrap font-mono bg-white/5 p-4 rounded-lg border border-white/10">
                                        {selectedResult.content}
                                    </div>
                                </div>
                            )}
                        </div>

                        {/* Modal Footer */}
                        <div className="border-t border-white/10 p-6 bg-white/[0.02]">
                            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4 text-xs">
                                {selectedResult.fileSize && (
                                    <div>
                                        <div className="text-gray-500 mb-1">
                                            File Size
                                        </div>
                                        <div className="flex items-center gap-1.5 text-gray-300">
                                            <HardDrive className="w-3 h-3" />
                                            {formatFileSize(
                                                selectedResult.fileSize,
                                            )}
                                        </div>
                                    </div>
                                )}
                                {selectedResult.modifiedDate && (
                                    <div>
                                        <div className="text-gray-500 mb-1">
                                            Modified
                                        </div>
                                        <div className="flex items-center gap-1.5 text-gray-300">
                                            <Clock className="w-3 h-3" />
                                            {selectedResult.modifiedDate}
                                        </div>
                                    </div>
                                )}
                                <div className="col-span-2">
                                    <div className="text-gray-500 mb-1">
                                        File Path
                                    </div>
                                    <div className="flex items-center gap-1.5 text-gray-300 font-mono text-xs">
                                        <Folder className="w-3 h-3 flex-shrink-0" />
                                        <span className="truncate">
                                            {selectedResult.filePath}
                                        </span>
                                    </div>
                                </div>
                            </div>
                            <div className="flex gap-2">
                                <button
                                    onClick={() =>
                                        window.open(
                                            "/api/file?path=" +
                                                encodeURIComponent(
                                                    selectedResult.filePath,
                                                ),
                                            "_blank",
                                        )
                                    }
                                    className="flex items-center gap-2 px-4 py-2 bg-white text-black rounded-md hover:opacity-90 text-sm font-medium"
                                >
                                    <ExternalLink className="w-4 h-4" />
                                    Open File
                                </button>
                                <button
                                    onClick={() => {
                                        setShowFullTextModal(false);
                                        setSelectedResult(null);
                                    }}
                                    className="flex-1 px-4 py-2 border border-white/10 rounded-md hover:bg-white/5 text-sm"
                                >
                                    Close
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}

export default App;
