import {
    FileText,
    Image as ImageIcon,
    ExternalLink,
    Folder,
    HardDrive,
    Clock,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { SearchResult } from "@/types/search";
import { formatFileSize, calculateRelevanceScore } from "@/lib/format";
import { getFileUrl } from "@/api/files";
import * as filesApi from "@/api/files";
import { toast } from "sonner";

interface ResultCardProps {
    result: SearchResult;
    index: number;
    onViewFullText: (result: SearchResult) => void;
    onOpenPDF: (result: SearchResult) => void;
}

export function ResultCard({
    result,
    index,
    onViewFullText,
    onOpenPDF,
}: ResultCardProps) {
    const isPDF =
        result.fileName?.toLowerCase().endsWith(".pdf") ||
        result.filePath?.toLowerCase().endsWith(".pdf");
    const hasPageNumbers =
        result.pageNumbers &&
        Array.isArray(result.pageNumbers) &&
        result.pageNumbers.length > 0;
    const relevance = calculateRelevanceScore(result.similarity);

    const handleOpenFileLocation = async () => {
        try {
            await filesApi.openFileLocation(result.filePath);
        } catch (error) {
            toast.error("Failed to open file location: " + String(error));
        }
    };

    return (
        <div
            key={result.chunkId + index}
            className="border rounded-lg p-5 bg-card/50 transition-all duration-150 ease-in-out hover:bg-muted hover:border-border"
        >
            {/* Header */}
            <div className="flex items-start justify-between gap-4 mb-3">
                <div className="flex items-center gap-2 text-sm text-muted-foreground flex-1 min-w-0">
                    {result.contentType === "image" ? (
                        <ImageIcon className="w-4 h-4 flex-shrink-0" />
                    ) : (
                        <FileText className="w-4 h-4 flex-shrink-0" />
                    )}
                    <span className="font-mono text-xs truncate font-medium text-foreground/80">
                        {result.fileName || result.filePath}
                    </span>
                    {isPDF && hasPageNumbers && result.pageNumbers ? (
                        <button
                            onClick={() => onOpenPDF(result)}
                            className="text-xs flex-shrink-0 bg-blue-500/10 px-2 py-0.5 rounded border border-blue-500/20 text-blue-600 dark:text-blue-400 hover:bg-blue-500/20 transition-colors cursor-pointer font-medium"
                            title="Preview PDF at this page"
                        >
                            Page {result.pageNumbers[0]}
                        </button>
                    ) : !isPDF &&
                      result.chunkIndex !== undefined &&
                      result.totalChunks ? (
                        <span className="text-xs flex-shrink-0 text-muted-foreground">
                            Chunk {result.chunkIndex + 1}/{result.totalChunks}
                        </span>
                    ) : null}
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                    <Badge variant="secondary" className="text-xs font-medium">
                        {relevance.toFixed(0)}%
                    </Badge>
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8"
                                onClick={() => onViewFullText(result)}
                            >
                                <FileText className="w-4 h-4" />
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent>View full text</TooltipContent>
                    </Tooltip>
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8"
                                onClick={() =>
                                    window.open(getFileUrl(result.filePath), "_blank")
                                }
                            >
                                <ExternalLink className="w-4 h-4" />
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent>Open file</TooltipContent>
                    </Tooltip>
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8"
                                onClick={handleOpenFileLocation}
                            >
                                <Folder className="w-4 h-4" />
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent>Show in file explorer</TooltipContent>
                    </Tooltip>
                </div>
            </div>

            {/* Content */}
            {result.contentType === "image" && result.imageData ? (
                <div className="mb-3">
                    <img
                        src={`data:image/jpeg;base64,${result.imageData}`}
                        alt={result.fileName}
                        className="max-w-full h-auto max-h-64 rounded-lg border"
                    />
                    <p className="text-xs text-muted-foreground mt-2 italic">
                        {result.content}
                    </p>
                </div>
            ) : (
                <p className="text-sm leading-relaxed text-foreground/70 mb-3 line-clamp-2">
                    {result.content}
                </p>
            )}

            {/* Metadata Footer */}
            <div className="flex items-center gap-4 text-xs text-muted-foreground pt-3 border-t border-border/50">
                {result.fileSize && (
                    <div className="flex items-center gap-1.5">
                        <HardDrive className="w-3 h-3" />
                        <span>{formatFileSize(result.fileSize)}</span>
                    </div>
                )}
                {result.modifiedDate && (
                    <div className="flex items-center gap-1.5">
                        <Clock className="w-3 h-3" />
                        <span>{result.modifiedDate}</span>
                    </div>
                )}
                <div className="flex items-center gap-1.5">
                    <Folder className="w-3 h-3" />
                    <span className="font-mono truncate max-w-md">
                        {result.filePath}
                    </span>
                </div>
            </div>
        </div>
    );
}
