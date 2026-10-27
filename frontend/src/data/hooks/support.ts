import { useMutation } from "@tanstack/react-query";

import { copy } from "../../copy/index.js";
import { quietMeta } from "../client.js";
import { writeBundle } from "../endpoints/support.js";
import { errorMessageOf, failure } from "../errors.js";
import { pickSaveFile } from "../host.js";

export interface SavedBundle {
    path: string;
    bytes: number;
}

const bundleFilter = { displayName: copy.settings.about.support.filter, pattern: "*.json" };

export function bundleName(siteName: string, at: Date): string {
    const slug = siteName
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-+|-+$/g, "");
    const day = at.toISOString().slice(0, 10);
    return slug === "" ? `support-bundle-${day}.json` : `support-bundle-${slug}-${day}.json`;
}

export function bundleRefusal(thrown: unknown): string | null {
    const said = errorMessageOf(thrown);
    if (said === null) {
        return null;
    }
    const reported = failure(thrown).message;
    return reported === "" ? said : reported;
}

export function useSaveBundle() {
    return useMutation({
        mutationFn: async (request: { siteId: string; siteName: string }): Promise<SavedBundle | null> => {
            const path = await pickSaveFile({
                title: copy.settings.about.support.dialogTitle,
                filename: bundleName(request.siteName, new Date()),
                filters: [bundleFilter],
            });
            if (path === null) {
                return null;
            }
            const written = await writeBundle({ siteId: request.siteId, path });
            return { path: written.path, bytes: written.bytes };
        },
        meta: quietMeta(["INVALID", "EXTERNAL", "NOT_FOUND"]),
    });
}
