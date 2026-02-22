import { api } from "./client";
import type { SetupStatusResponse, SetupRequest } from "@/types/setup";

export async function getSetupStatus(): Promise<SetupStatusResponse> {
    return api.get<SetupStatusResponse>("/api/setup/status");
}

export async function saveSetup(config: SetupRequest): Promise<void> {
    await api.post("/api/setup", config);
}
