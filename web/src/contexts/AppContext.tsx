import { createContext, useContext, useState, useMemo, type ReactNode } from "react";
import type { Index } from "@/types/index";

type ViewMode = "search" | "status" | "chat";

interface AppContextValue {
    indexes: Index[];
    selectedIndex: string;
    viewMode: ViewMode;
    needsSetup: boolean | null;
    quitConfirmed: boolean;
    setIndexes: (indexes: Index[]) => void;
    setSelectedIndex: (index: string) => void;
    setViewMode: (mode: ViewMode) => void;
    setNeedsSetup: (needs: boolean | null) => void;
    setQuitConfirmed: (confirmed: boolean) => void;
}

const AppContext = createContext<AppContextValue | null>(null);

export function AppProvider({ children }: { children: ReactNode }) {
    const [indexes, setIndexes] = useState<Index[]>([]);
    const [selectedIndex, setSelectedIndex] = useState("");
    const [viewMode, setViewMode] = useState<ViewMode>("search");
    const [needsSetup, setNeedsSetup] = useState<boolean | null>(null);
    const [quitConfirmed, setQuitConfirmed] = useState(false);

    const value = useMemo(
        () => ({
            indexes,
            selectedIndex,
            viewMode,
            needsSetup,
            quitConfirmed,
            setIndexes,
            setSelectedIndex,
            setViewMode,
            setNeedsSetup,
            setQuitConfirmed,
        }),
        [indexes, selectedIndex, viewMode, needsSetup, quitConfirmed],
    );

    return (
        <AppContext.Provider value={value}>
            {children}
        </AppContext.Provider>
    );
}

export function useAppContext() {
    const context = useContext(AppContext);
    if (!context) {
        throw new Error("useAppContext must be used within an AppProvider");
    }
    return context;
}
