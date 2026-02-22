import type { IndexStatus } from "@/types/index";

interface StatusSummaryGridProps {
    status: IndexStatus;
}

export function StatusSummaryGrid({ status }: StatusSummaryGridProps) {
    return (
        <div className="grid grid-cols-4 gap-4">
            <div className="bg-muted rounded-lg p-4 border">
                <div className="text-2xl font-semibold mb-1">{status.total}</div>
                <div className="text-xs text-muted-foreground">Total Files</div>
            </div>
            <div className="bg-green-500/10 rounded-lg p-4 border border-green-500/20">
                <div className="text-2xl font-semibold text-green-600 dark:text-green-400 mb-1">
                    {status.upToDate}
                </div>
                <div className="text-xs text-muted-foreground">Up to Date</div>
            </div>
            <div className="bg-yellow-500/10 rounded-lg p-4 border border-yellow-500/20">
                <div className="text-2xl font-semibold text-yellow-600 dark:text-yellow-400 mb-1">
                    {status.toUpdate?.length || 0}
                </div>
                <div className="text-xs text-muted-foreground">Modified</div>
            </div>
            <div className="bg-blue-500/10 rounded-lg p-4 border border-blue-500/20">
                <div className="text-2xl font-semibold text-blue-600 dark:text-blue-400 mb-1">
                    {status.toAdd?.length || 0}
                </div>
                <div className="text-xs text-muted-foreground">New Files</div>
            </div>
        </div>
    );
}
