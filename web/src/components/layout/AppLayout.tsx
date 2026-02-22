import { useState, useEffect } from "react";
import { Sidebar } from "./Sidebar";
import { TopBar } from "./TopBar";
import { MainContent } from "./MainContent";
import { TerminalPanel } from "@/components/terminal/TerminalPanel";
import { NewIndexDialog } from "@/components/modals/NewIndexDialog";
import { IndexProgressModal } from "@/components/modals/IndexProgressModal";
import { SettingsDialog } from "@/components/modals/SettingsDialog";
import { useAppContext } from "@/contexts/AppContext";
import { useIndexes } from "@/hooks/useIndexes";
import { useTerminal } from "@/hooks/useTerminal";
import { useResizable } from "@/hooks/useResizable";
import { useIndexProgress } from "@/hooks/useIndexProgress";
import * as serverApi from "@/api/server";
import * as indexesApi from "@/api/indexes";
import { toast } from "sonner";

export function AppLayout() {
    const { setQuitConfirmed } = useAppContext();
    const { fetchIndexes } = useIndexes();
    const terminal = useTerminal();
    const indexProgress = useIndexProgress();
    const sidebar = useResizable({
        direction: "horizontal",
        initialSize: 256,
        minSize: 200,
        maxSize: 500,
    });
    const terminalResize = useResizable({
        direction: "vertical",
        initialSize: 200,
        minSize: 100,
        maxSize: 600,
    });

    const [showSidebar, setShowSidebar] = useState(true);
    const [showTerminal, setShowTerminal] = useState(false);
    const [showNewIndexDialog, setShowNewIndexDialog] = useState(false);
    const [showSettingsDialog, setShowSettingsDialog] = useState(false);

    useEffect(() => {
        fetchIndexes();
    }, [fetchIndexes]);

    const handleQuit = async () => {
        await serverApi.shutdown();
        setQuitConfirmed(true);
    };

    const handleCreateIndex = async (path: string, contentType: string) => {
        indexProgress.start(path, {
            onComplete: fetchIndexes,
            onCancelled: () => setShowNewIndexDialog(false),
        });

        try {
            const response = await indexesApi.createIndex(path, contentType);
            if (!response.ok) {
                const data = await response.json();
                indexProgress.reset();
                toast.error("Error: " + data.error);
            }
        } catch (error) {
            indexProgress.reset();
            toast.error("Failed to create index: " + String(error));
        }
    };

    const handleCloseProgress = () => {
        indexProgress.reset();
        setShowNewIndexDialog(false);
    };

    return (
        <div className="flex h-screen bg-background text-foreground overflow-hidden">
            {showSidebar && (
                <Sidebar
                    width={sidebar.size}
                    onResizeStart={sidebar.onMouseDown}
                    onShowNewIndexDialog={() => setShowNewIndexDialog(true)}
                />
            )}

            <div className="flex-1 flex flex-col overflow-hidden">
                <TopBar
                    onToggleSidebar={() => setShowSidebar(!showSidebar)}
                    onToggleTerminal={() => setShowTerminal(!showTerminal)}
                    showTerminal={showTerminal}
                    onOpenSettings={() => setShowSettingsDialog(true)}
                    onQuit={handleQuit}
                />
                <MainContent />
            </div>

            <SettingsDialog
                open={showSettingsDialog}
                onOpenChange={setShowSettingsDialog}
            />

            <NewIndexDialog
                open={showNewIndexDialog}
                onOpenChange={setShowNewIndexDialog}
                onSubmit={handleCreateIndex}
                indexing={indexProgress.isRunning}
            />

            {indexProgress.progress && (
                <IndexProgressModal
                    progress={indexProgress.progress}
                    phase={indexProgress.phase}
                    isComplete={indexProgress.isComplete}
                    cancelling={indexProgress.cancelling}
                    onCancel={indexProgress.cancel}
                    onClose={handleCloseProgress}
                />
            )}

            {showTerminal && (
                <TerminalPanel
                    logs={terminal.logs}
                    height={terminalResize.size}
                    onResizeStart={terminalResize.onMouseDown}
                    onClose={() => setShowTerminal(false)}
                    onClear={terminal.clearLogs}
                />
            )}
        </div>
    );
}
