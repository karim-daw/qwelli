import { useState, useCallback } from "react";
import { useSSE } from "./useSSE";
import type { TerminalLog } from "@/types/terminal";

const MAX_TERMINAL_LOGS = 500;

export function useTerminal() {
    const [logs, setLogs] = useState<TerminalLog[]>([]);

    const handleMessage = useCallback((data: unknown) => {
        const msg = data as { type?: string; level?: string; message?: string };
        if (msg.type === "log") {
            setLogs((prev) => {
                const next = [
                    ...prev,
                    {
                        type: msg.type || "log",
                        level: msg.level || "info",
                        message: msg.message || "",
                        timestamp: Date.now(),
                    },
                ];
                return next.length > MAX_TERMINAL_LOGS
                    ? next.slice(-MAX_TERMINAL_LOGS)
                    : next;
            });
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
