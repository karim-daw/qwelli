import { useEffect, useRef } from "react";

interface UseSSEOptions {
    url: string;
    onMessage: (data: unknown) => void;
    onError?: () => void;
    enabled?: boolean;
    reconnect?: boolean;
    reconnectInterval?: number;
}

export function useSSE(options: UseSSEOptions) {
    const { url, onMessage, onError, enabled = true, reconnect = false, reconnectInterval = 2000 } = options;
    const onMessageRef = useRef(onMessage);
    const onErrorRef = useRef(onError);

    onMessageRef.current = onMessage;
    onErrorRef.current = onError;

    useEffect(() => {
        if (!enabled) return;

        let eventSource: EventSource | null = null;
        let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
        let isMounted = true;

        const connect = () => {
            if (!isMounted) return;

            eventSource = new EventSource(url);

            eventSource.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    onMessageRef.current(data);
                } catch {
                    // ignore parse errors
                }
            };

            eventSource.onerror = () => {
                if (eventSource) eventSource.close();
                onErrorRef.current?.();
                if (reconnect && isMounted) {
                    reconnectTimer = setTimeout(connect, reconnectInterval);
                }
            };
        };

        connect();

        return () => {
            isMounted = false;
            if (eventSource) eventSource.close();
            if (reconnectTimer) clearTimeout(reconnectTimer);
        };
    }, [url, enabled, reconnect, reconnectInterval]);
}
