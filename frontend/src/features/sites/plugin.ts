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
