export async function shutdown(): Promise<void> {
    try {
        await fetch("/api/shutdown", { method: "POST" });
    } catch {
        // Server may close before responding
    }
}
