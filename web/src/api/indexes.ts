import { api } from "./client";
import type { Index, IndexStatus } from "@/types/index";

interface IndexListResponse {
    indexes: Index[];
}

export async function listIndexes(): Promise<Index[]> {
    const data = await api.get<IndexListResponse>("/api/indexes");
    return data.indexes || [];
}

export async function createIndex(
    folderPath: string,
    contentType: string,
): Promise<Response> {
    return api.rawPost("/api/index/create", { folderPath, contentType });
}

export async function deleteIndex(indexPath: string): Promise<void> {
    await api.post("/api/index/delete", { indexPath });
}

export async function updateIndex(indexPath: string): Promise<Response> {
    return api.rawPost("/api/index/update", { indexPath });
}

export async function cancelIndex(indexPath: string): Promise<void> {
    await api.post("/api/index/cancel", { indexPath });
}

export async function getIndexStatus(indexPath: string): Promise<IndexStatus> {
    return api.get<IndexStatus>(
        "/api/index/status?path=" + encodeURIComponent(indexPath),
    );
}
