import { listTools } from "../endpoints/tools.js";
import { keys } from "../keys.js";
import { useUnlockedQuery } from "../query.js";

export function useTools() {
    return useUnlockedQuery({
        queryKey: keys.tools.list(),
        queryFn: ({ signal }) => listTools({}, signal),
    });
}
