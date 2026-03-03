import type { FileStatus } from "@/types/index";
import type { LucideIcon } from "lucide-react";

interface FileChangeListProps {
    files: FileStatus[];
    icon: LucideIcon;
    label: string;
    colorClass: string;
}

export function FileChangeList({
    files,
    icon: Icon,
    label,
    colorClass,
}: FileChangeListProps) {
    if (!files || files.length === 0) return null;

    return (
        <div>
            <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
                <Icon className={`w-4 h-4 ${colorClass}`} />
                {label} ({files.length})
            </h3>
            <div className="space-y-2">
                {files.slice(0, 10).map((file) => (
                    <div
                        key={file.path}
                        className="text-xs bg-muted rounded px-3 py-2 border"
                    >
                        <div className="font-mono truncate">{file.path}</div>
                    </div>
                ))}
            </div>
        </div>
    );
}
