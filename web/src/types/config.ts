export interface ConfigResponse {
    apiKeyMasked: string;
    model: string;
    endpoint: string;
    foundryEndpoint: string;
    foundryKeyMasked: string;
    foundryModel: string;
}

export interface UpdateConfigRequest {
    apiKey?: string;
    model?: string;
    endpoint?: string;
    foundryEndpoint?: string;
    foundryApiKey?: string;
    foundryModel?: string;
}

export interface UpdateConfigResponse {
    status: string;
    warnings?: string[];
}
