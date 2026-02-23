import { useState, useCallback, useRef, useEffect } from "react";
import { useSSE } from "./useSSE";
import type { TerminalLog } from "@/types/terminal";

const MAX_TERMINAL_LOGS = 500;

export function useTerminal() {
    const [logs, setLogs] = useState<TerminalLog[]>([]);
    const idRef = useRef(0);
    const pendingRef = useRef<TerminalLog[]>([]);

    const handleMessage = useCallback((data: unknown) => {
        const msg = data as { type?: string; level?: string; message?: string };
        if (msg.type === "log") {
            pendingRef.current.push({
                id: ++idRef.current,
                type: msg.type || "log",
                level: msg.level || "info",
                message: msg.message || "",
                timestamp: Date.now(),
            });
        }
    }, []);

    useEffect(() => {
        const interval = setInterval(() => {
            if (pendingRef.current.length === 0) return;
            const batch = pendingRef.current;
            pendingRef.current = [];
            setLogs((prev) => {
                const next = [...prev, ...batch];
                return next.length > MAX_TERMINAL_LOGS
                    ? next.slice(-MAX_TERMINAL_LOGS)
                    : next;
            });
        }, 100);
        return () => clearInterval(interval);
    }, []);

    useSSE({
        url: "/api/terminal/stream",
        onMessage: handleMessage,
        reconnect: true,
        reconnectInterval: 2000,
    });

    const clearLogs = useCallback(() => {
        setLogs([]);
        pendingRef.current = [];
    }, []);

    return { logs, clearLogs };
}
