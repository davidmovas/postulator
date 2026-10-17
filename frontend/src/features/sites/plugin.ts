import { copy } from "../../copy/index.js";
import type { Tone } from "../../ui/index.js";

export interface StoreBanner {
    tone: Tone;
    title: string;
    body: string;
}

export function storeBanner(commerce: string, pluginInstalled: boolean, username: string): StoreBanner | null {
    const store = copy.sites.store;
    switch (commerce) {
        case "ready":
            return pluginInstalled
                ? { tone: "ok", title: store.readyTitle, body: store.readyBody }
                : { tone: "warn", title: store.readyTitle, body: store.needsPlugin };
        case "forbidden":
            return { tone: "warn", title: store.forbiddenTitle, body: store.forbiddenBody(username) };
        default:
            return null;
    }
}

const capabilities = copy.sites.plugin.capabilityNames as Readonly<Record<string, string>>;
const seoPlugins = copy.sites.plugin.seoNames as Readonly<Record<string, string>>;

export function capabilityLabel(capability: string): string {
    return capabilities[capability] ?? capability.split("_").join(" ");
}

export function seoPluginLabel(name: string): string {
    return seoPlugins[name] ?? name;
}

export interface PluginReport {
    installed: boolean;
    outdated: boolean;
    version: string;
    requiredVersion: string;
    capabilities: readonly string[] | null;
    seoPlugin: string;
}

export interface PluginBanner {
    tone: Tone;
    title: string;
    body: string;
    offersZip: boolean;
}

export function pluginBanner(plugin: PluginReport, siteName: string): PluginBanner {
    const words = copy.sites.plugin;
    if (!plugin.installed) {
        return { tone: "warn", title: words.missingTitle(siteName), body: words.missingBody, offersZip: true };
    }
    if (plugin.outdated) {
        return {
            tone: "warn",
            title: words.outdatedTitle(plugin.version, plugin.requiredVersion),
            body: words.outdatedBody,
            offersZip: true,
        };
    }
    const listed = plugin.capabilities ?? [];
    const named = listed.length === 0 ? words.noCapabilities : listed.map(capabilityLabel).join(", ");
    const seo = plugin.seoPlugin === "" ? words.noSeo : seoPluginLabel(plugin.seoPlugin);
    return {
        tone: "ok",
        title: words.installed(plugin.version),
        body: `${words.capabilities}: ${named} · ${words.seo}: ${seo}`,
        offersZip: false,
    };
}
