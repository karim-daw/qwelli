import {
    FileText,
    Image as ImageIcon,
    ExternalLink,
    Folder,
    HardDrive,
    Clock,
} from "lucide-react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { SearchResult } from "@/types/search";
import { formatFileSize, calculateRelevanceScore } from "@/lib/format";
import { getFileUrl } from "@/api/files";
import * as filesApi from "@/api/files";
import { toast } from "sonner";

interface FullTextModalProps {
    result: SearchResult;
    open: boolean;
    onClose: () => void;
}

export function FullTextModal({ result, open, onClose }: FullTextModalProps) {
    const relevance = calculateRelevanceScore(result.similarity);

    const handleOpenFileLocation = async () => {
        try {
            await filesApi.openFileLocation(result.filePath);
        } catch (error) {
            toast.error("Failed to open file location: " + String(error));
        }
    };

    return (
        <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
            <DialogContent className="max-w-4xl max-h-[90vh] flex flex-col p-0">
                {/* Header */}
                <DialogHeader className="p-6 border-b">
                    <DialogTitle className="flex items-center gap-2">
                        {result.contentType === "image" ? (
                            <ImageIcon className="w-5 h-5 flex-shrink-0" />
                        ) : (
                            <FileText className="w-5 h-5 flex-shrink-0" />
                        )}
                        <span className="truncate">
                            {result.fileName || result.filePath}
                        </span>
                    </DialogTitle>
                    <div className="flex items-center gap-4 text-xs text-muted-foreground mt-2">
                        {result.pageNumbers &&
                            result.pageNumbers.length > 0 && (
                                <Badge
                                    variant="outline"
                                    className="text-blue-600 dark:text-blue-400 border-blue-500/20 bg-blue-500/10"
                                >
                                    Page {result.pageNumbers[0]}
                                </Badge>
                            )}
                        {result.chunkIndex !== undefined &&
                            result.totalChunks && (
                                <span>
                                    Chunk {result.chunkIndex + 1} /{" "}
                                    {result.totalChunks}
                                </span>
                            )}
                        <Badge variant="secondary">
                            {relevance.toFixed(0)}% match
                        </Badge>
                    </div>
                </DialogHeader>

                {/* Body */}
                <ScrollArea className="flex-1 p-6">
                    {result.contentType === "image" && result.imageData ? (
                        <div className="flex flex-col items-center">
                            <img
                                src={`data:image/jpeg;base64,${result.imageData}`}
                                alt={result.fileName}
                                className="max-w-full h-auto rounded-lg border shadow-lg"
                            />
                            <p className="text-sm text-muted-foreground mt-4 italic text-center">
                                {result.content}
                            </p>
                        </div>
                    ) : (
                        <div className="text-sm leading-relaxed text-foreground/80 whitespace-pre-wrap font-mono bg-muted p-4 rounded-lg border">
                            {result.content}
                        </div>
                    )}
                </ScrollArea>

                {/* Footer */}
                <div className="border-t p-6 bg-card/50">
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4 text-xs">
                        {result.fileSize && (
                            <div>
                                <div className="text-muted-foreground mb-1">
                                    File Size
                                </div>
                                <div className="flex items-center gap-1.5">
                                    <HardDrive className="w-3 h-3" />
                                    {formatFileSize(result.fileSize)}
                                </div>
                            </div>
                        )}
                        {result.modifiedDate && (
                            <div>
                                <div className="text-muted-foreground mb-1">
                                    Modified
                                </div>
                                <div className="flex items-center gap-1.5">
                                    <Clock className="w-3 h-3" />
                                    {result.modifiedDate}
                                </div>
                            </div>
                        )}
                        <div className="col-span-2">
                            <div className="text-muted-foreground mb-1">
                                File Path
                            </div>
                            <div className="flex items-center gap-1.5 font-mono text-xs">
                                <Folder className="w-3 h-3 flex-shrink-0" />
                                <span className="truncate">
                                    {result.filePath}
                                </span>
                            </div>
                        </div>
                    </div>
                    <div className="flex gap-2">
                        <Button
                            onClick={() =>
                                window.open(
                                    getFileUrl(result.filePath),
                                    "_blank",
                                )
                            }
                        >
                            <ExternalLink className="w-4 h-4 mr-2" />
                            Open File
                        </Button>
                        <Button
                            variant="outline"
                            onClick={handleOpenFileLocation}
                        >
                            <Folder className="w-4 h-4 mr-2" />
                            Show in Explorer
                        </Button>
                        <Button
                            variant="outline"
                            onClick={onClose}
                            className="flex-1"
                        >
                            Close
                        </Button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}
