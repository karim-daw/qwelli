import { useState, useEffect } from "react";
import { Eye, EyeOff, Loader2 } from "lucide-react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import * as configApi from "@/api/config";

interface SettingsDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

export function SettingsDialog({ open, onOpenChange }: SettingsDialogProps) {
    const [apiKey, setApiKey] = useState("");
    const [apiKeyTouched, setApiKeyTouched] = useState(false);
    const [showApiKey, setShowApiKey] = useState(false);
    const [model, setModel] = useState("");
    const [endpoint, setEndpoint] = useState("");
    const [originalModel, setOriginalModel] = useState("");
    const [originalEndpoint, setOriginalEndpoint] = useState("");
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        if (!open) return;
        setLoading(true);
        setApiKeyTouched(false);
        setShowApiKey(false);
        configApi
            .getConfig()
            .then((cfg) => {
                setApiKey(cfg.apiKeyMasked);
                setModel(cfg.model);
                setEndpoint(cfg.endpoint);
                setOriginalModel(cfg.model);
                setOriginalEndpoint(cfg.endpoint);
            })
            .catch((err) => {
                toast.error("Failed to load config: " + String(err));
            })
            .finally(() => setLoading(false));
    }, [open]);

    const handleSave = async () => {
        const req: Record<string, string> = {};
        if (apiKeyTouched && apiKey) {
            req.apiKey = apiKey;
        }
        if (model !== originalModel) {
            req.model = model;
        }
        if (endpoint !== originalEndpoint) {
            req.endpoint = endpoint;
        }

        if (Object.keys(req).length === 0) {
            onOpenChange(false);
            return;
        }

        setSaving(true);
        try {
            const res = await configApi.updateConfig(req);
            toast.success("Settings saved");
            if (res.warnings) {
                for (const w of res.warnings) {
                    toast.warning(w);
                }
            }
            onOpenChange(false);
        } catch (err) {
            toast.error("Failed to save: " + String(err));
        } finally {
            setSaving(false);
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="max-w-md">
                <DialogHeader>
                    <DialogTitle>Settings</DialogTitle>
                    <DialogDescription>
                        Manage your API connection
                    </DialogDescription>
                </DialogHeader>

                {loading ? (
                    <div className="flex items-center justify-center py-8">
                        <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
                    </div>
                ) : (
                    <div className="space-y-4">
                        <div>
                            <Label className="mb-2 block">API Key</Label>
                            <div className="flex gap-2">
                                <Input
                                    type={showApiKey ? "text" : "password"}
                                    value={apiKey}
                                    onChange={(e) => {
                                        setApiKey(e.target.value);
                                        setApiKeyTouched(true);
                                    }}
                                    placeholder="Enter API key"
                                />
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="icon"
                                    onClick={() => setShowApiKey(!showApiKey)}
                                    className="shrink-0"
                                >
                                    {showApiKey ? (
                                        <EyeOff className="w-4 h-4" />
                                    ) : (
                                        <Eye className="w-4 h-4" />
                                    )}
                                </Button>
                            </div>
                        </div>

                        <div>
                            <Label className="mb-2 block">Embedding Model</Label>
                            <Input
                                type="text"
                                value={model}
                                onChange={(e) => setModel(e.target.value)}
                                placeholder="voyage-multimodal-3"
                            />
                        </div>

                        <div>
                            <Label className="mb-2 block">API Endpoint</Label>
                            <Input
                                type="text"
                                value={endpoint}
                                onChange={(e) => setEndpoint(e.target.value)}
                                placeholder="https://api.voyageai.com/v1/multimodalembeddings"
                            />
                        </div>

                        <div className="flex gap-2">
                            <Button
                                type="button"
                                variant="outline"
                                onClick={() => onOpenChange(false)}
                                disabled={saving}
                                className="flex-1"
                            >
                                Cancel
                            </Button>
                            <Button
                                type="button"
                                onClick={handleSave}
                                disabled={saving}
                                className="flex-1"
                            >
                                {saving ? "Saving..." : "Save"}
                            </Button>
                        </div>
                    </div>
                )}
            </DialogContent>
        </Dialog>
    );
}
