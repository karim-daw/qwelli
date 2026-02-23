import { useRef, useEffect, useState, useCallback } from "react";
import { MessageSquare, Trash2, ArrowDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useAppContext } from "@/contexts/AppContext";
import { useChat } from "@/hooks/useChat";
import { ChatMessage } from "./ChatMessage";
import { ChatInput } from "./ChatInput";

export function ChatView() {
    const { selectedIndex } = useAppContext();
    const { messages, isStreaming, sendMessage, cancelStream, clearHistory } =
        useChat(selectedIndex);
    const scrollContainerRef = useRef<HTMLDivElement>(null);
    const bottomRef = useRef<HTMLDivElement>(null);
    const [isNearBottom, setIsNearBottom] = useState(true);
    const [showScrollBtn, setShowScrollBtn] = useState(false);
    const rafRef = useRef<number | null>(null);

    // Track scroll position, throttled to one update per animation frame
    const handleScroll = useCallback(() => {
        if (rafRef.current !== null) return;
        rafRef.current = requestAnimationFrame(() => {
            rafRef.current = null;
            const el = scrollContainerRef.current;
            if (!el) return;
            const threshold = 80;
            const nearBottom =
                el.scrollHeight - el.scrollTop - el.clientHeight < threshold;
            setIsNearBottom(nearBottom);
            setShowScrollBtn(!nearBottom);
        });
    }, []);

    // Clean up pending rAF on unmount
    useEffect(() => {
        return () => {
            if (rafRef.current !== null) cancelAnimationFrame(rafRef.current);
        };
    }, []);

    // Auto-scroll only when user is near the bottom
    useEffect(() => {
        if (isNearBottom) {
            bottomRef.current?.scrollIntoView({ behavior: "instant" });
        }
    }, [messages, isNearBottom]);

    const scrollToBottom = useCallback(() => {
        bottomRef.current?.scrollIntoView({ behavior: "smooth" });
        setShowScrollBtn(false);
    }, []);

    return (
        <div className="flex-1 flex flex-col overflow-hidden relative">
            {/* Scrollable messages area */}
            <div
                ref={scrollContainerRef}
                onScroll={handleScroll}
                className="flex-1 overflow-y-auto"
            >
                <div className="max-w-3xl mx-auto px-6 py-6 flex flex-col gap-6">
                    {messages.length === 0 && (
                        <div className="text-center text-muted-foreground py-24">
                            <MessageSquare className="w-12 h-12 mx-auto mb-3 opacity-30" />
                            <p className="text-sm">
                                Ask a question about your documents
                            </p>
                        </div>
                    )}

                    {messages.map((m) => (
                        <ChatMessage key={m.id} message={m} />
                    ))}

                    {messages.length > 0 && !isStreaming && (
                        <div className="flex justify-center">
                            <Button
                                variant="ghost"
                                size="sm"
                                className="text-xs text-muted-foreground hover:text-foreground gap-1.5"
                                onClick={clearHistory}
                            >
                                <Trash2 className="w-3 h-3" />
                                Clear conversation
                            </Button>
                        </div>
                    )}

                    <div ref={bottomRef} />
                </div>
            </div>

            {/* Scroll-to-bottom button */}
            {showScrollBtn && (
                <div className="absolute left-1/2 -translate-x-1/2 bottom-24 z-10">
                    <Button
                        variant="outline"
                        size="icon"
                        className="h-8 w-8 rounded-full shadow-md bg-background/90 backdrop-blur-sm border-border/80"
                        onClick={scrollToBottom}
                    >
                        <ArrowDown className="w-4 h-4" />
                    </Button>
                </div>
            )}

            {/* Pinned input area */}
            <div className="relative flex-shrink-0">
                <div className="absolute -top-8 left-0 right-0 h-8 bg-gradient-to-t from-background to-transparent pointer-events-none" />
                <div className="max-w-3xl mx-auto px-6 pb-4 pt-2">
                    <ChatInput
                        onSend={sendMessage}
                        disabled={isStreaming || !selectedIndex}
                        isStreaming={isStreaming}
                        onCancel={cancelStream}
                    />
                </div>
            </div>
        </div>
    );
}
