import { useEffect } from "react";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AppProvider } from "@/contexts/AppContext";
import { AppMetaProvider, useAppMetaContext } from "@/contexts/AppMetaContext";
import { SearchProvider } from "@/contexts/SearchContext";
import { SetupScreen } from "@/components/screens/SetupScreen";
import { QuitScreen } from "@/components/screens/QuitScreen";
import { AppLayout } from "@/components/layout/AppLayout";
import { useTheme } from "@/hooks/useTheme";
import * as setupApi from "@/api/setup";

function AppContent() {
    const { needsSetup, setNeedsSetup, quitConfirmed } = useAppMetaContext();
    useTheme();

    useEffect(() => {
        setupApi
            .getSetupStatus()
            .then((data) => setNeedsSetup(data.needsSetup === true))
            .catch(() => setNeedsSetup(false));
    }, [setNeedsSetup]);

    if (needsSetup === null) return null;
    if (needsSetup) return <SetupScreen />;
    if (quitConfirmed) return <QuitScreen />;

    return (
        <SearchProvider>
            <AppLayout />
        </SearchProvider>
    );
}

export default function App() {
    return (
        <TooltipProvider>
            <AppMetaProvider>
                <AppProvider>
                    <AppContent />
                    <Toaster />
                </AppProvider>
            </AppMetaProvider>
        </TooltipProvider>
    );
}
