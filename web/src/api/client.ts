export class ApiError extends Error {
    constructor(
        message: string,
        public status: number,
        public data?: unknown,
    ) {
        super(message);
        this.name = "ApiError";
    }
}

async function handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
        let data: unknown;
        try {
            data = await response.json();
        } catch {
            // ignore parse failure
        }
        const message =
            (data as { error?: string })?.error ||
            `Request failed with status ${response.status}`;
        throw new ApiError(message, response.status, data);
    }
    return response.json() as Promise<T>;
}

export const api = {
    async get<T>(url: string): Promise<T> {
        const response = await fetch(url);
        return handleResponse<T>(response);
    },

    async post<T>(url: string, body?: unknown): Promise<T> {
        const response = await fetch(url, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: body ? JSON.stringify(body) : undefined,
        });
        return handleResponse<T>(response);
    },

    async rawPost(url: string, body?: unknown): Promise<Response> {
        return fetch(url, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: body ? JSON.stringify(body) : undefined,
        });
    },

};
