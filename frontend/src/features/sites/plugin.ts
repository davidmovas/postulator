import { copy } from "../../copy/index.js";

const capabilities = copy.sites.plugin.capabilityNames as Readonly<Record<string, string>>;
const seoPlugins = copy.sites.plugin.seoNames as Readonly<Record<string, string>>;

export function capabilityLabel(capability: string): string {
    return capabilities[capability] ?? capability.split("_").join(" ");
}

export function seoPluginLabel(name: string): string {
    return seoPlugins[name] ?? name;
}
