import { useState, useCallback } from "react";
import type { SearchResult } from "@/types/search";

export interface PDFPreviewData {
    filePath: string;
    initialPage: number;
    fileName: string;
}

export function usePDFPreview() {
    const [isOpen, setIsOpen] = useState(false);
    const [data, setData] = useState<PDFPreviewData | null>(null);

    const open = useCallback((result: SearchResult) => {
        const isPDF =
            result.fileName?.toLowerCase().endsWith(".pdf") ||
            result.filePath?.toLowerCase().endsWith(".pdf");
        if (!isPDF) return;

        setData({
            filePath: result.filePath,
            initialPage: result.pageNumbers?.[0] || 1,
            fileName: result.fileName || result.filePath,
        });
        setIsOpen(true);
    }, []);

    const close = useCallback(() => {
        setIsOpen(false);
        setData(null);
    }, []);

    return { isOpen, data, open, close };
}
