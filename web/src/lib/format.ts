export function formatTimestamp(timestamp: number): string {
    const now = Date.now();
    const diff = now - timestamp;

    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return "Just now";
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days < 7) return `${days}d ago`;

    return new Date(timestamp).toLocaleDateString();
}

export function formatFileSize(bytes?: number): string {
    if (!bytes) return "Unknown";
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
    if (bytes < 1024 * 1024 * 1024)
        return (bytes / (1024 * 1024)).toFixed(1) + " MB";
    return (bytes / (1024 * 1024 * 1024)).toFixed(1) + " GB";
}

export function convertWindowsPathToWSL(path: string): string {
    const windowsPathRegex = /^([A-Za-z]):[/\\]/;
    const match = path.match(windowsPathRegex);

    if (match) {
        const drive = match[1].toLowerCase();
        let converted = path.replace(windowsPathRegex, "/mnt/" + drive + "/");
        converted = converted.replace(/\\/g, "/");
        return converted;
    }

    return path;
}

export function calculateRelevanceScore(similarity: number): number {
    if (similarity < 0) {
        return Math.abs(similarity) * 100;
    }
    return (1 - similarity) * 100;
}
