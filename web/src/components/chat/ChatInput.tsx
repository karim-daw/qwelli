import { useState, useRef, useEffect } from "react";
import { ArrowUp, Square } from "lucide-react";
import { Button } from "@/components/ui/button";

interface ChatInputProps {
    onSend: (text: string) => void;
    disabled: boolean;
    isStreaming: boolean;
    onCancel: () => void;
}

export function ChatInput({
    onSend,
    disabled,
    isStreaming,
    onCancel,
}: ChatInputProps) {
    const [value, setValue] = useState("");
    const textareaRef = useRef<HTMLTextAreaElement>(null);

    useEffect(() => {
        if (!isStreaming) {
            textareaRef.current?.focus();
        }
    }, [isStreaming]);

    const handleSubmit = () => {
        if (!value.trim() || disabled) return;
        onSend(value);
        setValue("");
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
        if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            handleSubmit();
        }
    };

    // Auto-resize textarea
    useEffect(() => {
        const el = textareaRef.current;
        if (!el) return;
        el.style.height = "auto";
        el.style.height = Math.min(el.scrollHeight, 200) + "px";
    }, [value]);

    return (
        <div className="flex flex-col gap-2">
            <div className="relative flex items-end border border-border rounded-xl bg-background shadow-sm focus-within:ring-1 focus-within:ring-ring">
                <textarea
                    ref={textareaRef}
                    value={value}
                    onChange={(e) => setValue(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder="Ask about your documents..."
                    disabled={isStreaming}
                    rows={1}
                    className="flex-1 resize-none bg-transparent px-4 py-3 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none disabled:opacity-50"
                />
                <div className="flex-shrink-0 p-2">
                    {isStreaming ? (
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 rounded-lg bg-destructive/10 hover:bg-destructive/20 text-destructive"
                            onClick={onCancel}
                        >
                            <Square className="w-3.5 h-3.5" />
                        </Button>
                    ) : (
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 rounded-lg bg-primary/10 hover:bg-primary/20 text-primary disabled:opacity-30"
                            onClick={handleSubmit}
                            disabled={!value.trim() || disabled}
                        >
                            <ArrowUp className="w-4 h-4" />
                        </Button>
                    )}
                </div>
            </div>
        </div>
    );
}
