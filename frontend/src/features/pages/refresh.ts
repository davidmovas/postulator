import { useQueryClient } from "@tanstack/react-query";

import { keys } from "../../data/keys.js";

export function usePageDetailRefresh(): (pageId: string) => void {
    const client = useQueryClient();
    return (pageId: string): void => {
        void client.invalidateQueries({ queryKey: keys.pages.detail(pageId) });
        void client.invalidateQueries({ queryKey: keys.graph.entityAll() });
    };
}
