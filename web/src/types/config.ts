export interface ConfigResponse {
    apiKeyMasked: string;
    model: string;
    endpoint: string;
}

export interface UpdateConfigRequest {
    apiKey?: string;
    model?: string;
    endpoint?: string;
}

export interface UpdateConfigResponse {
    status: string;
    warnings?: string[];
}
