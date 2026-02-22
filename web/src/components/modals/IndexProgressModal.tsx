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

interface IndexProgressModalProps {
    progress: IndexProgress;
    phase: string;
    isComplete: boolean;
    cancelling: boolean;
    onCancel: () => void;
    onClose: () => void;
}

export function IndexProgressModal({
    progress,
    phase,
    isComplete,
    cancelling,
    onCancel,
    onClose,
}: IndexProgressModalProps) {
    return (
        <Dialog open onOpenChange={() => isComplete && onClose()}>
            <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto" onInteractOutside={(e) => e.preventDefault()}>
                <DialogHeader>
                    <DialogTitle>
                        {isComplete
                            ? "Indexing Complete!"
                            : "Indexing in progress"}
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
                                Successfully indexed {progress.total} files
                            </span>
                        </div>
                        <p className="text-sm text-muted-foreground">
                            Your files are now searchable
                        </p>
                    </div>
                )}

                <div className="flex gap-3">
                    {!isComplete ? (
                        <>
                            <Button
                                variant="destructive"
                                onClick={onCancel}
                                disabled={cancelling}
                                className="flex-1"
                            >
                                {cancelling
                                    ? "Cancelling..."
                                    : "Cancel Indexing"}
                            </Button>
                            <div className="flex-1 text-xs text-muted-foreground flex items-center justify-center">
                                This may take a few moments depending on folder
                                size
                            </div>
                        </>
                    ) : (
                        <Button onClick={onClose} className="flex-1">
                            Done
                        </Button>
                    )}
                </div>
            </DialogContent>
        </Dialog>
    );
}
