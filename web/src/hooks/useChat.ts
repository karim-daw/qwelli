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
                steps: [],
            };
            const assistantId = genId();
            const assistantMsg: ChatMessage = {
                id: assistantId,
                role: "assistant",
                content: "",
                steps: [{}],
                isStreaming: true,
            };

            setMessages((prev) => [...prev, userMsg, assistantMsg]);
            setIsStreaming(true);

            const controller = new AbortController();
            abortRef.current = controller;

            const handleEvent = (event: ChatEvent) => {
                setMessages((prev) => {
                    const idx = prev.length - 1;
                    if (idx < 0 || prev[idx].id !== assistantId) return prev;

                    const msg = { ...prev[idx] };
                    const steps = [...msg.steps];
                    const last = steps.length - 1;

                    switch (event.type) {
                        case "thinking": {
                            // If the current step already has tool calls, the agent
                            // has started a new iteration — open a new step for it.
                            if (steps[last].toolCalls?.length) {
                                steps.push({ thinking: event.text || "" });
                            } else {
                                steps[last] = {
                                    ...steps[last],
                                    thinking:
                                        (steps[last].thinking || "") +
                                        (event.text || ""),
                                };
                            }
                            msg.steps = steps;
                            break;
                        }
                        case "text_delta": {
                            const lastStep = steps[last];
                            // If the current step's tools are all done, this text
                            // belongs to a new iteration — start a new step.
                            const postTools =
                                !!lastStep.toolCalls?.length &&
                                !lastStep.toolCalls.some((tc) => tc.isRunning);
                            if (postTools) {
                                steps.push({ text: event.text || "" });
                            } else {
                                steps[last] = {
                                    ...steps[last],
                                    text:
                                        (steps[last].text || "") +
                                        (event.text || ""),
                                };
                            }
                            msg.steps = steps;
                            break;
                        }
                        case "tool_call": {
                            const prevCalls = steps[last].toolCalls;
                            // If every tool in the current step is already done, this
                            // tool_call belongs to a new agent iteration — start a new step.
                            const prevBatchDone =
                                !!prevCalls?.length &&
                                !prevCalls.some((tc) => tc.isRunning);
                            if (prevBatchDone) {
                                steps.push({
                                    toolCalls: [
                                        {
                                            id: event.tool_call_id,
                                            name: event.tool_name || "",
                                            input: event.tool_input || {},
                                            isRunning: true,
                                        },
                                    ],
                                });
                            } else {
                                const toolCalls = [...(prevCalls || [])];
                                toolCalls.push({
                                    id: event.tool_call_id,
                                    name: event.tool_name || "",
                                    input: event.tool_input || {},
                                    isRunning: true,
                                });
                                steps[last] = { ...steps[last], toolCalls };
                            }
                            msg.steps = steps;
                            break;
                        }
                        case "tool_result": {
                            // Search from the end — results always belong to
                            // the most recent step that has a running tool with this ID.
                            for (let si = steps.length - 1; si >= 0; si--) {
                                const tcs = steps[si].toolCalls;
                                if (!tcs) continue;
                                const ti = event.tool_call_id
                                    ? tcs.findIndex(
                                          (tc) =>
                                              tc.id === event.tool_call_id &&
                                              tc.isRunning,
                                      )
                                    : tcs.length - 1;
                                if (ti >= 0) {
                                    const updated = [...tcs];
                                    updated[ti] = {
                                        ...updated[ti],
                                        result: event.tool_result,
                                        isError: event.is_error,
                                        isRunning: false,
                                    };
                                    steps[si] = {
                                        ...steps[si],
                                        toolCalls: updated,
                                    };
                                    break;
                                }
                            }
                            msg.steps = steps;
                            break;
                        }
                        case "done":
                            msg.isStreaming = false;
                            break;
                        case "error":
                            msg.isStreaming = false;
                            if (!msg.steps.some((s) => s.text)) {
                                msg.content =
                                    event.text || "An error occurred.";
                            }
                            break;
                    }

                    const updated = prev.slice();
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
