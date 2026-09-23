import { Dialogs } from "@wailsio/runtime";

import { copy } from "../copy/index.js";
import { announceThrown } from "./client.js";
import { openBrowser } from "./endpoints/browser.js";
import { isTorClosedToLinks } from "./errors.js";
import { pushToast } from "./toasts.js";

export interface HostFilter {
    displayName: string;
    pattern: string;
}

export interface HostPickOptions {
    title?: string;
    message?: string;
    buttonText?: string;
    directory?: string;
    filename?: string;
    filters?: readonly HostFilter[];
}

function toFilters(filters: readonly HostFilter[] | undefined) {
    return filters?.map((held) => ({ DisplayName: held.displayName, Pattern: held.pattern }));
}

function normalise(path: string): string | null {
    return path === "" ? null : path;
}

export async function pickOpenFile(options: HostPickOptions = {}): Promise<string | null> {
    const picked = await Dialogs.OpenFile({
        Title: options.title,
        Message: options.message,
        ButtonText: options.buttonText,
        Directory: options.directory,
        Filters: toFilters(options.filters),
        CanChooseFiles: true,
        CanChooseDirectories: false,
        AllowsMultipleSelection: false,
    });
    return normalise(picked);
}

export async function pickSaveFile(options: HostPickOptions = {}): Promise<string | null> {
    const picked = await Dialogs.SaveFile({
        Title: options.title,
        Message: options.message,
        ButtonText: options.buttonText,
        Directory: options.directory,
        Filename: options.filename,
        Filters: toFilters(options.filters),
        CanCreateDirectories: true,
    });
    return normalise(picked);
}

const browsable: ReadonlySet<string> = new Set<string>(["http:", "https:"]);

export function isBrowsable(url: string): boolean {
    try {
        return browsable.has(new URL(url).protocol);
    } catch {
        return false;
    }
}

export async function openExternal(url: string): Promise<boolean> {
    if (!isBrowsable(url)) {
        return false;
    }
    try {
        await openBrowser({ url });
        return true;
    } catch (thrown) {
        if (isTorClosedToLinks(thrown)) {
            pushToast("warning", (await copied(url)) ? copy.app.torClosedCopied : copy.app.torClosedToLinks, null, null);
            return false;
        }
        announceThrown(thrown);
        return false;
    }
}

async function copied(text: string): Promise<boolean> {
    try {
        await navigator.clipboard.writeText(text);
        return true;
    } catch {
        return false;
    }
}
