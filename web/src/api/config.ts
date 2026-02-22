import { api } from "./client";
import type { ConfigResponse, UpdateConfigRequest, UpdateConfigResponse } from "@/types/config";

export async function getConfig(): Promise<ConfigResponse> {
    return api.get<ConfigResponse>("/api/config");
}

export async function updateConfig(req: UpdateConfigRequest): Promise<UpdateConfigResponse> {
    return api.post<UpdateConfigResponse>("/api/config/update", req);
}
