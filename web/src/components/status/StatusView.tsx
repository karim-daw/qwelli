import { useState, useEffect } from "react";
import { RefreshCw, Plus, AlertCircle, X, CheckCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { StatusSummaryGrid } from "./StatusSummaryGrid";
import { FileChangeList } from "./FileChangeList";
import { UpdateProgressModal } from "@/components/modals/UpdateProgressModal";
import { useAppContext } from "@/contexts/AppContext";
import { useIndexProgress } from "@/hooks/useIndexProgress";
import * as indexesApi from "@/api/indexes";
import type { IndexStatus } from "@/types/index";
import { toast } from "sonner";

export function StatusView() {
    const { selectedIndex } = useAppContext();
    const [indexStatus, setIndexStatus] = useState<IndexStatus | null>(null);
    const [loadingStatus, setLoadingStatus] = useState(false);
    const updateProgress = useIndexProgress();

    useEffect(() => {
        if (selectedIndex) {
            setLoadingStatus(true);
            indexesApi
                .getIndexStatus(selectedIndex)
                .then(setIndexStatus)
                .catch((error) => console.error("Failed to fetch status:", error))
                .finally(() => setLoadingStatus(false));
        }
    }, [selectedIndex]);

    const handleSync = async () => {
        updateProgress.start(selectedIndex, {
            onComplete: () => {
                indexesApi
                    .getIndexStatus(selectedIndex)
                    .then(setIndexStatus)
                    .catch((error) =>
                        console.error("Failed to refresh status:", error),
                    );
            },
        });

        try {
            const response = await indexesApi.updateIndex(selectedIndex);
            if (!response.ok) {
                const data = await response.json();
                updateProgress.reset();
                toast.error("Error: " + data.error);
            }
        } catch (error) {
            updateProgress.reset();
            toast.error("Failed to sync: " + String(error));
        }
    };

    if (loadingStatus) {
        return (
            <div className="text-center text-muted-foreground py-20">
                Loading status...
            </div>
        );
    }

    if (!indexStatus) {
        return (
            <div className="text-center text-muted-foreground py-20">
                Select an index to view status
            </div>
        );
    }

    const hasChanges =
        (indexStatus.toAdd?.length || 0) > 0 ||
        (indexStatus.toUpdate?.length || 0) > 0 ||
        (indexStatus.toDelete?.length || 0) > 0;

    return (
        <div className="space-y-6">
            {/* Action Bar */}
            <div className="flex items-center justify-between">
                <h2 className="text-lg font-medium">Index Status</h2>
                {hasChanges && (
                    <Button
                        onClick={handleSync}
                        disabled={updateProgress.isRunning}
                    >
                        <RefreshCw
                            className={`w-4 h-4 mr-2 ${updateProgress.isRunning ? "animate-spin" : ""}`}
                        />
                        {updateProgress.isRunning ? "Syncing..." : "Sync Changes"}
                    </Button>
                )}
            </div>

            {/* Status Summary */}
            <StatusSummaryGrid status={indexStatus} />

            {/* Change Details */}
            <FileChangeList
                files={indexStatus.toAdd}
                icon={Plus}
                label="New Files"
                colorClass="text-blue-600 dark:text-blue-400"
            />
            <FileChangeList
                files={indexStatus.toUpdate}
                icon={AlertCircle}
                label="Modified Files"
                colorClass="text-yellow-600 dark:text-yellow-400"
            />
            <FileChangeList
                files={indexStatus.toDelete}
                icon={X}
                label="Deleted Files"
                colorClass="text-red-600 dark:text-red-400"
            />

            {/* All up to date message */}
            {!hasChanges && (
                <div className="text-center py-20">
                    <CheckCircle className="w-12 h-12 text-green-600 dark:text-green-400 mx-auto mb-3" />
                    <p className="text-muted-foreground">Index is up to date!</p>
                </div>
            )}

            {/* Update Progress Modal */}
            {updateProgress.progress && (
                <UpdateProgressModal
                    progress={updateProgress.progress}
                    phase={updateProgress.phase}
                    isComplete={updateProgress.isComplete}
                    onClose={() => updateProgress.reset()}
                />
            )}
        </div>
    );
}
