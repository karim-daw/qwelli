import { useState, useRef, useCallback } from "react";
import type { ChatMessage, ChatEvent } from "@/types/chat";
import * as chatApi from "@/api/chat";

let nextId = 0;
function genId() {
    return `msg-${++nextId}-${Date.now()}`;
}

export function useChat(indexPath: string) {
    const [messages, setMessages] = useState<ChatMessage[]>([]);
    const [isStreaming, setIsStreaming] = useState(false);
    const abortRef = useRef<AbortController | null>(null);

    const sendMessage = useCallback(
        async (text: string) => {
            if (!text.trim() || !indexPath || isStreaming) return;

            const userMsg: ChatMessage = {
                id: genId(),
                role: "user",
                content: text.trim(),
            };
            const assistantId = genId();
            const assistantMsg: ChatMessage = {
                id: assistantId,
                role: "assistant",
                content: "",
                toolCalls: [],
                isStreaming: true,
            };

            setMessages((prev) => [...prev, userMsg, assistantMsg]);
            setIsStreaming(true);

            const controller = new AbortController();
            abortRef.current = controller;

            const handleEvent = (event: ChatEvent) => {
                setMessages((prev) => {
                    const updated = [...prev];
                    const idx = updated.findIndex(
                        (m) => m.id === assistantId,
                    );
                    if (idx === -1) return prev;
                    const msg = { ...updated[idx] };
                    const toolCalls = [...(msg.toolCalls || [])];

                    switch (event.type) {
                        case "thinking":
                            msg.isStreaming = true;
                            break;
                        case "text_delta":
                            msg.content += event.text || "";
                            break;
                        case "tool_call":
                            toolCalls.push({
                                name: event.tool_name || "",
                                input: event.tool_input || {},
                                isRunning: true,
                            });
                            msg.toolCalls = toolCalls;
                            break;
                        case "tool_result": {
                            const last = toolCalls.length - 1;
                            if (last >= 0) {
                                toolCalls[last] = {
                                    ...toolCalls[last],
                                    result: event.tool_result,
                                    isError: event.is_error,
                                    isRunning: false,
                                };
                                msg.toolCalls = toolCalls;
                            }
                            break;
                        }
                        case "done":
                            msg.isStreaming = false;
                            break;
                        case "error":
                            msg.isStreaming = false;
                            if (!msg.content) {
                                msg.content =
                                    event.text || "An error occurred.";
                            }
                            break;
                    }

                    updated[idx] = msg;
                    return updated;
                });
            };

            try {
                await chatApi.sendMessage(
                    indexPath,
                    text.trim(),
                    handleEvent,
                    controller.signal,
                );
            } catch (err) {
                if ((err as Error).name !== "AbortError") {
                    setMessages((prev) => {
                        const updated = [...prev];
                        const idx = updated.findIndex(
                            (m) => m.id === assistantId,
                        );
                        if (idx === -1) return prev;
                        updated[idx] = {
                            ...updated[idx],
                            isStreaming: false,
                            content:
                                updated[idx].content ||
                                (err as Error).message ||
                                "An error occurred.",
                        };
                        return updated;
                    });
                }
            } finally {
                setIsStreaming(false);
                abortRef.current = null;
            }
        },
        [indexPath, isStreaming],
    );

    const cancelStream = useCallback(() => {
        abortRef.current?.abort();
        setIsStreaming(false);
        setMessages((prev) => {
            const updated = [...prev];
            const last = updated.length - 1;
            if (last >= 0 && updated[last].role === "assistant") {
                updated[last] = { ...updated[last], isStreaming: false };
            }
            return updated;
        });
    }, []);

    const clearHistory = useCallback(async () => {
        try {
            await chatApi.clearHistory(indexPath);
        } catch {
            // ignore — server may have already cleared
        }
        setMessages([]);
    }, [indexPath]);

    return {
        messages,
        isStreaming,
        sendMessage,
        cancelStream,
        clearHistory,
    };
}
