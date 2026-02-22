import { api } from "./client";

export function getFileUrl(filePath: string): string {
    return "/api/file?path=" + encodeURIComponent(filePath);
}

export async function openFolder(folderPath: string): Promise<void> {
    await api.post("/api/open-folder", { path: folderPath });
}

export async function openFileLocation(filePath: string): Promise<void> {
    await api.post("/api/open-file-location", { path: filePath });
}
