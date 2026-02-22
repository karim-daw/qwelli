export interface SetupStatusResponse {
    needsSetup: boolean;
}

export interface SetupRequest {
    apiKey: string;
    model: string;
    endpoint: string;
    enableMultimodal: boolean;
    enableReranker: boolean;
}
