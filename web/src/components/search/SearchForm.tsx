import { Input } from "@/components/ui/input";
import { useSearchContext } from "@/contexts/SearchContext";
import { useAppContext } from "@/contexts/AppContext";
import { useSearch } from "@/hooks/useSearch";
import { cn } from "@/lib/utils";

export function SearchForm() {
    const { query, strategy, contentFilter, topK, setQuery, setStrategy, setContentFilter, setTopK } =
        useSearchContext();
    const { selectedIndex } = useAppContext();
    const { executeSearch } = useSearch();

    return (
        <form
            onSubmit={(e) => {
                e.preventDefault();
                executeSearch();
            }}
            className="relative"
        >
            <Input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search knowledge base..."
                disabled={!selectedIndex}
                className="w-full py-3.5 text-base rounded-xl"
            />

            {/* Inline Settings Controls */}
            <div className="mt-3 flex items-center gap-4 text-xs">
                {/* Search Strategy */}
                <div className="flex items-center gap-2">
                    <span className="text-muted-foreground">Strategy:</span>
                    <div className="flex gap-1">
                        {["semantic", "keyword", "hybrid"].map((s) => (
                            <button
                                key={s}
                                type="button"
                                onClick={() => setStrategy(s)}
                                className={cn(
                                    "px-2.5 py-1 rounded-md transition-all text-xs font-medium",
                                    strategy === s
                                        ? "bg-primary text-primary-foreground"
                                        : "text-muted-foreground hover:text-foreground hover:bg-accent",
                                )}
                            >
                                {s.charAt(0).toUpperCase() + s.slice(1)}
                            </button>
                        ))}
                    </div>
                </div>

                <div className="w-px h-4 bg-border" />

                {/* Content Type */}
                <div className="flex items-center gap-2">
                    <span className="text-muted-foreground">Type:</span>
                    <div className="flex gap-1">
                        {[
                            { value: "all", label: "All" },
                            { value: "text", label: "Text" },
                            { value: "image", label: "Images" },
                        ].map((c) => (
                            <button
                                key={c.value}
                                type="button"
                                onClick={() => setContentFilter(c.value)}
                                className={cn(
                                    "px-2.5 py-1 rounded-md transition-all text-xs font-medium",
                                    contentFilter === c.value
                                        ? "bg-primary text-primary-foreground"
                                        : "text-muted-foreground hover:text-foreground hover:bg-accent",
                                )}
                            >
                                {c.label}
                            </button>
                        ))}
                    </div>
                </div>

                <div className="w-px h-4 bg-border" />

                {/* Results Count */}
                <div className="flex items-center gap-2">
                    <span className="text-muted-foreground">Results:</span>
                    <div className="flex items-center gap-2">
                        <input
                            type="range"
                            min="5"
                            max="50"
                            step="5"
                            value={topK}
                            onChange={(e) => setTopK(parseInt(e.target.value))}
                            className="w-20 h-1 bg-secondary rounded-lg appearance-none cursor-pointer accent-primary"
                        />
                        <span className="text-muted-foreground min-w-[2rem] text-right">
                            {topK}
                        </span>
                    </div>
                </div>
            </div>
        </form>
    );
}
