import { api } from "./client";
import type { ChatEvent } from "@/types/chat";

export async function sendMessage(
    index: string,
    message: string,
    onEvent: (event: ChatEvent) => void,
    signal?: AbortSignal,
): Promise<void> {
    const response = await fetch("/api/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ index, message }),
        signal,
    });

    if (!response.ok) {
        let errorMsg = `Chat request failed with status ${response.status}`;
        try {
            const data = await response.json();
            if (data.error) errorMsg = data.error;
        } catch {
            // ignore parse failure
        }
        throw new Error(errorMsg);
    }

    const reader = response.body?.getReader();
    if (!reader) throw new Error("No response body");

    const decoder = new TextDecoder();
    let buffer = "";

    try {
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split("\n");
            buffer = lines.pop() || "";

            for (const line of lines) {
                const trimmed = line.trim();
                if (!trimmed.startsWith("data: ")) continue;
                const json = trimmed.slice(6);
                if (!json) continue;
                try {
                    const event: ChatEvent = JSON.parse(json);
                    onEvent(event);
                } catch {
                    // ignore malformed events
                }
            }
        }
    } finally {
        reader.releaseLock();
    }
}

export async function clearHistory(index: string): Promise<void> {
    await api.post("/api/chat/clear", { index });
}
