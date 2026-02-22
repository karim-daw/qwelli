import { useState, useCallback } from "react";
import { useSSE } from "./useSSE";
import type { TerminalLog } from "@/types/terminal";

export function useTerminal() {
    const [logs, setLogs] = useState<TerminalLog[]>([]);

    const handleMessage = useCallback((data: unknown) => {
        const msg = data as { type?: string; level?: string; message?: string };
        if (msg.type === "log") {
            setLogs((prev) => [
                ...prev,
                {
                    type: msg.type || "log",
                    level: msg.level || "info",
                    message: msg.message || "",
                    timestamp: Date.now(),
                },
            ]);
        }
    }, []);

    useSSE({
        url: "/api/terminal/stream",
        onMessage: handleMessage,
        reconnect: true,
        reconnectInterval: 2000,
    });

    const clearLogs = useCallback(() => {
        setLogs([]);
    }, []);

    return { logs, clearLogs };
}
