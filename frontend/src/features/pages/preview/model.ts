export type PreviewState = "public" | "draft" | "draft-needs-plugin" | "draft-plugin-outdated" | "not-on-site" | "archived";

export interface PreviewedPage {
    status: string;
    wpId: number | null;
}

export interface SitePlugin {
    installed: boolean;
    capabilities: readonly string[] | null;
}

export const previewCapability = "preview";

export const viewports = {
    desktop: 1280,
    tablet: 820,
    phone: 390,
} as const;

export type Viewport = keyof typeof viewports;

export function previewState(page: PreviewedPage, plugin: SitePlugin | null): PreviewState {
    if (page.status === "archived") {
        return "archived";
    }
    if (page.status === "planned" || page.wpId === null) {
        return "not-on-site";
    }
    if (page.status === "published") {
        return "public";
    }
    if (plugin === null) {
        return "draft";
    }
    if (!plugin.installed) {
        return "draft-needs-plugin";
    }
    return (plugin.capabilities ?? []).includes(previewCapability) ? "draft" : "draft-plugin-outdated";
}

export function expiresInMinutes(expiresAt: string | null, now: number): number | null {
    if (expiresAt === null) {
        return null;
    }
    const at = Date.parse(expiresAt);
    if (Number.isNaN(at)) {
        return null;
    }
    return Math.max(0, Math.floor((at - now) / 60_000));
}

export function frameScale(available: number, width: number): number {
    if (available <= 0 || width <= 0) {
        return 1;
    }
    return Math.min(1, available / width);
}

export function editorPreviewUrl(baseUrl: string, wpType: string, wpId: number | null): string | null {
    if (wpId === null || baseUrl === "") {
        return null;
    }
    const key = wpType === "page" ? "page_id" : "p";
    return `${baseUrl.replace(/\/+$/, "")}/?${key}=${wpId}&preview=true`;
}
