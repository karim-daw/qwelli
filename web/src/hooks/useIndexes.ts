import { useCallback } from "react";
import { useAppContext } from "@/contexts/AppContext";
import * as indexesApi from "@/api/indexes";
import { toast } from "sonner";

export function useIndexes() {
    const { indexes, selectedIndex, setIndexes, setSelectedIndex } =
        useAppContext();

    const fetchIndexes = useCallback(async () => {
        try {
            const fetched = await indexesApi.listIndexes();
            setIndexes(fetched);
            if (fetched.length > 0 && !selectedIndex) {
                setSelectedIndex(fetched[0].path);
            }
        } catch (error) {
            console.error("Failed to fetch indexes:", error);
        }
    }, [setIndexes, setSelectedIndex, selectedIndex]);

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
