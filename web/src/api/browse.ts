import { api } from "./client";
import type { BrowseResponse } from "@/types/browse";

export async function listDirectory(
    path?: string,
    signal?: AbortSignal,
): Promise<BrowseResponse> {
    const url = path
        ? `/api/browse?path=${encodeURIComponent(path)}`
        : `/api/browse`;
    return api.get<BrowseResponse>(url, signal);
}
