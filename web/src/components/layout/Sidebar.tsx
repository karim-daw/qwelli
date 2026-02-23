import { useState, useMemo } from "react";
import {
    Folder,
    Plus,
    X,
    Activity,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useAppContext } from "@/contexts/AppContext";
import { useSearchContext } from "@/contexts/SearchContext";
import { useIndexes } from "@/hooks/useIndexes";
import * as filesApi from "@/api/files";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import type { Index } from "@/types/index";

interface SidebarProps {
    width: number;
    onResizeStart: () => void;
    onShowNewIndexDialog: () => void;
}

export function Sidebar({
    width,
    onResizeStart,
    onShowNewIndexDialog,
}: SidebarProps) {
    const { setSelectedIndex, setViewMode } = useAppContext();
    const { reset: resetSearch } = useSearchContext();
    const { indexes, selectedIndex, deleteIndex } = useIndexes();
    const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

    type IndexGroup = { label: string; indexes: Index[] };

    const groupedIndexes = useMemo((): IndexGroup[] => {
        const now = new Date();
        const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        const yesterdayStart = new Date(todayStart);
        yesterdayStart.setDate(todayStart.getDate() - 1);
        const last7Start = new Date(todayStart);
        last7Start.setDate(todayStart.getDate() - 7);

        const sorted = [...indexes].sort(
            (a, b) => new Date(b.lastModified).getTime() - new Date(a.lastModified).getTime()
        );

        const groups: IndexGroup[] = [
            { label: "Today", indexes: [] },
            { label: "Yesterday", indexes: [] },
            { label: "Last 7 Days", indexes: [] },
            { label: "Older", indexes: [] },
        ];

        for (const idx of sorted) {
            const d = new Date(idx.lastModified);
            const dayStart = new Date(d.getFullYear(), d.getMonth(), d.getDate());
            if (dayStart >= todayStart)             groups[0].indexes.push(idx);
            else if (dayStart >= yesterdayStart)    groups[1].indexes.push(idx);
            else if (dayStart >= last7Start)        groups[2].indexes.push(idx);
            else                                    groups[3].indexes.push(idx);
        }

        return groups.filter((g) => g.indexes.length > 0);
    }, [indexes]);

    const handleSelectIndex = (index: Index) => {
        setSelectedIndex(index.path);
        setViewMode("search");
        resetSearch();
    };

    const handleOpenFolder = async (path: string) => {
        try {
            await filesApi.openFolder(path);
        } catch (error) {
            toast.error("Failed to open folder: " + String(error));
        }
    };

    const handleViewStatus = (index: Index) => {
        setSelectedIndex(index.path);
        setViewMode("status");
    };

    const handleConfirmDelete = async () => {
        if (deleteConfirm) {
            await deleteIndex(deleteConfirm);
            resetSearch();
            setDeleteConfirm(null);
        }
    };

    return (
        <div style={{ width }} className="flex-shrink-0 border-r relative">
            <div className="h-full flex flex-col">
                {/* Header */}
                <div className="p-4 border-b">
                    <div className="flex items-center justify-between mb-3">
                        <h2 className="text-sm font-medium">Indexes</h2>
                        <div className="flex items-center gap-2">
                            <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7"
                                onClick={onShowNewIndexDialog}
                            >
                                <Plus className="w-4 h-4" />
                            </Button>
                        </div>
                    </div>
                </div>

                {/* Index List */}
                <div className="flex-1 overflow-y-auto custom-scrollbar">
                    {groupedIndexes.map((group) => (
                        <div key={group.label}>
                            <div className="px-3 py-1 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                                {group.label}
                            </div>
                            {group.indexes.map((index) => (
                                <button
                                    key={index.path}
                                    onClick={() => handleSelectIndex(index)}
                                    className={cn(
                                        "w-full text-left px-4 py-2.5 border-b hover:bg-accent transition-colors group",
                                        selectedIndex === index.path && "bg-accent",
                                    )}
                                >
                                    <div className="flex items-start justify-between gap-2">
                                        <div className="flex-1 min-w-0">
                                            <div className="flex items-center gap-2 mb-1">
                                                <button
                                                    onClick={(e) => {
                                                        e.stopPropagation();
                                                        handleOpenFolder(index.path);
                                                    }}
                                                    className="p-0.5 hover:bg-accent rounded transition-colors"
                                                    title="Open folder in file explorer"
                                                >
                                                    <Folder className="w-3.5 h-3.5 flex-shrink-0 text-muted-foreground hover:text-foreground" />
                                                </button>
                                                <span className="text-sm truncate">
                                                    {index.name}
                                                </span>
                                            </div>
                                            <div className="text-xs text-muted-foreground">
                                                {index.documentCount} docs
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-1">
                                            <button
                                                onClick={(e) => {
                                                    e.stopPropagation();
                                                    handleViewStatus(index);
                                                }}
                                                className="opacity-0 group-hover:opacity-100 p-1 hover:bg-accent rounded transition-opacity"
                                                title="View status & changes"
                                            >
                                                <Activity className="w-3.5 h-3.5" />
                                            </button>
                                            <button
                                                onClick={(e) => {
                                                    e.stopPropagation();
                                                    setDeleteConfirm(index.path);
                                                }}
                                                className="opacity-0 group-hover:opacity-100 p-1 hover:bg-red-500/20 rounded transition-opacity text-red-600 dark:text-red-400"
                                                title="Delete index"
                                            >
                                                <X className="w-3.5 h-3.5" />
                                            </button>
                                        </div>
                                    </div>
                                </button>
                            ))}
                        </div>
                    ))}
                </div>
            </div>

            {/* Resize Handle */}
            <div
                className="absolute top-0 right-0 w-1 h-full cursor-col-resize hover:bg-accent transition-colors z-10"
                onMouseDown={onResizeStart}
            />

            {/* Delete Confirmation */}
            <AlertDialog
                open={!!deleteConfirm}
                onOpenChange={() => setDeleteConfirm(null)}
            >
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete Index</AlertDialogTitle>
                        <AlertDialogDescription>
                            Are you sure you want to delete this index? This
                            cannot be undone.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction onClick={handleConfirmDelete}>
                            Delete
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}
