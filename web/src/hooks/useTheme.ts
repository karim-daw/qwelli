import { useState, useEffect, useCallback } from "react";

type Theme = "dark" | "light";

export function useTheme() {
    const [theme, setThemeState] = useState<Theme>(() => {
        const stored = localStorage.getItem("qwelli_theme");
        return (stored === "light" ? "light" : "dark") as Theme;
    });

    useEffect(() => {
        const root = document.documentElement;
        if (theme === "dark") {
            root.classList.add("dark");
            root.classList.remove("light");
        } else {
            root.classList.add("light");
            root.classList.remove("dark");
        }
        localStorage.setItem("qwelli_theme", theme);
    }, [theme]);

    const toggleTheme = useCallback(() => {
        setThemeState((prev) => (prev === "dark" ? "light" : "dark"));
    }, []);

    const isDark = theme === "dark";

    return { theme, isDark, toggleTheme, setTheme: setThemeState };
}
