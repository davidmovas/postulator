export const sheetExtensions = [".csv", ".xlsx"] as const;

export function sheetIn(paths: readonly string[]): string | null {
    for (const path of paths) {
        const trimmed = path.trim();
        const lower = trimmed.toLowerCase();
        if (trimmed !== "" && sheetExtensions.some((extension) => lower.endsWith(extension))) {
            return trimmed;
        }
    }
    return null;
}

export function siteImportedOn(pathname: string): string | null {
    const matched = /^\/s\/([^/]+)\/import$/.exec(pathname);
    return matched === null ? null : matched[1];
}

type DropListener = (paths: readonly string[]) => void;

const listeners = new Set<DropListener>();

export function publishDrop(paths: readonly string[]): void {
    listeners.forEach((listener) => {
        listener(paths);
    });
}

export function subscribeDrop(listener: DropListener): () => void {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
}
