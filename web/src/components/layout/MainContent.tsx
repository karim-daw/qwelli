import { useState, useEffect } from "react";
import { Folder, Search } from "lucide-react";
import { SearchResults } from "@/components/search/SearchResults";
import { RecentSearches } from "@/components/search/RecentSearches";
import { StatusView } from "@/components/status/StatusView";
import { ChatView } from "@/components/chat/ChatView";
import { FullTextModal } from "@/components/modals/FullTextModal";
import { PDFPreviewModal } from "@/components/modals/PDFPreviewModal";
import { useAppContext } from "@/contexts/AppContext";
import { useSearchContext } from "@/contexts/SearchContext";
import { useSearch } from "@/hooks/useSearch";
import { usePDFPreview } from "@/hooks/usePDFPreview";
import type { SearchResult } from "@/types/search";

export function MainContent() {
    const { selectedIndex, viewMode } = useAppContext();
    const { query, results, loading, cacheStatus, recentSearches, loadRecentSearches, clearRecentSearches } =
        useSearchContext();
    const { loadCachedSearch } = useSearch();
    const pdfPreview = usePDFPreview();
    const [fullTextResult, setFullTextResult] = useState<SearchResult | null>(null);

    // Load recent searches from localStorage when selected index changes
    useEffect(() => {
        if (selectedIndex) {
            loadRecentSearches(selectedIndex);
        }
    }, [selectedIndex, loadRecentSearches]);

    if (viewMode === "status") {
        return (
            <div className="flex-1 overflow-y-auto">
                <div className="max-w-5xl mx-auto px-6 py-6">
                    <StatusView />
                </div>
            </div>
        );
    }

    if (viewMode === "chat") {
        return <ChatView />;
    }

    return (
        <div className="flex-1 overflow-y-auto">
            <div className="max-w-5xl mx-auto px-6 py-6">
                {!selectedIndex ? (
                    <div className="text-center text-muted-foreground py-20">
                        <Folder className="w-12 h-12 mx-auto mb-3 opacity-50" />
                        <p>Select an index from the sidebar to start</p>
                    </div>
                ) : loading ? (
                    <div className="text-center text-muted-foreground py-20">
                        Searching...
                    </div>
                ) : results.length > 0 ? (
                    <SearchResults
                        results={results}
                        cacheStatus={cacheStatus}
                        onViewFullText={setFullTextResult}
                        onOpenPDF={pdfPreview.open}
                    />
                ) : query ? (
                    <div className="text-center text-muted-foreground py-20">
                        <p>No results found for &ldquo;{query}&rdquo;</p>
                    </div>
                ) : recentSearches.length > 0 ? (
                    <RecentSearches
                        searches={recentSearches}
                        onLoadSearch={loadCachedSearch}
                        onClear={() => clearRecentSearches(selectedIndex)}
                    />
                ) : (
                    <div className="text-center text-muted-foreground py-20">
                        <Search className="w-12 h-12 mx-auto mb-3 opacity-50" />
                        <p>Start searching to see recent searches here</p>
                    </div>
                )}
            </div>

            {fullTextResult && (
                <FullTextModal
                    result={fullTextResult}
                    open={!!fullTextResult}
                    onClose={() => setFullTextResult(null)}
                />
            )}

            {pdfPreview.isOpen && pdfPreview.data && (
                <PDFPreviewModal
                    filePath={pdfPreview.data.filePath}
                    fileName={pdfPreview.data.fileName}
                    initialPage={pdfPreview.data.initialPage}
                    onClose={pdfPreview.close}
                />
            )}
        </div>
    );
}
