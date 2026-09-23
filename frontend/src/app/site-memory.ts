const storageKey = "postulator.site.last";

export function readLastSite(): string | null {
    try {
        const held = window.localStorage.getItem(storageKey);
        return held === null || held === "" ? null : held;
    } catch {
        return null;
    }
}

export function rememberSite(siteId: string): void {
    try {
        window.localStorage.setItem(storageKey, siteId);
    } catch {
        return;
    }
}

export function landing(lastSiteId: string | null, siteIds: readonly string[]): string {
    if (siteIds.length === 0) {
        return "/sites";
    }
    const opened = lastSiteId !== null && siteIds.includes(lastSiteId) ? lastSiteId : siteIds[0];
    return `/s/${opened}/overview`;
}
