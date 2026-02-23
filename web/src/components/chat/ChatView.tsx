import { useRef, useEffect } from "react";
import { MessageSquare, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useAppContext } from "@/contexts/AppContext";
import { useChat } from "@/hooks/useChat";
import { ChatMessage } from "./ChatMessage";
import { ChatInput } from "./ChatInput";

export function ChatView() {
    const { selectedIndex } = useAppContext();
    const { messages, isStreaming, sendMessage, cancelStream, clearHistory } =
        useChat(selectedIndex);
    const scrollRef = useRef<HTMLDivElement>(null);

    // Auto-scroll on new content
    useEffect(() => {
        scrollRef.current?.scrollIntoView({ behavior: "smooth" });
    }, [messages]);

    return (
        <div className="flex-1 overflow-y-auto">
            <div className="max-w-3xl mx-auto px-6 py-6 flex flex-col gap-6">
                {messages.length === 0 && (
                    <div className="text-center text-muted-foreground py-16">
                        <MessageSquare className="w-10 h-10 mx-auto mb-3 opacity-40" />
                        <p className="text-sm">
                            Ask a question about your documents
                        </p>
                    </div>
                )}

                {messages.map((m) => (
                    <ChatMessage key={m.id} message={m} />
                ))}

                <ChatInput
                    onSend={sendMessage}
                    disabled={isStreaming || !selectedIndex}
                    isStreaming={isStreaming}
                    onCancel={cancelStream}
                />

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

                <div ref={scrollRef} />
            </div>
        </div>
    );
}
