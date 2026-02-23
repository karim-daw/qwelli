import { useCallback, useRef } from "react";
import { useAppContext } from "@/contexts/AppContext";
import * as indexesApi from "@/api/indexes";
import { toast } from "sonner";

export function useIndexes() {
    const { indexes, selectedIndex, setIndexes, setSelectedIndex } =
        useAppContext();

    const selectedIndexRef = useRef(selectedIndex);
    selectedIndexRef.current = selectedIndex;

    const fetchIndexes = useCallback(async () => {
        try {
            const fetched = await indexesApi.listIndexes();
            setIndexes(fetched);
            if (fetched.length > 0 && !selectedIndexRef.current) {
                setSelectedIndex(fetched[0].path);
            }
        } catch (error) {
            console.error("Failed to fetch indexes:", error);
        }
    }, [setIndexes, setSelectedIndex]);

    const deleteIndex = useCallback(
        async (indexPath: string) => {
            try {
                await indexesApi.deleteIndex(indexPath);
                fetchIndexes();
                if (selectedIndex === indexPath) {
                    setSelectedIndex("");
                }
            } catch (error) {
                toast.error("Failed to delete index: " + String(error));
            }
        },
        [fetchIndexes, selectedIndex, setSelectedIndex],
    );

    return { indexes, selectedIndex, fetchIndexes, deleteIndex };
}
