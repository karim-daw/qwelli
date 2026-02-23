export interface ChatMessage {
    id: string;
    role: "user" | "assistant";
    content: string;
    toolCalls?: ToolCall[];
    isStreaming?: boolean;
}

export interface ToolCall {
    name: string;
    input: Record<string, unknown>;
    result?: string;
    isError?: boolean;
    isRunning?: boolean;
}

export interface ChatEvent {
    type:
        | "thinking"
        | "tool_call"
        | "tool_result"
        | "text_delta"
        | "done"
        | "error";
    text?: string;
    tool_name?: string;
    tool_input?: Record<string, unknown>;
    tool_result?: string;
    is_error?: boolean;
}
