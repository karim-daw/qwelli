import { useState } from "react";
import { Folder } from "lucide-react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { convertWindowsPathToWSL } from "@/lib/format";

interface NewIndexDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSubmit: (path: string, contentType: string) => void;
    indexing: boolean;
}

export function NewIndexDialog({
    open,
    onOpenChange,
    onSubmit,
    indexing,
}: NewIndexDialogProps) {
    const [newIndexPath, setNewIndexPath] = useState("");
    const [contentType, setContentType] = useState<"both" | "text" | "images">(
        "both",
    );

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (!newIndexPath.trim()) return;
        onSubmit(newIndexPath, contentType);
    };

    const handleClose = () => {
        if (!indexing) {
            onOpenChange(false);
            setNewIndexPath("");
            setContentType("both");
        }
    };

    return (
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent className="max-w-md">
                <DialogHeader>
                    <DialogTitle>Index a new folder</DialogTitle>
                </DialogHeader>
                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <Label className="mb-2 block">
                            Folder path (absolute path)
                        </Label>
                        <div className="flex gap-2">
                            <Input
                                type="text"
                                value={newIndexPath}
                                onChange={(e) => setNewIndexPath(e.target.value)}
                                placeholder="C:\Users\karim\Documents or /home/user/Documents"
                                disabled={indexing}
                            />
                            <label className="px-4 py-2 bg-secondary hover:bg-accent border rounded-md cursor-pointer transition-colors flex items-center gap-2 text-sm">
                                <Folder className="w-4 h-4" />
                                Browse
                                <input
                                    type="file"
                                    // @ts-expect-error webkitdirectory is not in types
                                    webkitdirectory=""
                                    directory=""
                                    multiple
                                    className="hidden"
                                    onChange={(e) => {
                                        const files = e.target.files;
                                        if (files && files.length > 0) {
                                            const firstFile = files[0];
                                            const relativePath = (
                                                firstFile as unknown as {
                                                    webkitRelativePath: string;
                                                }
                                            ).webkitRelativePath;
                                            if (relativePath) {
                                                const parts =
                                                    relativePath.split("/");
                                                setNewIndexPath(parts[0]);
                                            }
                                        }
                                    }}
                                    disabled={indexing}
                                />
                            </label>
                        </div>
                        <p className="text-xs text-muted-foreground mt-2">
                            Paste full path or use Browse (then complete the path
                            manually)
                        </p>
                        {newIndexPath &&
                            convertWindowsPathToWSL(newIndexPath) !==
                                newIndexPath && (
                                <p className="text-xs text-muted-foreground mt-1">
                                    &rarr; {convertWindowsPathToWSL(newIndexPath)}
                                </p>
                            )}
                    </div>

                    {/* Content Type Selection */}
                    <div>
                        <Label className="mb-2 block">Index Content Type</Label>
                        <div className="flex gap-2">
                            {(
                                [
                                    { value: "both", label: "Both" },
                                    { value: "text", label: "Text Only" },
                                    { value: "images", label: "Images Only" },
                                ] as const
                            ).map((opt) => (
                                <Button
                                    key={opt.value}
                                    type="button"
                                    variant={
                                        contentType === opt.value
                                            ? "default"
                                            : "outline"
                                    }
                                    className={cn("flex-1")}
                                    onClick={() => setContentType(opt.value)}
                                    disabled={indexing}
                                >
                                    {opt.label}
                                </Button>
                            ))}
                        </div>
                        <p className="text-xs text-muted-foreground mt-2">
                            {contentType === "both" &&
                                "Index both text and images from PDFs"}
                            {contentType === "text" &&
                                "Index only text content (faster, smaller index)"}
                            {contentType === "images" &&
                                "Index only images from PDFs"}
                        </p>
                    </div>

                    <div className="flex gap-2">
                        <Button
                            type="button"
                            variant="outline"
                            onClick={handleClose}
                            disabled={indexing}
                            className="flex-1"
                        >
                            Cancel
                        </Button>
                        <Button
                            type="submit"
                            disabled={indexing || !newIndexPath.trim()}
                            className="flex-1"
                        >
                            {indexing ? "Indexing..." : "Start Indexing"}
                        </Button>
                    </div>
                </form>
            </DialogContent>
        </Dialog>
    );
}
