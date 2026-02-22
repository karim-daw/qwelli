import { useState, useCallback, useRef } from "react";
import type { IndexProgress, SSEMessage } from "@/types/progress";
import * as indexesApi from "@/api/indexes";
import { toast } from "sonner";

export function useIndexProgress() {
    const [progress, setProgress] = useState<IndexProgress | null>(null);
    const [phase, setPhase] = useState("");
    const [isComplete, setIsComplete] = useState(false);
    const [isRunning, setIsRunning] = useState(false);
    const [cancelling, setCancelling] = useState(false);
    const eventSourceRef = useRef<EventSource | null>(null);

    const start = useCallback(
        (
            indexPath: string,
            options: {
                onComplete?: () => void;
                onCancelled?: () => void;
            } = {},
        ) => {
            setIsRunning(true);
            setIsComplete(false);
            setPhase("");

            const eventSource = new EventSource(
                "/api/index/progress?path=" + encodeURIComponent(indexPath),
            );
            eventSourceRef.current = eventSource;

            eventSource.onmessage = (event) => {
                const data = JSON.parse(event.data) as SSEMessage;

                switch (data.type) {
                    case "progress":
                        setProgress({
                            current: data.current,
                            total: data.total,
                            file: data.file,
                            indexPath,
                        });
                        break;
                    case "phase":
                        setPhase(data.message || data.phase || "");
                        break;
                    case "complete":
                        eventSource.close();
                        setIsComplete(true);
                        setIsRunning(false);
                        setPhase("");
                        options.onComplete?.();
                        break;
                    case "cancelled":
                        eventSource.close();
                        setProgress(null);
                        setIsRunning(false);
                        setIsComplete(false);
                        setCancelling(false);
                        setPhase("");
                        options.onCancelled?.();
                        break;
                    case "error":
                        eventSource.close();
                        setProgress(null);
                        setIsRunning(false);
                        setIsComplete(false);
                        setPhase("");
                        toast.error(data.message);
                        break;
                }
            };

            eventSource.onerror = () => {
                eventSource.close();
                setProgress(null);
                setIsRunning(false);
            };
        },
        [],
    );

    const cancel = useCallback(async () => {
        if (!progress || cancelling) return;
        setCancelling(true);
        try {
            await indexesApi.cancelIndex(progress.indexPath);
        } catch (error) {
            toast.error("Failed to cancel: " + String(error));
            setCancelling(false);
        }
    }, [progress, cancelling]);

    const reset = useCallback(() => {
        if (eventSourceRef.current) {
            eventSourceRef.current.close();
        }
        setProgress(null);
        setPhase("");
        setIsComplete(false);
        setIsRunning(false);
        setCancelling(false);
    }, []);

    return {
        progress,
        phase,
        isComplete,
        isRunning,
        cancelling,
        start,
        cancel,
        reset,
    };
}
