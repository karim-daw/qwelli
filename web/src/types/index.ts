export interface Index {
    name: string;
    path: string;
    documentCount: number;
    lastModified: string;
}

export interface IndexStatus {
    toAdd: FileStatus[];
    toUpdate: FileStatus[];
    toDelete: FileStatus[];
    total: number;
    upToDate: number;
}

export interface FileStatus {
    path: string;
    fileType: string;
    size: number;
    modifiedAt: string;
    reason: string;
}
