import type { ReactElement, ReactNode } from "react";

import { copy } from "../../copy/index.js";
import type { JsonObject } from "../../domain/merge-patch.js";
import { cx, RestartAltIcon, StatusBadge } from "../../ui/index.js";
import type { Tone } from "../../ui/index.js";
import type { Layer, SpecPath } from "./patch.js";
import { layerAt } from "./patch.js";
import type { SpecDraft } from "./spec.js";

export interface LayerView {
    editing: Layer;
    site: JsonObject | null;
    page: JsonObject | null;
    base: SpecDraft;
}

const layerTone: Readonly<Record<Layer, Tone>> = { global: "muted", site: "accent", page: "info" };

const layerName: Readonly<Record<Layer, string>> = {
    global: copy.templates.layer.fromGlobal,
    site: copy.templates.layer.fromSite,
    page: copy.templates.layer.fromPage,
};

export function markOf(layers: LayerView, path: SpecPath): Layer {
    return layers.editing === "global" ? "global" : layerAt(path, layers.site, layers.page);
}

export interface LayerBadgeProps {
    layer: Layer;
}

export function LayerBadge({ layer }: LayerBadgeProps): ReactElement {
    return (
        <StatusBadge tone={layerTone[layer]} dot={layer !== "global"}>
            {layerName[layer]}
        </StatusBadge>
    );
}

export interface LayerFieldProps {
    layers: LayerView;
    path: SpecPath;
    templateValue: string;
    onFollow: () => void;
    children: ReactNode;
    className?: string;
}

export function LayerField({
    layers,
    path,
    templateValue,
    onFollow,
    children,
    className,
}: LayerFieldProps): ReactElement {
    const layer = markOf(layers, path);
    if (layer === "global") {
        return <div className={className}>{children}</div>;
    }
    const fromPage = layers.editing === "page";
    return (
        <div className={cx("border-l-2 pl-2.5", fromPage ? "border-info-border" : "border-accent-border", className)}>
            {children}
            <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-2xs">
                <span className={cx("font-semibold", fromPage ? "text-info" : "text-accent")}>{layerName[layer]}</span>
                <span className="text-ink-faint">
                    {fromPage ? copy.templates.layer.siteSays(templateValue) : copy.templates.layer.templateSays(templateValue)}
                </span>
                <button
                    type="button"
                    onClick={onFollow}
                    className="inline-flex items-center gap-0.5 font-medium text-ink-dim underline-offset-2 hover:text-ink hover:underline"
                >
                    <RestartAltIcon size={12} className="shrink-0" />
                    {fromPage ? copy.templates.layer.followSite : copy.templates.layer.follow}
                </button>
            </div>
        </div>
    );
}
