import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { failure } from "../../../data/errors.js";
import { usePageModel } from "../../../data/hooks/pages.js";
import { SkeletonRows } from "../../../ui/index.js";
import { BlocksPane } from "./blocks.js";
import { FindingsPane } from "./findings.js";
import { HierarchyPane } from "./hierarchy.js";
import { IdentityPane } from "./identity.js";
import { ModelLinksPane } from "./links.js";
import { ReasonsPane } from "./reasons.js";
import { RenderPane } from "./render.js";
import { TrailsPane } from "./trails.js";

export interface PageModelTabProps {
    pageId: string;
    siteId: string;
    search: string;
}

export function PageModelTab({ pageId, siteId, search }: PageModelTabProps): ReactElement {
    const model = usePageModel(pageId);

    if (model.isPending) {
        return (
            <div className="p-3">
                <SkeletonRows rows={8} label={copy.pages.model.loading} />
            </div>
        );
    }
    const held = model.data;
    if (held === undefined) {
        return (
            <div className="p-3">
                <p className="text-sm text-ink">{failure(model.error).message}</p>
            </div>
        );
    }

    const onSite = held.wpId !== null && held.wpId !== 0;
    const render = held.render;

    return (
        <div className="flex flex-col gap-3 p-3">
            <IdentityPane model={held} onSite={onSite} />
            <HierarchyPane model={held} siteId={siteId} search={search} />
            <ReasonsPane model={held} />
            {onSite ? (
                <>
                    <RenderPane render={render} />
                    {render === null ? null : <TrailsPane trails={render.trails ?? []} siteId={siteId} search={search} />}
                    <ModelLinksPane links={held.links ?? []} siteId={siteId} search={search} />
                    <BlocksPane bodyKind={held.bodyKind} blocks={held.blocks ?? []} />
                </>
            ) : null}
            <FindingsPane findings={held.findings ?? []} />
        </div>
    );
}
