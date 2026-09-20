import { useMutation } from "@tanstack/react-query";

import { openBrowser } from "../endpoints/browser.js";

export function useOpenExternal() {
    return useMutation({
        mutationFn: (request: Parameters<typeof openBrowser>[0]) => openBrowser(request),
    });
}
