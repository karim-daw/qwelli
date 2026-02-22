import { CheckCircle } from "lucide-react";

export function QuitScreen() {
    return (
        <div className="flex h-screen bg-background text-foreground items-center justify-center">
            <div className="text-center">
                <CheckCircle className="w-16 h-16 mx-auto mb-4 text-green-500 dark:text-green-400" />
                <p className="text-lg font-medium">Qwelli has been stopped</p>
                <p className="text-sm text-muted-foreground mt-1">
                    You can close this tab
                </p>
            </div>
        </div>
    );
}
