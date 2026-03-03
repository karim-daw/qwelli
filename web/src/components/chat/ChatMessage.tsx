import { Fragment, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Brain, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";
import type { ChatMessage as ChatMessageType } from "@/types/chat";
import { ToolCallBlock } from "./ToolCallBlock";

function ThinkingBlock({
    text,
    label,
    isStreaming,
}: {
    text: string;
    label: string;
    isStreaming?: boolean;
}) {
    const [open, setOpen] = useState(true);
    return (
        <div className="text-xs border border-border/50 rounded-lg overflow-hidden">
            <button
                onClick={() => setOpen((o) => !o)}
                className="w-full flex items-center gap-2 px-3 py-2 text-muted-foreground hover:text-foreground hover:bg-muted/40 transition-colors"
            >
                <Brain className="w-3.5 h-3.5 shrink-0" />
                <span className="flex-1 text-left font-medium">{label}</span>
                {isStreaming && (
                    <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground/50 animate-pulse" />
                )}
                <ChevronDown
                    className={cn(
                        "w-3.5 h-3.5 transition-transform",
                        open && "rotate-180",
                    )}
                />
            </button>
            {open && (
                <div className="px-3 py-2 border-t border-border/50 bg-muted/20 font-mono text-[11px] text-muted-foreground/80 whitespace-pre-wrap max-h-56 overflow-y-auto leading-relaxed">
                    {text}
                </div>
            )}
        </div>
    );
}

function ThinkingIndicator() {
    return (
        <div className="flex items-center gap-2.5 text-sm text-muted-foreground">
            <span className="inline-flex flex-col gap-1">
                <span className="w-4 h-1 rounded-full bg-muted-foreground/40 animate-pulse [animation-delay:0ms]" />
                <span className="w-4 h-1 rounded-full bg-muted-foreground/40 animate-pulse [animation-delay:150ms]" />
                <span className="w-4 h-1 rounded-full bg-muted-foreground/40 animate-pulse [animation-delay:300ms]" />
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

    const lastStep = message.steps[message.steps.length - 1];

    // Dots before anything has started streaming in
    const showIndicator =
        message.isStreaming &&
        !message.steps[0]?.thinking &&
        !message.steps[0]?.text &&
        !message.steps[0]?.toolCalls?.length;

    // Dots after tools finish while waiting for the next iteration to start
    const showReThinking =
        message.isStreaming &&
        !!lastStep?.toolCalls?.length &&
        !lastStep.toolCalls.some((tc) => tc.isRunning);

    return (
        <div className="flex flex-col gap-3 px-1">
            {showIndicator && <ThinkingIndicator />}

            {message.steps.map((step, i) => {
                const isLastStep = i === message.steps.length - 1;
                // Thinking is still streaming if it's the last step and no text
                // or tools have appeared in this step yet.
                const thinkingStreaming =
                    message.isStreaming &&
                    isLastStep &&
                    !step.text &&
                    !step.toolCalls?.length;
                const isFirst = i === 0;
                const leadToMoreTools =
                    !isLastStep || !!step.toolCalls?.length;
                const label = isFirst
                    ? "Reasoning"
                    : leadToMoreTools
                      ? "Reflecting"
                      : "Synthesizing";
                return (
                    <Fragment key={i}>
                        {step.thinking && (
                            <ThinkingBlock
                                text={step.thinking}
                                label={label}
                                isStreaming={thinkingStreaming}
                            />
                        )}
                        {step.text && (
                            <MarkdownBlock content={step.text} />
                        )}
                        {step.toolCalls?.map((tc, j) => (
                            <ToolCallBlock
                                key={`${tc.name}-${j}`}
                                toolCall={tc}
                            />
                        ))}
                    </Fragment>
                );
            })}

            {showReThinking && <ThinkingIndicator />}

            {/* Error fallback — only when no step produced text */}
            {!message.steps.some((s) => s.text) && message.content && (
                <MarkdownBlock content={message.content} />
            )}
        </div>
    );
}
