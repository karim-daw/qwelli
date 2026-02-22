import { useState } from "react";
import {
    FileText,
    ExternalLink,
    Download,
    ZoomIn,
    ZoomOut,
    ChevronLeft,
    ChevronRight,
    X,
} from "lucide-react";
import { Document, Page, pdfjs } from "react-pdf";
import "react-pdf/dist/Page/AnnotationLayer.css";
import "react-pdf/dist/Page/TextLayer.css";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { getFileUrl } from "@/api/files";

pdfjs.GlobalWorkerOptions.workerSrc = `//unpkg.com/pdfjs-dist@${pdfjs.version}/build/pdf.worker.min.mjs`;

interface PDFPreviewModalProps {
    filePath: string;
    fileName: string;
    initialPage: number;
    onClose: () => void;
}

export function PDFPreviewModal({
    filePath,
    fileName,
    initialPage,
    onClose,
}: PDFPreviewModalProps) {
    const [numPages, setNumPages] = useState(0);
    const [pageNumber, setPageNumber] = useState(initialPage);
    const [scale, setScale] = useState(1.0);

    const fileUrl = getFileUrl(filePath);

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            {/* Backdrop */}
            <div
                className="absolute inset-0 bg-black/80"
                onClick={onClose}
            />

            {/* Modal */}
            <div className="relative z-10 bg-background border rounded-lg w-full max-w-6xl max-h-[95vh] flex flex-col shadow-2xl">
                {/* Header */}
                <div className="flex items-center justify-between p-4 border-b flex-shrink-0">
                    <div className="flex-1 min-w-0 pr-4">
                        <h2 className="text-lg font-medium flex items-center gap-2 truncate">
                            <FileText className="w-5 h-5 flex-shrink-0" />
                            <span className="truncate">{fileName}</span>
                        </h2>
                        {numPages > 0 && (
                            <div className="text-xs text-muted-foreground mt-1">
                                {numPages} pages
                            </div>
                        )}
                    </div>
                    <div className="flex items-center gap-2">
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setScale((s) => Math.max(0.5, s - 0.25))}
                            title="Zoom out"
                        >
                            <ZoomOut className="w-4 h-4" />
                        </Button>
                        <span className="text-xs text-muted-foreground min-w-[3rem] text-center">
                            {Math.round(scale * 100)}%
                        </span>
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setScale((s) => Math.min(2.0, s + 0.25))}
                            title="Zoom in"
                        >
                            <ZoomIn className="w-4 h-4" />
                        </Button>
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => window.open(fileUrl, "_blank")}
                            title="Open in new tab"
                        >
                            <ExternalLink className="w-4 h-4" />
                        </Button>
                        <a
                            href={fileUrl}
                            download
                            className="inline-flex items-center justify-center h-9 w-9 rounded-md hover:bg-accent transition-colors"
                            title="Download PDF"
                        >
                            <Download className="w-4 h-4" />
                        </a>
                        <Button variant="ghost" size="icon" onClick={onClose}>
                            <X className="w-5 h-5" />
                        </Button>
                    </div>
                </div>

                {/* PDF Viewer */}
                <div className="flex-1 overflow-auto bg-gray-200 dark:bg-gray-900 flex items-center justify-center p-4 min-h-0">
                    <Document
                        file={fileUrl}
                        onLoadSuccess={({ numPages: n }) => setNumPages(n)}
                        loading={
                            <div className="text-muted-foreground py-20">
                                Loading PDF...
                            </div>
                        }
                        error={
                            <div className="text-destructive py-20">
                                Failed to load PDF
                            </div>
                        }
                    >
                        <Page
                            pageNumber={pageNumber}
                            scale={scale}
                            renderTextLayer={true}
                            renderAnnotationLayer={true}
                            className="shadow-lg"
                        />
                    </Document>
                </div>

                {/* Page Navigation Footer */}
                <div className="flex items-center justify-between p-4 border-t bg-background flex-shrink-0">
                    <div className="flex items-center gap-2">
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setPageNumber((p) => Math.max(1, p - 1))}
                            disabled={pageNumber <= 1}
                        >
                            <ChevronLeft className="w-4 h-4" />
                        </Button>
                        <div className="flex items-center gap-2 text-sm">
                            <span className="text-muted-foreground">Page</span>
                            <Input
                                type="number"
                                min="1"
                                max={numPages}
                                value={pageNumber}
                                onChange={(e) => {
                                    const page = parseInt(e.target.value);
                                    if (page >= 1 && page <= numPages) {
                                        setPageNumber(page);
                                    }
                                }}
                                className="w-16 text-center"
                            />
                            <span className="text-muted-foreground">
                                of {numPages}
                            </span>
                        </div>
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={() =>
                                setPageNumber((p) => Math.min(numPages, p + 1))
                            }
                            disabled={pageNumber >= numPages}
                        >
                            <ChevronRight className="w-4 h-4" />
                        </Button>
                    </div>
                    <Button variant="outline" onClick={onClose}>
                        Close
                    </Button>
                </div>
            </div>
        </div>
    );
}
