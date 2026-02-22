export interface IndexProgress {
    current: number;
    total: number;
    file: string;
    indexPath: string;
}

export interface SSEProgressMessage {
    type: "progress";
    current: number;
    total: number;
    file: string;
}

export interface SSEPhaseMessage {
    type: "phase";
    phase?: string;
    message?: string;
    current?: number;
    total?: number;
}

export interface SSECompleteMessage {
    type: "complete";
}

export interface SSECancelledMessage {
    type: "cancelled";
}

export interface SSEErrorMessage {
    type: "error";
    message: string;
}

export type SSEMessage =
    | SSEProgressMessage
    | SSEPhaseMessage
    | SSECompleteMessage
    | SSECancelledMessage
    | SSEErrorMessage;
