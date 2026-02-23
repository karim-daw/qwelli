import { useEffect, useRef } from "react";
import { Terminal as TerminalIcon, ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { TerminalLog } from "@/types/terminal";

interface TerminalPanelProps {
    logs: TerminalLog[];
    height: number;
    onResizeStart: () => void;
    onClose: () => void;
    onClear: () => void;
}

function getLogColor(level: string): string {
    switch (level) {
        case "error":
            return "text-red-700 dark:text-red-400";
        case "warning":
            return "text-yellow-700 dark:text-yellow-400";
        case "success":
            return "text-green-700 dark:text-green-400";
        case "info":
        default:
            return "text-gray-700 dark:text-gray-300";
    }
}

export function TerminalPanel({
    logs,
    height,
    onResizeStart,
    onClose,
    onClear,
}: TerminalPanelProps) {
    const terminalRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (terminalRef.current) {
            terminalRef.current.scrollTop = terminalRef.current.scrollHeight;
        }
    }, [logs]);

    return (
        <div
            className="fixed bottom-0 left-0 right-0 bg-background border-t shadow-2xl z-50 flex flex-col"
            style={{ height }}
        >
            {/* Resize Handle */}
            <div
                className="absolute top-0 left-0 right-0 h-1 cursor-row-resize hover:bg-accent transition-colors resize-handle"
                onMouseDown={onResizeStart}
            />

            {/* Header */}
            <div className="flex items-center justify-between px-4 py-2 border-b bg-muted">
                <div className="flex items-center gap-2">
                    <TerminalIcon className="w-4 h-4" />
                    <span className="text-sm font-medium">Terminal</span>
                    <span className="text-xs text-muted-foreground">
                        ({logs.length} logs)
                    </span>
                </div>
                <div className="flex items-center gap-2">
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={onClear}
                        className="h-7 px-2 text-xs"
                    >
                        Clear
                    </Button>
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={onClose}
                        className="h-7 w-7"
                    >
                        <ChevronDown className="w-4 h-4" />
                    </Button>
                </div>
            </div>

            {/* Content */}
            <div
                ref={terminalRef}
                className="flex-1 min-h-0 overflow-auto p-4 font-mono text-sm custom-scrollbar bg-gray-50 dark:bg-gray-950"
            >
                {logs.length === 0 ? (
                    <p className="text-muted-foreground">
                        Waiting for terminal output...
                    </p>
                ) : (
                    <div>
                        {logs.map((log) => {
                            const timestamp = new Date(
                                log.timestamp,
                            ).toLocaleTimeString();
                            return (
                                <p key={log.id} className={getLogColor(log.level)}>
                                    <span className="text-muted-foreground">
                                        [{timestamp}]
                                    </span>{" "}
                                    {log.message}
                                </p>
                            );
                        })}
                    </div>
                )}
            </div>
        </div>
    );
}
