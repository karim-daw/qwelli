import { useState, useEffect, useRef } from "react";
import {
    Search,
    Folder,
    Plus,
    ExternalLink,
    Image as ImageIcon,
    X,
    RefreshCw,
    AlertCircle,
    CheckCircle,
    Menu,
    ChevronRight,
    FileText,
    Clock,
    HardDrive,
    Activity,
    ZoomIn,
    ZoomOut,
    Download,
    ChevronLeft,
    ChevronRight as ChevronRightIcon,
    Calendar,
    AlignLeft,
    Sun,
    Moon,
    Terminal as TerminalIcon,
    ChevronDown,
    Power,
} from "lucide-react";
import { Document, Page, pdfjs } from "react-pdf";
import "react-pdf/dist/Page/AnnotationLayer.css";
import "react-pdf/dist/Page/TextLayer.css";
// this is a comment
// this is a comment

// Configure PDF.js worker
pdfjs.GlobalWorkerOptions.workerSrc = `//unpkg.com/pdfjs-dist@${pdfjs.version}/build/pdf.worker.min.mjs`;

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

interface RecentSearch {
    query: string;
    strategy: string;
    contentFilter: string;
    topK: number;
    timestamp: number;
    resultCount: number;
    results: SearchResult[];
    cacheStatus?: string;
}

function App() {
    const [indexes, setIndexes] = useState<Index[]>([]);
    const [selectedIndex, setSelectedIndex] = useState<string>("");
    const [query, setQuery] = useState("");
    const [results, setResults] = useState<SearchResult[]>([]);
    const [loading, setLoading] = useState(false);
    const [showNewIndexDialog, setShowNewIndexDialog] = useState(false);
    const [showSidebar, setShowSidebar] = useState(true);
    const [viewMode, setViewMode] = useState<"search" | "status">("search");
    const [indexStatus, setIndexStatus] = useState<IndexStatus | null>(null);
    const [loadingStatus, setLoadingStatus] = useState(false);
    const [newIndexPath, setNewIndexPath] = useState("");
    const [indexContentType, setIndexContentType] = useState<"text" | "images" | "text_images" | "text_pdfimages" | "all">("all");
    const [indexing, setIndexing] = useState(false);
    const [indexProgress, setIndexProgress] = useState<IndexProgress | null>(
        null,
    );
    const [indexingComplete, setIndexingComplete] = useState(false);
    const [cancelling, setCancelling] = useState(false);
    const [currentPhase, setCurrentPhase] = useState<string>("");
    const [selectedResult, setSelectedResult] = useState<SearchResult | null>(
        null,
    );
    const [showFullTextModal, setShowFullTextModal] = useState(false);
    const [cacheStatus, setCacheStatus] = useState<string | null>(null);
    // Search settings
    const [strategy, setStrategy] = useState("semantic");
    const [topK, setTopK] = useState(10);
    const [contentFilter, setContentFilter] = useState("all");

    // Recent searches
    const [recentSearches, setRecentSearches] = useState<RecentSearch[]>([]);
    const MAX_RECENT_SEARCHES = 10;

    // PDF preview
    const [showPDFPreview, setShowPDFPreview] = useState(false);
    const [pdfPreviewData, setPDFPreviewData] = useState<{
        filePath: string;
        initialPage: number;
        fileName: string;
    } | null>(null);
    const [pdfNumPages, setPdfNumPages] = useState<number>(0);
    const [pdfPageNumber, setPdfPageNumber] = useState(1);
    const [pdfScale, setPdfScale] = useState(1.0);

    // Update/sync progress
    const [updating, setUpdating] = useState(false);
    const [updateProgress, setUpdateProgress] = useState<IndexProgress | null>(
        null,
    );
    const [updateComplete, setUpdateComplete] = useState(false);

    // New UI features
    const [indexSortType, setIndexSortType] = useState<"alphabetical" | "date">("alphabetical");
    const [theme, setTheme] = useState<"dark" | "light">("dark");
    const [showTerminal, setShowTerminal] = useState(false);
    const [sidebarWidth, setSidebarWidth] = useState(256); // 16rem = 256px
    const [terminalHeight, setTerminalHeight] = useState(200);
    const [isResizingSidebar, setIsResizingSidebar] = useState(false);
    const [isResizingTerminal, setIsResizingTerminal] = useState(false);
    const [terminalLogs, setTerminalLogs] = useState<Array<{type: string, level: string, message: string, timestamp: number}>>([]);
    const terminalRef = useRef<HTMLDivElement>(null);

    // First-run setup
    const [needsSetup, setNeedsSetup] = useState<boolean | null>(null);
    const [setupApiKey, setSetupApiKey] = useState("");
    const [setupSaving, setSetupSaving] = useState(false);
    const [setupError, setSetupError] = useState("");
    const [quitConfirmed, setQuitConfirmed] = useState(false);

    // localStorage helper functions
    const loadRecentSearches = (indexPath: string): RecentSearch[] => {
        try {
            const stored = localStorage.getItem("qwelli_recent_searches");
            if (!stored) return [];

            const allSearches: Record<string, RecentSearch[]> =
                JSON.parse(stored);
            return allSearches[indexPath] || [];
        } catch (error) {
            console.error("Failed to load recent searches:", error);
            return [];
        }
    };

    const saveRecentSearch = (indexPath: string, search: RecentSearch) => {
        try {
            const stored = localStorage.getItem("qwelli_recent_searches");
            const allSearches: Record<string, RecentSearch[]> = stored
                ? JSON.parse(stored)
                : {};

            // Get current searches for this index
            const indexSearches = allSearches[indexPath] || [];

            // Remove duplicate if same query exists
            const filtered = indexSearches.filter(
                (s) =>
                    !(
                        s.query === search.query &&
                        s.strategy === search.strategy &&
                        s.contentFilter === search.contentFilter &&
                        s.topK === search.topK
                    ),
            );

            // Add new search at the beginning
            filtered.unshift(search);

            // Keep only MAX_RECENT_SEARCHES
            allSearches[indexPath] = filtered.slice(0, MAX_RECENT_SEARCHES);

            localStorage.setItem(
                "qwelli_recent_searches",
                JSON.stringify(allSearches),
            );
            setRecentSearches(allSearches[indexPath]);
        } catch (error) {
            console.error("Failed to save recent search:", error);
        }
    };

    const clearRecentSearches = (indexPath: string) => {
        try {
            const stored = localStorage.getItem("qwelli_recent_searches");
            if (!stored) return;

            const allSearches: Record<string, RecentSearch[]> =
                JSON.parse(stored);
            delete allSearches[indexPath];

            localStorage.setItem(
                "qwelli_recent_searches",
                JSON.stringify(allSearches),
            );
            setRecentSearches([]);
        } catch (error) {
            console.error("Failed to clear recent searches:", error);
        }
    };

    const loadCachedSearch = (search: RecentSearch) => {
        // Load the search parameters
        setQuery(search.query);
        setStrategy(search.strategy);
        setContentFilter(search.contentFilter);
        setTopK(search.topK);

        // Load the cached results directly
        setResults(search.results);
        setCacheStatus("LOCAL");
    };

    const formatTimestamp = (timestamp: number): string => {
        const now = Date.now();
        const diff = now - timestamp;

        const minutes = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);

        if (minutes < 1) return "Just now";
        if (minutes < 60) return `${minutes}m ago`;
        if (hours < 24) return `${hours}h ago`;
        if (days < 7) return `${days}d ago`;

        return new Date(timestamp).toLocaleDateString();
    };

    const openPDFPreview = (result: SearchResult) => {
        const isPDF =
            result.fileName?.toLowerCase().endsWith(".pdf") ||
            result.filePath?.toLowerCase().endsWith(".pdf");
        if (!isPDF) return;

        setPDFPreviewData({
            filePath: result.filePath,
            initialPage: result.pageNumbers?.[0] || 1,
            fileName: result.fileName || result.filePath,
        });
        setPdfPageNumber(result.pageNumbers?.[0] || 1);
        setShowPDFPreview(true);
    };

    const closePDFPreview = () => {
        setShowPDFPreview(false);
        setPDFPreviewData(null);
        setPdfNumPages(0);
        setPdfPageNumber(1);
        setPdfScale(1.0);
    };

    const onPDFLoadSuccess = ({ numPages }: { numPages: number }) => {
        setPdfNumPages(numPages);
    };

    useEffect(() => {
        fetch("/api/setup/status")
            .then((res) => res.json())
            .then((data) => setNeedsSetup(data.needsSetup === true))
            .catch(() => setNeedsSetup(false));
    }, []);

    useEffect(() => {
        if (needsSetup === false) {
            fetchIndexes();
        }
    }, [needsSetup]);

    useEffect(() => {
        if (selectedIndex) {
            const recent = loadRecentSearches(selectedIndex);
            setRecentSearches(recent);
        }
    }, [selectedIndex]);

    // Connect to terminal SSE stream with auto-reconnect
    useEffect(() => {
        let eventSource: EventSource | null = null;
        let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
        let isMounted = true;

        const connect = () => {
            if (!isMounted) return;
            
            eventSource = new EventSource("/api/terminal/stream");

            eventSource.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    if (data.type === "log") {
                        setTerminalLogs((prev) => [
                            ...prev,
                            {
                                type: data.type,
                                level: data.level || "info",
                                message: data.message,
                                timestamp: Date.now(),
                            },
                        ]);
                    }
                } catch (error) {
                    // ignore parse errors
                }
            };

            eventSource.onerror = () => {
                if (eventSource) eventSource.close();
                // Auto-reconnect after 2 seconds
                if (isMounted) {
                    reconnectTimer = setTimeout(connect, 2000);
                }
            };
        };

        connect();

        return () => {
            isMounted = false;
            if (eventSource) eventSource.close();
            if (reconnectTimer) clearTimeout(reconnectTimer);
        };
    }, []);

    // Auto-scroll terminal to bottom on new logs
    useEffect(() => {
        if (terminalRef.current && showTerminal) {
            terminalRef.current.scrollTop = terminalRef.current.scrollHeight;
        }
    }, [terminalLogs, showTerminal]);

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

            const searchResults = data.results || [];
            // Debug: log page numbers for PDFs
            searchResults.forEach((r: SearchResult) => {
                const isPDF = r.fileName?.toLowerCase().endsWith(".pdf") ||
                    r.filePath?.toLowerCase().endsWith(".pdf");
                if (isPDF) {
                    console.log("PDF result:", {
                        fileName: r.fileName,
                        pageNumbers: r.pageNumbers,
                        hasPageNumbers: r.pageNumbers && r.pageNumbers.length > 0,
                        metadata: r.metadata
                    });
                }
            });
            setResults(searchResults);

            // Save to recent searches with results
            saveRecentSearch(selectedIndex, {
                query,
                strategy,
                contentFilter,
                topK,
                timestamp: Date.now(),
                resultCount: searchResults.length,
                results: searchResults,
                cacheStatus: cacheHeader || undefined,
            });
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

        try {
            // Connect to SSE progress stream
            const eventSource = new EventSource(
                "/api/index/progress?path=" + encodeURIComponent(newIndexPath),
            );

            eventSource.onmessage = (event) => {
                const data = JSON.parse(event.data);

                if (data.type === "progress") {
                    setIndexProgress({
                        current: data.current,
                        total: data.total,
                        file: data.file,
                        indexPath: newIndexPath,
                    });
                } else if (data.type === "phase") {
                    setCurrentPhase(data.message || data.phase);
                } else if (data.type === "complete") {
                    eventSource.close();
                    setIndexingComplete(true);
                    setIndexing(false);
                    setCurrentPhase("");
                    fetchIndexes();
                } else if (data.type === "cancelled") {
                    eventSource.close();
                    setIndexProgress(null);
                    setIndexing(false);
                    setIndexingComplete(false);
                    setCancelling(false);
                    setCurrentPhase("");
                    setShowNewIndexDialog(false);
                    setNewIndexPath("");
                } else if (data.type === "error") {
                    eventSource.close();
                    setIndexProgress(null);
                    setIndexing(false);
                    setIndexingComplete(false);
                    setCurrentPhase("");
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
                body: JSON.stringify({
                    folderPath: newIndexPath,
                    contentType: indexContentType
                }),
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
        if (!indexProgress || cancelling) return;

        setCancelling(true);
        try {
            const response = await fetch("/api/index/cancel", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ indexPath: indexProgress.indexPath }),
            });

            if (!response.ok) {
                const data = await response.json();
                alert("Failed to cancel: " + data.error);
                setCancelling(false);
            }
            // Note: setCancelling will be reset when the cancelled event is received
        } catch (error) {
            alert("Failed to cancel indexing: " + error);
            setCancelling(false);
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
                    setQuery("");
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

    const handleOpenFileLocation = async (filePath: string) => {
        try {
            const response = await fetch("/api/open-file-location", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ path: filePath }),
            });

            if (!response.ok) {
                const data = await response.json();
                alert("Failed to open file location: " + data.error);
            }
        } catch (error) {
            alert("Failed to open file location: " + error);
        }
    };

    const handleCloseProgressModal = () => {
        setIndexProgress(null);
        setIndexingComplete(false);
        setShowNewIndexDialog(false);
        setNewIndexPath("");
    };

    const currentIndex = indexes.find((i) => i.path === selectedIndex);

    // Sort indexes based on sort type
    const sortedIndexes = [...indexes].sort((a, b) => {
        if (indexSortType === "alphabetical") {
            return a.name.localeCompare(b.name);
        } else {
            // Sort by date (most recent first)
            return new Date(b.lastModified).getTime() - new Date(a.lastModified).getTime();
        }
    });

    // Handle sidebar resize
    const handleSidebarMouseDown = () => {
        setIsResizingSidebar(true);
    };

    const handleSidebarMouseMove = (e: MouseEvent) => {
        if (isResizingSidebar) {
            const newWidth = e.clientX;
            if (newWidth >= 200 && newWidth <= 500) {
                setSidebarWidth(newWidth);
            }
        }
    };

    const handleSidebarMouseUp = () => {
        setIsResizingSidebar(false);
    };

    // Handle terminal resize
    const handleTerminalMouseDown = () => {
        setIsResizingTerminal(true);
    };

    const handleTerminalMouseMove = (e: MouseEvent) => {
        if (isResizingTerminal) {
            const newHeight = window.innerHeight - e.clientY;
            if (newHeight >= 100 && newHeight <= 600) {
                setTerminalHeight(newHeight);
            }
        }
    };

    const handleTerminalMouseUp = () => {
        setIsResizingTerminal(false);
    };

    useEffect(() => {
        if (isResizingSidebar) {
            document.addEventListener("mousemove", handleSidebarMouseMove as any);
            document.addEventListener("mouseup", handleSidebarMouseUp);
        } else {
            document.removeEventListener("mousemove", handleSidebarMouseMove as any);
            document.removeEventListener("mouseup", handleSidebarMouseUp);
        }
        return () => {
            document.removeEventListener("mousemove", handleSidebarMouseMove as any);
            document.removeEventListener("mouseup", handleSidebarMouseUp);
        };
    }, [isResizingSidebar]);

    useEffect(() => {
        if (isResizingTerminal) {
            document.addEventListener("mousemove", handleTerminalMouseMove as any);
            document.addEventListener("mouseup", handleTerminalMouseUp);
        } else {
            document.removeEventListener("mousemove", handleTerminalMouseMove as any);
            document.removeEventListener("mouseup", handleTerminalMouseUp);
        }
        return () => {
            document.removeEventListener("mousemove", handleTerminalMouseMove as any);
            document.removeEventListener("mouseup", handleTerminalMouseUp);
        };
    }, [isResizingTerminal]);

    // Theme colors — centralised so every element adapts
    const isDark = theme === "dark";
    const bgColor = isDark ? "bg-black" : "bg-white";
    const textColor = isDark ? "text-white" : "text-gray-900";
    const borderColor = isDark ? "border-white/10" : "border-gray-200";
    const hoverBg = isDark ? "hover:bg-white/5" : "hover:bg-gray-100";
    const activeBg = isDark ? "bg-white/10" : "bg-gray-200";
    const inputBg = isDark ? "bg-white/5" : "bg-gray-100";

    // Extended palette for light/dark
    const mutedText = isDark ? "text-gray-400" : "text-gray-500";
    const subtleText = isDark ? "text-gray-500" : "text-gray-400";
    const bodyText = isDark ? "text-gray-300" : "text-gray-700";
    const cardBg = isDark ? "bg-white/[0.02]" : "bg-gray-50";
    const cardBorder = isDark ? "border-white/10" : "border-gray-200";
    const cardHoverBorder = isDark ? "hover:border-white/20" : "hover:border-gray-300";
    const panelBg = isDark ? "bg-white/5" : "bg-gray-100";
    const badgeBg = isDark ? "bg-white/10" : "bg-gray-200";
    const headerBg = isDark ? "bg-black/80" : "bg-white/80";
    const hoverText = isDark ? "hover:text-white" : "hover:text-gray-900";
    const hoverBgActive = isDark ? "hover:bg-white/10" : "hover:bg-gray-200";
    const iconColor = isDark ? "text-gray-400" : "text-gray-500";
    const iconHoverColor = isDark ? "hover:text-white" : "hover:text-gray-900";
    const overlayBg = isDark ? "bg-black/80" : "bg-black/50";
    const modalBg = isDark ? "bg-black" : "bg-white";
    const modalBorder = isDark ? "border-white/20" : "border-gray-300";
    const progressTrack = isDark ? "bg-white/10" : "bg-gray-200";
    const progressFill = isDark ? "bg-white" : "bg-gray-900";
    const progressDot = isDark ? "bg-white" : "bg-gray-900";
    const accentBtnBg = isDark ? "bg-white" : "bg-gray-900";
    const accentBtnText = isDark ? "text-black" : "text-white";
    const accentBtnHover = isDark ? "hover:bg-gray-200" : "hover:bg-gray-800";
    const secondaryBtnBorder = isDark ? "border-white/10" : "border-gray-300";
    const secondaryBtnHover = isDark ? "hover:bg-white/5" : "hover:bg-gray-100";
    const pdfViewerBg = isDark ? "bg-gray-900" : "bg-gray-200";
    const activeItemBg = isDark ? "bg-white/20" : "bg-gray-800";
    const activeItemText = isDark ? "text-white" : "text-white";
    const inactiveItemText = isDark ? "text-gray-400" : "text-gray-600";
    const inactiveItemHover = isDark ? "hover:text-gray-300 hover:bg-white/5" : "hover:text-gray-900 hover:bg-gray-100";
    const placeholderColor = isDark ? "placeholder:text-gray-500" : "placeholder:text-gray-400";
    const focusBorder = isDark ? "focus:border-white/30" : "focus:border-gray-400";

    const handleSetupSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setSetupError("");
        setSetupSaving(true);
        try {
            const res = await fetch("/api/setup", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    apiKey: setupApiKey.trim(),
                    model: "voyage-multimodal-3",
                    endpoint: "https://api.voyageai.com/v1/multimodalembeddings",
                    enableMultimodal: true,
                    enableReranker: true,
                }),
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.error || "Setup failed");
            window.location.reload();
        } catch (err) {
            setSetupError(err instanceof Error ? err.message : "Setup failed");
        } finally {
            setSetupSaving(false);
        }
    };

    const handleQuit = async () => {
        try {
            await fetch("/api/shutdown", { method: "POST" });
            setQuitConfirmed(true);
        } catch {
            setQuitConfirmed(true);
        }
    };

    if (needsSetup === true) {
        return (
            <div className={`flex h-screen ${bgColor} ${textColor} ${isDark ? "dark" : "light"}`}>
                <div className="flex-1 flex flex-col items-center justify-center p-8">
                    <div className={`w-full max-w-md ${modalBg} border ${modalBorder} rounded-xl p-8 shadow-2xl`}>
                        <h1 className="text-2xl font-semibold mb-2">Welcome to Qwelli</h1>
                        <p className={`text-sm ${mutedText} mb-6`}>
                            Enter your Voyage AI API key to get started. Get one at{" "}
                            <a href="https://dash.voyageai.com/" target="_blank" rel="noreferrer" className="underline">dash.voyageai.com</a>
                        </p>
                        <form onSubmit={handleSetupSubmit} className="space-y-4">
                            <div>
                                <label className={`block text-sm font-medium ${mutedText} mb-2`}>API Key</label>
                                <input
                                    type="password"
                                    value={setupApiKey}
                                    onChange={(e) => setSetupApiKey(e.target.value)}
                                    placeholder="sk-..."
                                    required
                                    className={`w-full px-4 py-2.5 ${inputBg} border ${cardBorder} rounded-lg ${focusBorder} outline-none ${placeholderColor}`}
                                    autoComplete="off"
                                />
                            </div>
                            {setupError && <p className="text-sm text-red-500">{setupError}</p>}
                            <button
                                type="submit"
                                disabled={setupSaving || !setupApiKey.trim()}
                                className={`w-full py-2.5 ${accentBtnBg} ${accentBtnText} rounded-lg font-medium disabled:opacity-50 disabled:cursor-not-allowed`}
                            >
                                {setupSaving ? "Saving..." : "Continue"}
                            </button>
                            <button
                                type="button"
                                onClick={handleQuit}
                                className={`w-full py-2 border ${secondaryBtnBorder} rounded-lg ${secondaryBtnHover} text-sm mt-2`}
                            >
                                Quit
                            </button>
                        </form>
                    </div>
                </div>
            </div>
        );
    }

    if (quitConfirmed) {
        return (
            <div className={`flex h-screen ${bgColor} ${textColor} items-center justify-center ${isDark ? "dark" : "light"}`}>
                <div className="text-center">
                    <CheckCircle className={`w-16 h-16 mx-auto mb-4 ${isDark ? "text-green-400" : "text-green-600"}`} />
                    <p className="text-lg font-medium">Qwelli has been stopped</p>
                    <p className={`text-sm ${mutedText} mt-1`}>You can close this tab</p>
                </div>
            </div>
        );
    }

    return (
        <div className={`flex h-screen ${bgColor} ${textColor} overflow-hidden ${isDark ? "dark" : "light"}`}>
            {/* Sidebar */}
            {showSidebar && (
                <div
                    style={{ width: sidebarWidth }}
                    className={`flex-shrink-0 border-r ${borderColor} relative`}
                >
                    <div className="h-full flex flex-col">
                        {/* Sidebar Header */}
                        <div className={`p-4 border-b ${borderColor}`}>
                            <div className="flex items-center justify-between mb-3">
                                <h2 className="text-sm font-medium">Indexes</h2>
                                <div className="flex items-center gap-2">
                                    <button
                                        onClick={() => setIndexSortType(indexSortType === "alphabetical" ? "date" : "alphabetical")}
                                        className={`p-1.5 ${hoverBg} rounded-md transition-colors`}
                                        title={`Sort ${indexSortType === "alphabetical" ? "by date" : "alphabetically"}`}
                                    >
                                        {indexSortType === "alphabetical" ? (
                                            <AlignLeft className="w-4 h-4" />
                                        ) : (
                                            <Calendar className="w-4 h-4" />
                                        )}
                                    </button>
                                    <button
                                        onClick={() => setShowNewIndexDialog(true)}
                                        className={`p-1.5 ${hoverBg} rounded-md transition-colors`}
                                    >
                                        <Plus className="w-4 h-4" />
                                    </button>
                                </div>
                            </div>
                        </div>

                        {/* Index List */}
                        <div className="flex-1 overflow-y-auto custom-scrollbar">
                            {sortedIndexes?.map((index) => (
                            <button
                                key={index.path}
                                onClick={() => {
                                    setSelectedIndex(index.path);
                                    setViewMode("search");
                                    setResults([]);
                                    setQuery("");
                                }}
                                className={`w-full text-left px-4 py-2.5 border-b ${borderColor} ${hoverBg} transition-colors group ${selectedIndex === index.path
                                    ? activeBg
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
                                                className={`p-0.5 ${hoverBgActive} rounded transition-colors`}
                                                title="Open folder in file explorer"
                                            >
                                                <Folder className={`w-3.5 h-3.5 flex-shrink-0 ${iconColor} ${iconHoverColor}`} />
                                            </button>
                                            <span className="text-sm truncate">
                                                {index.name}
                                            </span>
                                        </div>
                                        <div className={`text-xs ${subtleText}`}>
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
                                            className={`opacity-0 group-hover:opacity-100 p-1 ${hoverBgActive} rounded transition-opacity`}
                                            title="View status & changes"
                                        >
                                            <Activity className="w-3.5 h-3.5" />
                                        </button>
                                        <button
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                handleDeleteIndex(index.path);
                                            }}
                                            className={`opacity-0 group-hover:opacity-100 p-1 hover:bg-red-500/20 rounded transition-opacity ${isDark ? "text-red-400" : "text-red-600"}`}
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
                {/* Resize Handle */}
                <div
                    className={`absolute top-0 right-0 w-1 h-full cursor-col-resize ${hoverBg} transition-colors z-10`}
                    onMouseDown={handleSidebarMouseDown}
                />
            </div>
            )}

            {/* Main Content */}
            <div className="flex-1 flex flex-col overflow-hidden">
                {/* Top Bar */}
                <div className={`flex-shrink-0 border-b ${borderColor} ${headerBg} backdrop-blur-xl`}>
                    <div className="px-6 py-4">
                        <div className="flex items-center justify-between mb-4">
                            <div className="flex items-center gap-4">
                                <button
                                    onClick={() => setShowSidebar(!showSidebar)}
                                    className={`p-1.5 ${hoverBg} rounded-md transition-colors`}
                                >
                                    <Menu className="w-5 h-5" />
                                </button>
                                <h1 className="text-lg font-medium">Qwelli</h1>
                                {currentIndex && (
                                    <div className={`flex items-center gap-2 text-sm ${mutedText}`}>
                                        <ChevronRight className="w-4 h-4" />
                                        <span>{currentIndex.name}</span>
                                    </div>
                                )}
                            </div>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={() => setShowTerminal(!showTerminal)}
                                    className={`p-1.5 ${hoverBg} rounded-md transition-colors ${showTerminal ? activeBg : ""}`}
                                    title="Toggle terminal"
                                >
                                    <TerminalIcon className="w-5 h-5" />
                                </button>
                                <button
                                    onClick={() => setTheme(isDark ? "light" : "dark")}
                                    className={`p-1.5 ${hoverBg} rounded-md transition-colors`}
                                    title={`Switch to ${isDark ? "light" : "dark"} mode`}
                                >
                                    {isDark ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
                                </button>
                                <button
                                    onClick={handleQuit}
                                    className={`p-1.5 ${hoverBg} rounded-md transition-colors ${isDark ? "text-red-400 hover:text-red-300" : "text-red-600 hover:text-red-700"}`}
                                    title="Quit Qwelli"
                                >
                                    <Power className="w-5 h-5" />
                                </button>
                            </div>
                        </div>

                        {viewMode === "search" && (
                            <form onSubmit={handleSearch} className="relative">
                                <div className="relative">
                                    <input
                                        type="text"
                                        value={query}
                                        onChange={(e) =>
                                            setQuery(e.target.value)
                                        }
                                        placeholder="Search knowledge base..."
                                        disabled={!selectedIndex}
                                        className={`w-full pl-4 pr-4 py-3.5 text-base ${inputBg} border ${borderColor} rounded-xl ${focusBorder} outline-none transition-colors disabled:opacity-50 disabled:cursor-not-allowed ${placeholderColor}`}
                                    />
                                </div>

                                {/* Inline Settings Controls */}
                                <div className="mt-3 flex items-center gap-4 text-xs">
                                    {/* Search Strategy */}
                                    <div className="flex items-center gap-2">
                                        <span className={subtleText}>Strategy:</span>
                                        <div className="flex gap-1">
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
                                                        "px-2.5 py-1 rounded-md transition-all text-xs font-medium " +
                                                        (strategy === s
                                                            ? `${activeItemBg} ${activeItemText}`
                                                            : `${inactiveItemText} ${inactiveItemHover}`)
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

                                    {/* Divider */}
                                    <div className={`w-px h-4 ${borderColor}`} />

                                    {/* Content Type */}
                                    <div className="flex items-center gap-2">
                                        <span className={subtleText}>Type:</span>
                                        <div className="flex gap-1">
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
                                                        "px-2.5 py-1 rounded-md transition-all text-xs font-medium " +
                                                        (contentFilter ===
                                                            c.value
                                                            ? `${activeItemBg} ${activeItemText}`
                                                            : `${inactiveItemText} ${inactiveItemHover}`)
                                                    }
                                                >
                                                    {c.label}
                                                </button>
                                            ))}
                                        </div>
                                    </div>

                                    {/* Divider */}
                                    <div className={`w-px h-4 ${borderColor}`} />

                                    {/* Results Count */}
                                    <div className="flex items-center gap-2">
                                        <span className={subtleText}>Results:</span>
                                        <div className="flex items-center gap-2">
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
                                                className={`w-20 h-1 ${progressTrack} rounded-lg appearance-none cursor-pointer ${isDark ? "accent-white" : "accent-gray-900"}`}
                                            />
                                            <span className={`${mutedText} min-w-[2rem] text-right`}>
                                                {topK}
                                            </span>
                                        </div>
                                    </div>
                                </div>
                            </form>
                        )}
                    </div>
                </div>

                {/* Content Area */}
                <div className="flex-1 overflow-y-auto">
                    <div className="max-w-5xl mx-auto px-6 py-6">
                        {viewMode === "status" ? (
                            loadingStatus ? (
                                <div className={`text-center ${mutedText} py-20`}>
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
                                                            !confirm(
                                                                "Sync all changes? This will update the index with new, modified, and deleted files.",
                                                            )
                                                        ) {
                                                            return;
                                                        }

                                                        setUpdating(true);
                                                        setUpdateComplete(false);

                                                        try {
                                                            // Connect to SSE progress stream
                                                            const eventSource =
                                                                new EventSource(
                                                                    "/api/index/progress?path=" +
                                                                    encodeURIComponent(
                                                                        selectedIndex,
                                                                    ),
                                                                );

                                                            eventSource.onmessage =
                                                                (event) => {
                                                                    const data =
                                                                        JSON.parse(
                                                                            event.data,
                                                                        );

                                                                    if (
                                                                        data.type ===
                                                                        "progress"
                                                                    ) {
                                                                        setUpdateProgress(
                                                                            {
                                                                                current:
                                                                                    data.current,
                                                                                total: data.total,
                                                                                file: data.file,
                                                                                indexPath:
                                                                                    selectedIndex,
                                                                            },
                                                                        );
                                                                    } else if (
                                                                        data.type ===
                                                                        "phase"
                                                                    ) {
                                                                        setCurrentPhase(
                                                                            data.message ||
                                                                            data.phase,
                                                                        );
                                                                    } else if (
                                                                        data.type ===
                                                                        "complete"
                                                                    ) {
                                                                        eventSource.close();
                                                                        setUpdateComplete(
                                                                            true,
                                                                        );
                                                                        setUpdating(
                                                                            false,
                                                                        );
                                                                        setCurrentPhase(
                                                                            "",
                                                                        );
                                                                        // Refresh status to show updated state
                                                                        fetchIndexStatus(
                                                                            selectedIndex,
                                                                        );
                                                                    } else if (
                                                                        data.type ===
                                                                        "cancelled"
                                                                    ) {
                                                                        eventSource.close();
                                                                        setUpdateProgress(
                                                                            null,
                                                                        );
                                                                        setUpdating(
                                                                            false,
                                                                        );
                                                                        setUpdateComplete(
                                                                            false,
                                                                        );
                                                                        setCurrentPhase(
                                                                            "",
                                                                        );
                                                                    } else if (
                                                                        data.type ===
                                                                        "error"
                                                                    ) {
                                                                        eventSource.close();
                                                                        setUpdateProgress(
                                                                            null,
                                                                        );
                                                                        setUpdating(
                                                                            false,
                                                                        );
                                                                        setCurrentPhase(
                                                                            "",
                                                                        );
                                                                        alert(
                                                                            "Sync error: " +
                                                                            data.message,
                                                                        );
                                                                    }
                                                                };

                                                            eventSource.onerror =
                                                                () => {
                                                                    eventSource.close();
                                                                    setUpdateProgress(
                                                                        null,
                                                                    );
                                                                    setUpdating(
                                                                        false,
                                                                    );
                                                                };

                                                            // Start the sync operation
                                                            const response =
                                                                await fetch(
                                                                    "/api/index/update",
                                                                    {
                                                                        method: "POST",
                                                                        headers: {
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

                                                            if (!response.ok) {
                                                                const data =
                                                                    await response.json();
                                                                eventSource.close();
                                                                setUpdateProgress(
                                                                    null,
                                                                );
                                                                setUpdating(false);
                                                                alert(
                                                                    "Error: " +
                                                                    data.error,
                                                                );
                                                            }
                                                        } catch (error) {
                                                            setUpdateProgress(null);
                                                            setUpdating(false);
                                                            alert(
                                                                "Failed to sync: " +
                                                                error,
                                                            );
                                                        }
                                                    }}
                                                    disabled={updating}
                                                    className={`flex items-center gap-2 px-4 py-2 ${accentBtnBg} ${accentBtnText} rounded-md hover:opacity-90 text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed`}
                                                >
                                                    <RefreshCw
                                                        className={`w-4 h-4 ${updating ? "animate-spin" : ""}`}
                                                    />
                                                    {updating
                                                        ? "Syncing..."
                                                        : "Sync Changes"}
                                                </button>
                                            )}
                                    </div>

                                    {/* Status Summary */}
                                    <div className="grid grid-cols-4 gap-4">
                                        <div className={`${panelBg} rounded-lg p-4 border ${cardBorder}`}>
                                            <div className="text-2xl font-semibold mb-1">
                                                {indexStatus.total}
                                            </div>
                                            <div className={`text-xs ${mutedText}`}>
                                                Total Files
                                            </div>
                                        </div>
                                        <div className="bg-green-500/10 rounded-lg p-4 border border-green-500/20">
                                            <div className={`text-2xl font-semibold ${isDark ? "text-green-400" : "text-green-600"} mb-1`}>
                                                {indexStatus.upToDate}
                                            </div>
                                            <div className={`text-xs ${mutedText}`}>
                                                Up to Date
                                            </div>
                                        </div>
                                        <div className="bg-yellow-500/10 rounded-lg p-4 border border-yellow-500/20">
                                            <div className={`text-2xl font-semibold ${isDark ? "text-yellow-400" : "text-yellow-600"} mb-1`}>
                                                {indexStatus.toUpdate?.length ||
                                                    0}
                                            </div>
                                            <div className={`text-xs ${mutedText}`}>
                                                Modified
                                            </div>
                                        </div>
                                        <div className="bg-blue-500/10 rounded-lg p-4 border border-blue-500/20">
                                            <div className={`text-2xl font-semibold ${isDark ? "text-blue-400" : "text-blue-600"} mb-1`}>
                                                {indexStatus.toAdd?.length || 0}
                                            </div>
                                            <div className={`text-xs ${mutedText}`}>
                                                New Files
                                            </div>
                                        </div>
                                    </div>

                                    {/* Change Details */}
                                    {indexStatus.toAdd?.length > 0 && (
                                        <div>
                                            <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
                                                <Plus className={`w-4 h-4 ${isDark ? "text-blue-400" : "text-blue-600"}`} />
                                                New Files (
                                                {indexStatus.toAdd.length})
                                            </h3>
                                            <div className="space-y-2">
                                                {indexStatus.toAdd
                                                    .slice(0, 10)
                                                    .map((file, i) => (
                                                        <div
                                                            key={i}
                                                            className={`text-xs ${panelBg} rounded px-3 py-2 border ${cardBorder}`}
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
                                                <AlertCircle className={`w-4 h-4 ${isDark ? "text-yellow-400" : "text-yellow-600"}`} />
                                                Modified Files (
                                                {indexStatus.toUpdate.length})
                                            </h3>
                                            <div className="space-y-2">
                                                {indexStatus.toUpdate
                                                    .slice(0, 10)
                                                    .map((file, i) => (
                                                        <div
                                                            key={i}
                                                            className={`text-xs ${panelBg} rounded px-3 py-2 border ${cardBorder}`}
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
                                                <X className={`w-4 h-4 ${isDark ? "text-red-400" : "text-red-600"}`} />
                                                Deleted Files (
                                                {indexStatus.toDelete.length})
                                            </h3>
                                            <div className="space-y-2">
                                                {indexStatus.toDelete
                                                    .slice(0, 10)
                                                    .map((file, i) => (
                                                        <div
                                                            key={i}
                                                            className={`text-xs ${panelBg} rounded px-3 py-2 border ${cardBorder}`}
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
                                                <CheckCircle className={`w-12 h-12 ${isDark ? "text-green-400" : "text-green-600"} mx-auto mb-3`} />
                                                <p className={mutedText}>
                                                    Index is up to date!
                                                </p>
                                            </div>
                                        )}
                                </div>
                            ) : (
                                <div className={`text-center ${subtleText} py-20`}>
                                    Select an index to view status
                                </div>
                            )
                        ) : !selectedIndex ? (
                            <div className={`text-center ${subtleText} py-20`}>
                                <Folder className="w-12 h-12 mx-auto mb-3 opacity-50" />
                                <p>Select an index from the sidebar to start</p>
                            </div>
                        ) : loading ? (
                            <div className={`text-center ${mutedText} py-20`}>
                                Searching...
                            </div>
                        ) : results?.length > 0 ? (
                            <div className="space-y-4">
                                {/* Results header with cache indicator */}
                                <div className={`flex items-center justify-between pb-2 border-b ${cardBorder}`}>
                                    <div className={`text-sm ${mutedText}`}>
                                        Found {results.length} result
                                        {results.length !== 1 ? "s" : ""}
                                    </div>
                                    {(cacheStatus === "HIT" ||
                                        cacheStatus === "LOCAL") && (
                                            <div className={`flex items-center gap-1.5 text-xs ${isDark ? "text-green-400" : "text-green-700"} bg-green-500/10 px-2 py-1 rounded border border-green-500/20`}>
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
                                                {cacheStatus === "LOCAL"
                                                    ? "Cached Locally"
                                                    : "Cached"}
                                            </div>
                                        )}
                                </div>
                                {results?.map((result, index) => (
                                    <div
                                        key={result.chunkId + index}
                                        className={`border ${cardBorder} rounded-lg p-5 ${cardHoverBorder} transition-colors ${cardBg}`}
                                    >
                                        {/* Header */}
                                        <div className="flex items-start justify-between gap-4 mb-3">
                                            <div className={`flex items-center gap-2 text-sm ${mutedText} flex-1 min-w-0`}>
                                                {result.contentType ===
                                                    "image" ? (
                                                    <ImageIcon className="w-4 h-4 flex-shrink-0" />
                                                ) : (
                                                    <FileText className="w-4 h-4 flex-shrink-0" />
                                                )}
                                                <span className={`font-mono text-xs truncate font-medium ${bodyText}`}>
                                                    {result.fileName ||
                                                        result.filePath}
                                                </span>
                                                {(() => {
                                                    const isPDF = (result.fileName?.toLowerCase().endsWith(".pdf") ||
                                                        result.filePath?.toLowerCase().endsWith(".pdf"));
                                                    const hasPageNumbers = result.pageNumbers &&
                                                        Array.isArray(result.pageNumbers) &&
                                                        result.pageNumbers.length > 0;

                                                    if (isPDF && hasPageNumbers && result.pageNumbers) {
                                                        return (
                                                            <button
                                                                onClick={() =>
                                                                    openPDFPreview(
                                                                        result,
                                                                    )
                                                                }
                                                                className={`text-xs flex-shrink-0 bg-blue-500/10 px-2 py-0.5 rounded border border-blue-500/20 ${isDark ? "text-blue-400" : "text-blue-600"} hover:bg-blue-500/20 transition-colors cursor-pointer font-medium`}
                                                                title="Preview PDF at this page"
                                                            >
                                                                Page{" "}
                                                                {result.pageNumbers[0]}
                                                            </button>
                                                        );
                                                    }
                                                    // Don't show chunk label for PDFs
                                                    if (isPDF) {
                                                        return null;
                                                    }
                                                    // Show chunk label for non-PDFs
                                                    if (result.chunkIndex !== undefined && result.totalChunks) {
                                                        return (
                                                            <span className={`text-xs flex-shrink-0 ${subtleText}`}>
                                                                Chunk{" "}
                                                                {result.chunkIndex + 1}
                                                                /
                                                                {result.totalChunks}
                                                            </span>
                                                        );
                                                    }
                                                    return null;
                                                })()}
                                            </div>
                                            <div className="flex items-center gap-2 flex-shrink-0">
                                                <span className={`text-xs font-medium px-2 py-1 rounded ${badgeBg}`}>
                                                    {(() => {
                                                        // Handle reranked results (negative values are relevance scores)
                                                        if (result.similarity < 0) {
                                                            return (Math.abs(result.similarity) * 100).toFixed(0);
                                                        }
                                                        // Handle distance metrics (0-1 range)
                                                        return ((1 - result.similarity) * 100).toFixed(0);
                                                    })()}
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
                                                    className={`p-1.5 ${iconColor} ${iconHoverColor} ${hoverBgActive} rounded transition-colors`}
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
                                                    className={`p-1.5 ${iconColor} ${iconHoverColor} ${hoverBgActive} rounded transition-colors`}
                                                    title="Open file"
                                                >
                                                    <ExternalLink className="w-4 h-4" />
                                                </button>
                                                <button
                                                    onClick={() =>
                                                        handleOpenFileLocation(
                                                            result.filePath,
                                                        )
                                                    }
                                                    className={`p-1.5 ${iconColor} ${iconHoverColor} ${hoverBgActive} rounded transition-colors`}
                                                    title="Show in file explorer"
                                                >
                                                    <Folder className="w-4 h-4" />
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
                                                    className={`max-w-full h-auto max-h-64 rounded-lg border ${cardBorder}`}
                                                />
                                                <p className={`text-xs ${subtleText} mt-2 italic`}>
                                                    {result.content}
                                                </p>
                                            </div>
                                        ) : (
                                            <p className={`text-sm leading-relaxed ${bodyText} mb-3 line-clamp-2`}>
                                                {result.content}
                                            </p>
                                        )}

                                        {/* Metadata Footer */}
                                        <div className={`flex items-center gap-4 text-xs ${subtleText} pt-3 border-t ${isDark ? "border-white/5" : "border-gray-100"}`}>
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
                            <div className={`text-center ${subtleText} py-20`}>
                                <p>No results found for "{query}"</p>
                            </div>
                        ) : recentSearches.length > 0 ? (
                            <div className="space-y-4">
                                <div className={`flex items-center justify-between pb-2 border-b ${cardBorder}`}>
                                    <h3 className={`text-sm font-medium ${mutedText}`}>
                                        Recent Searches
                                    </h3>
                                    <button
                                        onClick={() =>
                                            clearRecentSearches(selectedIndex)
                                        }
                                        className={`text-xs ${subtleText} ${hoverText} transition-colors`}
                                    >
                                        Clear all
                                    </button>
                                </div>

                                <div className="grid gap-3">
                                    {recentSearches.map((search, idx) => (
                                        <button
                                            key={idx}
                                            onClick={() =>
                                                loadCachedSearch(search)
                                            }
                                            className={`text-left border ${cardBorder} rounded-lg p-4 ${cardHoverBorder} ${hoverBg} transition-all group`}
                                        >
                                            <div className="flex items-start justify-between gap-4 mb-2">
                                                <div className="flex-1 min-w-0">
                                                    <div className="flex items-center gap-2 mb-1">
                                                        <Search className={`w-4 h-4 ${mutedText} flex-shrink-0`} />
                                                        <span className="font-medium truncate">
                                                            {search.query}
                                                        </span>
                                                    </div>
                                                    <div className={`flex items-center gap-2 text-xs ${subtleText}`}>
                                                        <span className={`px-2 py-0.5 ${panelBg} rounded`}>
                                                            {search.strategy}
                                                        </span>
                                                        {search.contentFilter !==
                                                            "all" && (
                                                                <span className={`px-2 py-0.5 ${panelBg} rounded`}>
                                                                    {
                                                                        search.contentFilter
                                                                    }
                                                                </span>
                                                            )}
                                                        <span>
                                                            {search.resultCount}{" "}
                                                            results
                                                        </span>
                                                    </div>
                                                </div>

                                                <div className="flex items-center gap-2 flex-shrink-0">
                                                    <div className={`text-xs ${subtleText} flex items-center gap-1`}>
                                                        <Clock className="w-3 h-3" />
                                                        {formatTimestamp(
                                                            search.timestamp,
                                                        )}
                                                    </div>
                                                    <div className={`text-xs bg-green-500/10 ${isDark ? "text-green-400" : "text-green-700"} px-2 py-1 rounded border border-green-500/20 flex items-center gap-1`}>
                                                        <HardDrive className="w-3 h-3" />
                                                        Cached
                                                    </div>
                                                </div>
                                            </div>
                                        </button>
                                    ))}
                                </div>
                            </div>
                        ) : (
                            <div className={`text-center ${subtleText} py-20`}>
                                <Search className="w-12 h-12 mx-auto mb-3 opacity-50" />
                                <p>
                                    Start searching to see recent searches here
                                </p>
                            </div>
                        )}
                    </div>
                </div>
            </div>

            {/* New Index Dialog */}
            {showNewIndexDialog && (
                <div className={`fixed inset-0 ${overlayBg} backdrop-blur-sm flex items-center justify-center z-50`}>
                    <div className={`${modalBg} border ${modalBorder} rounded-lg max-w-md w-full mx-4 p-6 shadow-2xl`}>
                        <div className="flex items-center justify-between mb-4">
                            <h2 className="text-lg font-medium">
                                Index a new folder
                            </h2>
                            <button
                                onClick={() => setShowNewIndexDialog(false)}
                                className={`${iconColor} ${iconHoverColor}`}
                                disabled={indexing}
                            >
                                <X className="w-5 h-5" />
                            </button>
                        </div>
                        <form onSubmit={handleCreateIndex}>
                            <div className="mb-4">
                                <label className={`block text-sm ${mutedText} mb-2`}>
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
                                        className={`flex-1 px-3 py-2 ${inputBg} border ${cardBorder} rounded-md ${focusBorder} outline-none`}
                                        disabled={indexing}
                                    />
                                    <label className={`px-4 py-2 ${badgeBg} ${hoverBgActive} border ${cardBorder} rounded-md cursor-pointer transition-colors flex items-center gap-2`}>
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
                                <p className={`text-xs ${subtleText} mt-2`}>
                                    Paste full path or use Browse (then complete
                                    the path manually)
                                </p>
                                {newIndexPath &&
                                    convertWindowsPathToWSL(newIndexPath) !==
                                    newIndexPath && (
                                        <p className={`text-xs ${subtleText} mt-1`}>
                                            →{" "}
                                            {convertWindowsPathToWSL(
                                                newIndexPath,
                                            )}
                                        </p>
                                    )}
                            </div>

                            {/* Content Type Selection */}
                            <div className="mt-4">
                                <label className="block text-sm font-medium mb-2">
                                    Index Content Type
                                </label>
                                <select
                                    value={indexContentType}
                                    onChange={(e) => setIndexContentType(e.target.value as typeof indexContentType)}
                                    disabled={indexing}
                                    className={`w-full px-3 py-2 text-sm rounded-md border transition-colors ${isDark ? "bg-gray-800 border-gray-600 text-gray-200" : "bg-white border-gray-300 text-gray-800"}`}
                                >
                                    <option value="all">Full Multimodal</option>
                                    <option value="text">Text Only</option>
                                    <option value="images">Images Only</option>
                                    <option value="text_images">Text + Images</option>
                                    <option value="text_pdfimages">Text + PDF Images</option>
                                </select>
                                <p className={`text-xs ${subtleText} mt-2`}>
                                    {indexContentType === "all" && "Text, standalone images, and PDF image extraction"}
                                    {indexContentType === "text" && "Text files and PDF text only (fastest)"}
                                    {indexContentType === "images" && "Standalone image files only"}
                                    {indexContentType === "text_images" && "Text files, PDF text, and standalone images (no PDF image extraction)"}
                                    {indexContentType === "text_pdfimages" && "Text files, PDF text, and PDF image extraction (no standalone images)"}
                                </p>
                            </div>

                            <div className="flex gap-2">
                                <button
                                    type="button"
                                    onClick={() => setShowNewIndexDialog(false)}
                                    className={`flex-1 px-4 py-2 text-sm border ${secondaryBtnBorder} rounded-md ${secondaryBtnHover}`}
                                    disabled={indexing}
                                >
                                    Cancel
                                </button>
                                <button
                                    type="submit"
                                    className={`flex-1 px-4 py-2 text-sm ${accentBtnBg} ${accentBtnText} rounded-md hover:opacity-90 disabled:opacity-50`}
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
                <div className={`fixed inset-0 ${overlayBg} backdrop-blur-sm flex items-center justify-center z-50`}>
                    <div className={`${modalBg} border ${modalBorder} rounded-lg max-w-2xl w-full mx-4 p-8 shadow-2xl`}>
                        <div className="mb-6">
                            <h2 className="text-xl font-medium mb-2">
                                {indexingComplete
                                    ? "Indexing Complete!"
                                    : "Indexing in progress"}
                            </h2>
                            <p className={`text-sm ${mutedText} font-mono truncate`}>
                                {indexProgress.indexPath}
                            </p>
                        </div>

                        {/* Progress Bar */}
                        {!indexingComplete && (
                            <>
                                <div className="mb-6">
                                    <div className="flex items-center justify-between text-sm mb-2">
                                        <span className={mutedText}>
                                            Processing files...
                                        </span>
                                        <span className="font-medium">
                                            {indexProgress.current} /{" "}
                                            {indexProgress.total}
                                        </span>
                                    </div>
                                    <div className={`w-full h-2 ${progressTrack} rounded-full overflow-hidden`}>
                                        <div
                                            className={`h-full ${progressFill} transition-all duration-300 ease-out`}
                                            style={{
                                                width: `${indexProgress.total > 0 ? (indexProgress.current / indexProgress.total) * 100 : 0}%`,
                                            }}
                                        />
                                    </div>
                                    <div className={`mt-2 text-xs ${subtleText}`}>
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
                                <div className={`${panelBg} rounded-lg p-4 border ${cardBorder} mb-6`}>
                                    <div className="flex items-start gap-3">
                                        <div className="mt-1">
                                            <div className={`w-2 h-2 ${progressDot} rounded-full animate-pulse`} />
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <div className={`text-xs ${mutedText} mb-1`}>
                                                {currentPhase || "Current file"}
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
                                    <CheckCircle className={`w-6 h-6 ${isDark ? "text-green-500" : "text-green-600"}`} />
                                    <span className="text-lg font-medium">
                                        Successfully indexed{" "}
                                        {indexProgress.total} files
                                    </span>
                                </div>
                                <p className={`text-sm ${mutedText}`}>
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
                                        disabled={cancelling}
                                        className={`flex-1 px-4 py-2.5 bg-red-500/10 hover:bg-red-500/20 border border-red-500/20 rounded-lg transition-colors ${isDark ? "text-red-400" : "text-red-600"} font-medium disabled:opacity-50 disabled:cursor-not-allowed`}
                                    >
                                        {cancelling ? "Cancelling..." : "Cancel Indexing"}
                                    </button>
                                    <div className={`flex-1 text-xs ${subtleText} flex items-center justify-center`}>
                                        This may take a few moments depending on
                                        folder size
                                    </div>
                                </>
                            ) : (
                                <button
                                    onClick={handleCloseProgressModal}
                                    className={`flex-1 px-4 py-2.5 ${accentBtnBg} ${accentBtnHover} ${accentBtnText} rounded-lg transition-colors font-medium`}
                                >
                                    Done
                                </button>
                            )}
                        </div>
                    </div>
                </div>
            )}

            {/* Update/Sync Progress Modal */}
            {updateProgress && (
                <div className={`fixed inset-0 ${overlayBg} backdrop-blur-sm flex items-center justify-center z-50`}>
                    <div className={`${modalBg} border ${modalBorder} rounded-lg max-w-2xl w-full mx-4 p-8 shadow-2xl`}>
                        <div className="mb-6">
                            <h2 className="text-xl font-medium mb-2">
                                {updateComplete
                                    ? "Sync Complete!"
                                    : "Syncing changes"}
                            </h2>
                            <p className={`text-sm ${mutedText} font-mono truncate`}>
                                {updateProgress.indexPath}
                            </p>
                        </div>

                        {/* Progress Bar */}
                        {!updateComplete && (
                            <>
                                <div className="mb-6">
                                    <div className="flex items-center justify-between text-sm mb-2">
                                        <span className={mutedText}>
                                            Processing files...
                                        </span>
                                        <span className="font-medium">
                                            {updateProgress.current} /{" "}
                                            {updateProgress.total}
                                        </span>
                                    </div>
                                    <div className={`w-full h-2 ${progressTrack} rounded-full overflow-hidden`}>
                                        <div
                                            className={`h-full ${progressFill} transition-all duration-300 ease-out`}
                                            style={{
                                                width: `${updateProgress.total > 0 ? (updateProgress.current / updateProgress.total) * 100 : 0}%`,
                                            }}
                                        />
                                    </div>
                                    <div className={`mt-2 text-xs ${subtleText}`}>
                                        {updateProgress.total > 0
                                            ? (
                                                (updateProgress.current /
                                                    updateProgress.total) *
                                                100
                                            ).toFixed(1)
                                            : "0.0"}
                                        % complete
                                    </div>
                                </div>

                                {/* Current File */}
                                <div className={`${panelBg} rounded-lg p-4 border ${cardBorder} mb-6`}>
                                    <div className="flex items-start gap-3">
                                        <div className="mt-1">
                                            <div className={`w-2 h-2 ${progressDot} rounded-full animate-pulse`} />
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <div className={`text-xs ${mutedText} mb-1`}>
                                                {currentPhase || "Current file"}
                                            </div>
                                            <div className="font-mono text-sm truncate">
                                                {updateProgress.file}
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </>
                        )}

                        {/* Completion Message */}
                        {updateComplete && (
                            <div className="bg-green-500/10 border border-green-500/20 rounded-lg p-6 mb-6">
                                <div className="flex items-center gap-3 mb-2">
                                    <CheckCircle className={`w-6 h-6 ${isDark ? "text-green-500" : "text-green-600"}`} />
                                    <span className="text-lg font-medium">
                                        Successfully synced{" "}
                                        {updateProgress.total} file
                                        {updateProgress.total !== 1 ? "s" : ""}
                                    </span>
                                </div>
                                <p className={`text-sm ${mutedText}`}>
                                    Index is now up to date
                                </p>
                            </div>
                        )}

                        {/* Action Buttons */}
                        <div className="flex gap-3">
                            {updateComplete ? (
                                <button
                                    onClick={() => {
                                        setUpdateProgress(null);
                                        setUpdateComplete(false);
                                    }}
                                    className={`flex-1 px-4 py-2.5 ${accentBtnBg} ${accentBtnHover} ${accentBtnText} rounded-lg transition-colors font-medium`}
                                >
                                    Done
                                </button>
                            ) : (
                                <div className={`flex-1 text-xs ${subtleText} flex items-center justify-center`}>
                                    Updating index with changed files...
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            )}

            {/* Full Text Modal */}
            {showFullTextModal && selectedResult && (
                <div className={`fixed inset-0 ${overlayBg} backdrop-blur-sm flex items-center justify-center z-50 p-4`}>
                    <div className={`${modalBg} border ${modalBorder} rounded-lg max-w-4xl w-full max-h-[90vh] flex flex-col shadow-2xl`}>
                        {/* Modal Header */}
                        <div className={`flex items-start justify-between p-6 border-b ${cardBorder}`}>
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
                                <div className={`flex items-center gap-4 text-xs ${mutedText}`}>
                                    {selectedResult.pageNumbers &&
                                        selectedResult.pageNumbers.length >
                                        0 && (
                                            <span className={`bg-blue-500/10 px-2 py-1 rounded border border-blue-500/20 ${isDark ? "text-blue-400" : "text-blue-600"}`}>
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
                                    <span className={`px-2 py-1 rounded ${badgeBg}`}>
                                        {(() => {
                                            // Handle reranked results (negative values are relevance scores)
                                            if (selectedResult.similarity < 0) {
                                                return (Math.abs(selectedResult.similarity) * 100).toFixed(0);
                                            }
                                            // Handle distance metrics (0-1 range)
                                            return ((1 - selectedResult.similarity) * 100).toFixed(0);
                                        })()}
                                        % match
                                    </span>
                                </div>
                            </div>
                            <button
                                onClick={() => {
                                    setShowFullTextModal(false);
                                    setSelectedResult(null);
                                }}
                                className={`${iconColor} ${iconHoverColor} transition-colors`}
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
                                        className={`max-w-full h-auto rounded-lg border ${cardBorder} shadow-lg`}
                                    />
                                    <p className={`text-sm ${mutedText} mt-4 italic text-center`}>
                                        {selectedResult.content}
                                    </p>
                                </div>
                            ) : (
                                <div className={`prose ${isDark ? "prose-invert" : ""} max-w-none`}>
                                    <div className={`text-sm leading-relaxed ${bodyText} whitespace-pre-wrap font-mono ${panelBg} p-4 rounded-lg border ${cardBorder}`}>
                                        {selectedResult.content}
                                    </div>
                                </div>
                            )}
                        </div>

                        {/* Modal Footer */}
                        <div className={`border-t ${cardBorder} p-6 ${cardBg}`}>
                            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4 text-xs">
                                {selectedResult.fileSize && (
                                    <div>
                                        <div className={`${subtleText} mb-1`}>
                                            File Size
                                        </div>
                                        <div className={`flex items-center gap-1.5 ${bodyText}`}>
                                            <HardDrive className="w-3 h-3" />
                                            {formatFileSize(
                                                selectedResult.fileSize,
                                            )}
                                        </div>
                                    </div>
                                )}
                                {selectedResult.modifiedDate && (
                                    <div>
                                        <div className={`${subtleText} mb-1`}>
                                            Modified
                                        </div>
                                        <div className={`flex items-center gap-1.5 ${bodyText}`}>
                                            <Clock className="w-3 h-3" />
                                            {selectedResult.modifiedDate}
                                        </div>
                                    </div>
                                )}
                                <div className="col-span-2">
                                    <div className={`${subtleText} mb-1`}>
                                        File Path
                                    </div>
                                    <div className={`flex items-center gap-1.5 ${bodyText} font-mono text-xs`}>
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
                                    className={`flex items-center gap-2 px-4 py-2 ${accentBtnBg} ${accentBtnText} rounded-md hover:opacity-90 text-sm font-medium`}
                                >
                                    <ExternalLink className="w-4 h-4" />
                                    Open File
                                </button>
                                <button
                                    onClick={() =>
                                        handleOpenFileLocation(
                                            selectedResult.filePath,
                                        )
                                    }
                                    className={`flex items-center gap-2 px-4 py-2 border ${secondaryBtnBorder} rounded-md ${secondaryBtnHover} text-sm font-medium`}
                                >
                                    <Folder className="w-4 h-4" />
                                    Show in Explorer
                                </button>
                                <button
                                    onClick={() => {
                                        setShowFullTextModal(false);
                                        setSelectedResult(null);
                                    }}
                                    className={`flex-1 px-4 py-2 border ${secondaryBtnBorder} rounded-md ${secondaryBtnHover} text-sm`}
                                >
                                    Close
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {/* PDF Preview Modal */}
            {showPDFPreview && pdfPreviewData && (
                <div className={`fixed inset-0 ${overlayBg} backdrop-blur-sm flex items-center justify-center z-50 p-4`}>
                    <div className={`${modalBg} border ${modalBorder} rounded-lg w-full max-w-6xl max-h-[95vh] flex flex-col shadow-2xl`}>
                        {/* Modal Header */}
                        <div className={`flex items-center justify-between p-4 border-b ${cardBorder} flex-shrink-0`}>
                            <div className="flex-1 min-w-0 pr-4">
                                <h2 className="text-lg font-medium flex items-center gap-2 truncate">
                                    <FileText className="w-5 h-5 flex-shrink-0" />
                                    <span className="truncate">
                                        {pdfPreviewData.fileName}
                                    </span>
                                </h2>
                                {pdfNumPages > 0 && (
                                    <div className={`text-xs ${mutedText} mt-1`}>
                                        {pdfNumPages} pages
                                    </div>
                                )}
                            </div>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={() =>
                                        setPdfScale(
                                            Math.max(0.5, pdfScale - 0.25),
                                        )
                                    }
                                    className={`p-2 ${hoverBgActive} rounded transition-colors`}
                                    title="Zoom out"
                                >
                                    <ZoomOut className="w-4 h-4" />
                                </button>
                                <span className={`text-xs ${mutedText} min-w-[3rem] text-center`}>
                                    {Math.round(pdfScale * 100)}%
                                </span>
                                <button
                                    onClick={() =>
                                        setPdfScale(
                                            Math.min(2.0, pdfScale + 0.25),
                                        )
                                    }
                                    className={`p-2 ${hoverBgActive} rounded transition-colors`}
                                    title="Zoom in"
                                >
                                    <ZoomIn className="w-4 h-4" />
                                </button>
                                <button
                                    onClick={() =>
                                        window.open(
                                            "/api/file?path=" +
                                            encodeURIComponent(
                                                pdfPreviewData.filePath,
                                            ),
                                            "_blank",
                                        )
                                    }
                                    className={`p-2 ${hoverBgActive} rounded transition-colors`}
                                    title="Open in new tab"
                                >
                                    <ExternalLink className="w-4 h-4" />
                                </button>
                                <a
                                    href={
                                        "/api/file?path=" +
                                        encodeURIComponent(
                                            pdfPreviewData.filePath,
                                        )
                                    }
                                    download
                                    className={`p-2 ${hoverBgActive} rounded transition-colors`}
                                    title="Download PDF"
                                >
                                    <Download className="w-4 h-4" />
                                </a>
                                <button
                                    onClick={closePDFPreview}
                                    className={`p-2 ${hoverBgActive} rounded transition-colors`}
                                >
                                    <X className="w-5 h-5" />
                                </button>
                            </div>
                        </div>

                        {/* PDF Viewer */}
                        <div className={`flex-1 overflow-auto ${pdfViewerBg} flex items-center justify-center p-4`}>
                            <Document
                                file={
                                    "/api/file?path=" +
                                    encodeURIComponent(pdfPreviewData.filePath)
                                }
                                onLoadSuccess={onPDFLoadSuccess}
                                loading={
                                    <div className={`${mutedText} py-20`}>
                                        Loading PDF...
                                    </div>
                                }
                                error={
                                    <div className={`${isDark ? "text-red-400" : "text-red-600"} py-20`}>
                                        Failed to load PDF
                                    </div>
                                }
                            >
                                <Page
                                    pageNumber={pdfPageNumber}
                                    scale={pdfScale}
                                    renderTextLayer={true}
                                    renderAnnotationLayer={true}
                                    className="shadow-lg"
                                />
                            </Document>
                        </div>

                        {/* Page Navigation Footer */}
                        <div className={`flex items-center justify-between p-4 border-t ${cardBorder} ${headerBg} flex-shrink-0`}>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={() =>
                                        setPdfPageNumber(
                                            Math.max(1, pdfPageNumber - 1),
                                        )
                                    }
                                    disabled={pdfPageNumber <= 1}
                                    className={`p-2 ${hoverBgActive} rounded transition-colors disabled:opacity-30 disabled:cursor-not-allowed`}
                                >
                                    <ChevronLeft className="w-4 h-4" />
                                </button>
                                <div className="flex items-center gap-2 text-sm">
                                    <span className={mutedText}>Page</span>
                                    <input
                                        type="number"
                                        min="1"
                                        max={pdfNumPages}
                                        value={pdfPageNumber}
                                        onChange={(e) => {
                                            const page = parseInt(
                                                e.target.value,
                                            );
                                            if (
                                                page >= 1 &&
                                                page <= pdfNumPages
                                            ) {
                                                setPdfPageNumber(page);
                                            }
                                        }}
                                        className={`w-16 px-2 py-1 ${inputBg} border ${cardBorder} rounded text-center ${focusBorder} outline-none`}
                                    />
                                    <span className={mutedText}>
                                        of {pdfNumPages}
                                    </span>
                                </div>
                                <button
                                    onClick={() =>
                                        setPdfPageNumber(
                                            Math.min(
                                                pdfNumPages,
                                                pdfPageNumber + 1,
                                            ),
                                        )
                                    }
                                    disabled={pdfPageNumber >= pdfNumPages}
                                    className={`p-2 ${hoverBgActive} rounded transition-colors disabled:opacity-30 disabled:cursor-not-allowed`}
                                >
                                    <ChevronRightIcon className="w-4 h-4" />
                                </button>
                            </div>
                            <button
                                onClick={closePDFPreview}
                                className={`px-4 py-2 border ${secondaryBtnBorder} rounded-md ${secondaryBtnHover} text-sm`}
                            >
                                Close
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* Terminal Component */}
            {showTerminal && (
                <div
                    className={`fixed bottom-0 left-0 right-0 ${bgColor} border-t ${borderColor} shadow-2xl z-50`}
                    style={{ height: terminalHeight }}
                >
                    {/* Terminal Header */}
                    <div className={`flex items-center justify-between px-4 py-2 border-b ${borderColor} ${isDark ? "bg-gray-900" : "bg-gray-100"}`}>
                        <div className="flex items-center gap-2">
                            <TerminalIcon className="w-4 h-4" />
                            <span className="text-sm font-medium">Terminal</span>
                            <span className={`text-xs ${subtleText}`}>
                                ({terminalLogs.length} logs)
                            </span>
                        </div>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={() => setTerminalLogs([])}
                                className={`px-2 py-1 text-xs ${hoverBg} rounded transition-colors`}
                                title="Clear terminal"
                            >
                                Clear
                            </button>
                            <button
                                onClick={() => setShowTerminal(false)}
                                className={`p-1 ${hoverBg} rounded transition-colors`}
                            >
                                <ChevronDown className="w-4 h-4" />
                            </button>
                        </div>
                    </div>

                    {/* Resize Handle */}
                    <div
                        className={`absolute top-0 left-0 right-0 h-1 cursor-row-resize ${hoverBg} transition-colors resize-handle`}
                        onMouseDown={handleTerminalMouseDown}
                    />

                    {/* Terminal Content */}
                    <div ref={terminalRef} className={`h-full overflow-auto p-4 font-mono text-sm custom-scrollbar ${isDark ? "bg-gray-950" : "bg-gray-50"}`}>
                        {terminalLogs.length === 0 ? (
                            <p className={subtleText}>
                                Waiting for terminal output...
                            </p>
                        ) : (
                            <div>
                                {terminalLogs.map((log, index) => {
                                    const getLogColor = (level: string) => {
                                        switch (level) {
                                            case "error":
                                                return isDark ? "text-red-400" : "text-red-700";
                                            case "warning":
                                                return isDark ? "text-yellow-400" : "text-yellow-700";
                                            case "success":
                                                return isDark ? "text-green-400" : "text-green-700";
                                            case "info":
                                            default:
                                                return isDark ? "text-gray-300" : "text-gray-700";
                                        }
                                    };

                                    const timestamp = new Date(log.timestamp).toLocaleTimeString();
                                    
                                    return (
                                        <p key={index} className={getLogColor(log.level)}>
                                            <span className={subtleText}>
                                                [{timestamp}]
                                            </span>{" "}
                                            {log.message}
                                        </p>
                                    );
                                })}
                            </div>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}

export default App;
