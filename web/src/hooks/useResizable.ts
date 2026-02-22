import { useState, useEffect, useCallback } from "react";

interface UseResizableOptions {
    direction: "horizontal" | "vertical";
    initialSize: number;
    minSize: number;
    maxSize: number;
}

export function useResizable(options: UseResizableOptions) {
    const { direction, initialSize, minSize, maxSize } = options;
    const [size, setSize] = useState(initialSize);
    const [isResizing, setIsResizing] = useState(false);

    const onMouseDown = useCallback(() => {
        setIsResizing(true);
    }, []);

    useEffect(() => {
        if (!isResizing) return;

        const handleMouseMove = (e: MouseEvent) => {
            let newSize: number;
            if (direction === "horizontal") {
                newSize = e.clientX;
            } else {
                newSize = window.innerHeight - e.clientY;
            }
            if (newSize >= minSize && newSize <= maxSize) {
                setSize(newSize);
            }
        };

        const handleMouseUp = () => {
            setIsResizing(false);
        };

        document.addEventListener("mousemove", handleMouseMove);
        document.addEventListener("mouseup", handleMouseUp);

        return () => {
            document.removeEventListener("mousemove", handleMouseMove);
            document.removeEventListener("mouseup", handleMouseUp);
        };
    }, [isResizing, direction, minSize, maxSize]);

    return { size, isResizing, onMouseDown };
}
