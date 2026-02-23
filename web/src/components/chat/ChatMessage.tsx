import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { ChatMessage as ChatMessageType } from "@/types/chat";
import { ToolCallBlock } from "./ToolCallBlock";

function ThinkingIndicator() {
    return (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <span className="inline-flex gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground/50 animate-bounce [animation-delay:0ms]" />
                <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground/50 animate-bounce [animation-delay:150ms]" />
                <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground/50 animate-bounce [animation-delay:300ms]" />
            </span>
            <span>Thinking...</span>
        </div>
    );
}

function MarkdownBlock({ content }: { content: string }) {
    return (
        <div className="prose prose-sm dark:prose-invert max-w-none prose-p:my-2 prose-headings:my-3 prose-ul:my-2 prose-ol:my-2 prose-li:my-0.5 prose-pre:bg-muted prose-pre:border prose-pre:border-border">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {content}
            </ReactMarkdown>
        </div>
    );
}

export function ChatMessage({ message }: { message: ChatMessageType }) {
    if (message.role === "user") {
        return (
            <div className="rounded-xl bg-muted/70 dark:bg-muted/50 px-4 py-3 border border-border/50">
                <div className="text-sm text-foreground whitespace-pre-wrap">
                    {message.content}
                </div>
            </div>
        );
    }

    // Assistant message
    const hasToolCalls =
        message.toolCalls && message.toolCalls.length > 0;
    const hasContent = message.content.length > 0;
    const showThinking =
        message.isStreaming && !hasContent && !hasToolCalls;

    return (
        <div className="flex flex-col gap-2 px-1">
            {showThinking && <ThinkingIndicator />}
            {hasToolCalls &&
                message.toolCalls!.map((tc, i) => (
                    <ToolCallBlock key={`${tc.name}-${i}`} toolCall={tc} />
                ))}
            {message.isStreaming &&
                hasToolCalls &&
                !hasContent &&
                !message.toolCalls!.some((tc) => tc.isRunning) && (
                    <ThinkingIndicator />
                )}
            {hasContent && <MarkdownBlock content={message.content} />}
        </div>
    );
}
