import { useState } from "react";
import { useAppContext } from "@/contexts/AppContext";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import * as setupApi from "@/api/setup";
import * as serverApi from "@/api/server";

export function SetupScreen() {
    const { setQuitConfirmed } = useAppContext();
    const [apiKey, setApiKey] = useState("");
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState("");

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError("");
        setSaving(true);
        try {
            await setupApi.saveSetup({
                apiKey: apiKey.trim(),
                model: "voyage-multimodal-3.5",
                endpoint: "https://api.voyageai.com/v1/multimodalembeddings",
                enableMultimodal: true,
                enableReranker: true,
            });
            window.location.reload();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Setup failed");
        } finally {
            setSaving(false);
        }
    };

    const handleQuit = async () => {
        await serverApi.shutdown();
        setQuitConfirmed(true);
    };

    return (
        <div className="flex h-screen bg-background text-foreground">
            <div className="flex-1 flex flex-col items-center justify-center p-8">
                <Card className="w-full max-w-md shadow-2xl">
                    <CardHeader>
                        <CardTitle className="text-2xl">Welcome to Qwelli</CardTitle>
                        <CardDescription>
                            Enter your Voyage AI API key to get started. Get one at{" "}
                            <a
                                href="https://dash.voyageai.com/"
                                target="_blank"
                                rel="noreferrer"
                                className="underline"
                            >
                                dash.voyageai.com
                            </a>
                        </CardDescription>
                    </CardHeader>
                    <CardContent>
                        <form onSubmit={handleSubmit} className="space-y-4">
                            <div>
                                <Label htmlFor="api-key">API Key</Label>
                                <Input
                                    id="api-key"
                                    type="password"
                                    value={apiKey}
                                    onChange={(e) => setApiKey(e.target.value)}
                                    placeholder="sk-..."
                                    required
                                    autoComplete="off"
                                    className="mt-2"
                                />
                            </div>
                            {error && (
                                <p className="text-sm text-destructive">{error}</p>
                            )}
                            <Button
                                type="submit"
                                disabled={saving || !apiKey.trim()}
                                className="w-full"
                            >
                                {saving ? "Saving..." : "Continue"}
                            </Button>
                            <Button
                                type="button"
                                variant="outline"
                                onClick={handleQuit}
                                className="w-full"
                            >
                                Quit
                            </Button>
                        </form>
                    </CardContent>
                </Card>
            </div>
        </div>
    );
}
