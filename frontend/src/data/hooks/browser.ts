import { useMutation, useQueryClient } from "@tanstack/react-query";

import { locateBrowser, openBrowser } from "../endpoints/browser.js";
import { setSetting } from "../endpoints/settings.js";
import { keys } from "../keys.js";
import { useUnlockedQuery } from "../query.js";

export const torPathKey = "browser.torPath";

export function useOpenExternal() {
    return useMutation({
        mutationFn: (request: Parameters<typeof openBrowser>[0]) => openBrowser(request),
    });
}

export function useBrowserLocation() {
    return useUnlockedQuery({
        queryKey: keys.browser.locate(),
        queryFn: ({ signal }) => locateBrowser({}, signal),
    });
}

export function useSetTorPath() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (path: string) => setSetting({ key: torPathKey, value: path }),
        onSuccess: (answered) => {
            void client.invalidateQueries({ queryKey: keys.settings.value(answered.key) });
            void client.invalidateQueries({ queryKey: keys.browser.locate() });
        },
    });
}
