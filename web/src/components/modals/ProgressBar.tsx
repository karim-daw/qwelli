import type { IndexProgress } from "@/types/progress";

interface ProgressBarProps {
    progress: IndexProgress;
    phase: string;
}

export function ProgressBar({ progress, phase }: ProgressBarProps) {
    const percent =
        progress.total > 0 ? (progress.current / progress.total) * 100 : 0;

    return (
        <>
            <div className="mb-6">
                <div className="flex items-center justify-between text-sm mb-2">
                    <span className="text-muted-foreground">
                        Processing files...
                    </span>
                    <span className="font-medium">
                        {progress.current} / {progress.total}
                    </span>
                </div>
                <div className="w-full h-2 bg-secondary rounded-full overflow-hidden">
                    <div
                        className="h-full bg-primary transition-all duration-300 ease-out"
                        style={{ width: `${percent}%` }}
                    />
                </div>
                <div className="mt-2 text-xs text-muted-foreground">
                    {percent.toFixed(1)}% complete
                </div>
            </div>

            {/* Current File */}
            <div className="bg-muted rounded-lg p-4 border mb-6">
                <div className="flex items-start gap-3">
                    <div className="mt-1">
                        <div className="w-2 h-2 bg-primary rounded-full animate-pulse" />
                    </div>
                    <div className="flex-1 min-w-0">
                        <div className="text-xs text-muted-foreground mb-1">
                            {phase || "Current file"}
                        </div>
                        <div className="font-mono text-sm truncate">
                            {progress.file}
                        </div>
                    </div>
                </div>
            </div>
        </>
    );
}
