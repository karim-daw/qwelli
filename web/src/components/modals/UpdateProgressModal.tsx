import { CheckCircle } from "lucide-react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { ProgressBar } from "./ProgressBar";
import type { IndexProgress } from "@/types/progress";

interface UpdateProgressModalProps {
    progress: IndexProgress;
    phase: string;
    isComplete: boolean;
    onClose: () => void;
}

export function UpdateProgressModal({
    progress,
    phase,
    isComplete,
    onClose,
}: UpdateProgressModalProps) {
    return (
        <Dialog open onOpenChange={() => isComplete && onClose()}>
            <DialogContent className="max-w-2xl" onInteractOutside={(e) => e.preventDefault()}>
                <DialogHeader>
                    <DialogTitle>
                        {isComplete ? "Sync Complete!" : "Syncing changes"}
                    </DialogTitle>
                    <DialogDescription className="font-mono truncate">
                        {progress.indexPath}
                    </DialogDescription>
                </DialogHeader>

                {!isComplete && <ProgressBar progress={progress} phase={phase} />}

                {isComplete && (
                    <div className="bg-green-500/10 border border-green-500/20 rounded-lg p-6 mb-6">
                        <div className="flex items-center gap-3 mb-2">
                            <CheckCircle className="w-6 h-6 text-green-600 dark:text-green-500" />
                            <span className="text-lg font-medium">
                                Successfully synced {progress.total} file
                                {progress.total !== 1 ? "s" : ""}
                            </span>
                        </div>
                        <p className="text-sm text-muted-foreground">
                            Index is now up to date
                        </p>
                    </div>
                )}

                <div className="flex gap-3">
                    {isComplete ? (
                        <Button onClick={onClose} className="flex-1">
                            Done
                        </Button>
                    ) : (
                        <div className="flex-1 text-xs text-muted-foreground flex items-center justify-center">
                            Updating index with changed files...
                        </div>
                    )}
                </div>
            </DialogContent>
        </Dialog>
    );
}
