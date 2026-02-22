import { Menu, ChevronRight, Sun, Moon, Power, Terminal as TerminalIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { SearchForm } from "@/components/search/SearchForm";
import { useAppContext } from "@/contexts/AppContext";
import { useTheme } from "@/hooks/useTheme";

interface TopBarProps {
    onToggleSidebar: () => void;
    onToggleTerminal: () => void;
    showTerminal: boolean;
    onQuit: () => void;
}

export function TopBar({
    onToggleSidebar,
    onToggleTerminal,
    showTerminal,
    onQuit,
}: TopBarProps) {
    const { indexes, selectedIndex, viewMode } = useAppContext();
    const { isDark, toggleTheme } = useTheme();

    const currentIndex = indexes.find((i) => i.path === selectedIndex);

    return (
        <div className="flex-shrink-0 border-b bg-background/80 backdrop-blur-xl">
            <div className="px-6 py-4">
                <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center gap-4">
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={onToggleSidebar}
                        >
                            <Menu className="w-5 h-5" />
                        </Button>
                        <h1 className="text-lg font-medium">Qwelli</h1>
                        {currentIndex && (
                            <div className="flex items-center gap-2 text-sm text-muted-foreground">
                                <ChevronRight className="w-4 h-4" />
                                <span>{currentIndex.name}</span>
                            </div>
                        )}
                    </div>
                    <div className="flex items-center gap-2">
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    onClick={onToggleTerminal}
                                    className={showTerminal ? "bg-accent" : ""}
                                >
                                    <TerminalIcon className="w-5 h-5" />
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent>Toggle terminal</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    onClick={toggleTheme}
                                >
                                    {isDark ? (
                                        <Sun className="w-5 h-5" />
                                    ) : (
                                        <Moon className="w-5 h-5" />
                                    )}
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent>
                                Switch to {isDark ? "light" : "dark"} mode
                            </TooltipContent>
                        </Tooltip>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    onClick={onQuit}
                                    className="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300"
                                >
                                    <Power className="w-5 h-5" />
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent>Quit Qwelli</TooltipContent>
                        </Tooltip>
                    </div>
                </div>

                {viewMode === "search" && <SearchForm />}
            </div>
        </div>
    );
}
