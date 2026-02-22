import { Search, Clock, HardDrive } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { RecentSearch } from "@/types/search";
import { formatTimestamp } from "@/lib/format";

interface RecentSearchesProps {
    searches: RecentSearch[];
    onLoadSearch: (search: RecentSearch) => void;
    onClear: () => void;
}

export function RecentSearches({
    searches,
    onLoadSearch,
    onClear,
}: RecentSearchesProps) {
    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between pb-2 border-b">
                <h3 className="text-sm font-medium text-muted-foreground">
                    Recent Searches
                </h3>
                <button
                    onClick={onClear}
                    className="text-xs text-muted-foreground hover:text-foreground transition-colors"
                >
                    Clear all
                </button>
            </div>

            <div className="grid gap-3">
                {searches.map((search, idx) => (
                    <button
                        key={idx}
                        onClick={() => onLoadSearch(search)}
                        className="text-left border rounded-lg p-4 hover:border-border/80 hover:bg-accent/50 transition-all group"
                    >
                        <div className="flex items-start justify-between gap-4 mb-2">
                            <div className="flex-1 min-w-0">
                                <div className="flex items-center gap-2 mb-1">
                                    <Search className="w-4 h-4 text-muted-foreground flex-shrink-0" />
                                    <span className="font-medium truncate">
                                        {search.query}
                                    </span>
                                </div>
                                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                    <Badge variant="secondary" className="text-xs">
                                        {search.strategy}
                                    </Badge>
                                    {search.contentFilter !== "all" && (
                                        <Badge variant="secondary" className="text-xs">
                                            {search.contentFilter}
                                        </Badge>
                                    )}
                                    <span>{search.resultCount} results</span>
                                </div>
                            </div>

                            <div className="flex items-center gap-2 flex-shrink-0">
                                <div className="text-xs text-muted-foreground flex items-center gap-1">
                                    <Clock className="w-3 h-3" />
                                    {formatTimestamp(search.timestamp)}
                                </div>
                                <div className="text-xs bg-green-500/10 text-green-700 dark:text-green-400 px-2 py-1 rounded border border-green-500/20 flex items-center gap-1">
                                    <HardDrive className="w-3 h-3" />
                                    Cached
                                </div>
                            </div>
                        </div>
                    </button>
                ))}
            </div>
        </div>
    );
}
