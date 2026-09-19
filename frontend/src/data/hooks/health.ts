import { useQuery } from "@tanstack/react-query";

import { ping } from "../endpoints/health.js";
import { keys } from "../keys.js";

export function useBuildInfo() {
    return useQuery({
        queryKey: keys.health.ping(),
        queryFn: ({ signal }) => ping({}, signal),
        networkMode: "always",
        staleTime: Infinity,
        retry: false,
    });
}
