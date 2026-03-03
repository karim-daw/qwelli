export interface MessageStep {
    thinking?: string;
    text?: string;
    toolCalls?: ToolCall[];
}

export interface ChatMessage {
    id: string;
    role: "user" | "assistant";
    content: string;
    steps: MessageStep[]; // one entry per agent loop iteration, in order
    isStreaming?: boolean;
}

export interface ToolCall {
    id?: string; // Anthropic tool_use block ID — used to match tool_result to the right tool_call
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
    tool_call_id?: string; // links tool_result back to its tool_call
    tool_input?: Record<string, unknown>;
    tool_result?: string;
    is_error?: boolean;
}
