import { ResultCard } from "./ResultCard";
import type { SearchResult } from "@/types/search";

interface SearchResultsProps {
    results: SearchResult[];
    cacheStatus: string | null;
    onViewFullText: (result: SearchResult) => void;
    onOpenPDF: (result: SearchResult) => void;
}

export function SearchResults({
    results,
    cacheStatus,
    onViewFullText,
    onOpenPDF,
}: SearchResultsProps) {
    return (
        <div className="space-y-4">
            {/* Results header with cache indicator */}
            <div className="flex items-center justify-between pb-2 border-b">
                <div className="text-sm text-muted-foreground">
                    Found {results.length} result
                    {results.length !== 1 ? "s" : ""}
                </div>
                {(cacheStatus === "HIT" || cacheStatus === "LOCAL") && (
                    <div className="flex items-center gap-1.5 text-xs text-green-700 dark:text-green-400 bg-green-500/10 px-2 py-1 rounded border border-green-500/20">
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
                        {cacheStatus === "LOCAL" ? "Cached Locally" : "Cached"}
                    </div>
                )}
            </div>
            {results.map((result, index) => (
                <ResultCard
                    key={result.chunkId + index}
                    result={result}
                    index={index}
                    onViewFullText={onViewFullText}
                    onOpenPDF={onOpenPDF}
                />
            ))}
        </div>
    );
}
