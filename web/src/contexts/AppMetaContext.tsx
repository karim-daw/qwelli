import { createContext, useContext, useState, useMemo, type ReactNode } from "react";

interface AppMetaContextValue {
    needsSetup: boolean | null;
    quitConfirmed: boolean;
    setNeedsSetup: (needs: boolean | null) => void;
    setQuitConfirmed: (confirmed: boolean) => void;
}

const AppMetaContext = createContext<AppMetaContextValue | null>(null);

export function AppMetaProvider({ children }: { children: ReactNode }) {
    const [needsSetup, setNeedsSetup] = useState<boolean | null>(null);
    const [quitConfirmed, setQuitConfirmed] = useState(false);

    const value = useMemo(
        () => ({ needsSetup, quitConfirmed, setNeedsSetup, setQuitConfirmed }),
        [needsSetup, quitConfirmed],
    );

    return (
        <AppMetaContext.Provider value={value}>
            {children}
        </AppMetaContext.Provider>
    );
}

export function useAppMetaContext() {
    const context = useContext(AppMetaContext);
    if (!context) {
        throw new Error("useAppMetaContext must be used within an AppMetaProvider");
    }
    return context;
}
