import type { ReactElement } from "react";
import { useMemo } from "react";
import { Link } from "react-router";

import { copy } from "../../../copy/index.js";
import { useSiteMap } from "../../../data/hooks/pages.js";
import type { PageModel } from "../../../data/types.js";
import { Panel, PanelHeader, StatusBadge } from "../../../ui/index.js";
import { findingsAddress } from "../../findings/model/params.js";
import { whySentence } from "../../map/labels.js";
import { Note, PageRef } from "./facts.js";
import { parentReasonSentence } from "./labels.js";

const said = copy.pages.model.hierarchy;

const siteRoot = "/";

function NoParent({ model, siteId }: { model: PageModel; siteId: string }): ReactElement {
    const found = useSiteMap({ siteId, search: model.path }, model.path !== "");
    const pages = found.data?.pages;
    const why = useMemo(
        () =>
            (pages ?? []).flatMap((view) => view.nodes.items ?? []).find((node) => node.page.pageId === model.pageId)
                ?.why ?? null,
        [pages, model.pageId],
    );
    const sentence = whySentence(why);
    const findingId = why?.findingId ?? "";
    return (
        <div className="flex flex-col gap-1 px-3 py-2" data-page-why={why?.reason ?? ""}>
            <span className="text-xs text-ink-soft">{sentence ?? said.top}</span>
            {findingId === "" ? null : (
                <Link to={findingsAddress(siteId, model.pageId)} className="w-fit text-2xs text-accent hover:underline">
                    {copy.map.why.finding}
                </Link>
            )}
        </div>
    );
}

export interface HierarchyPaneProps {
    model: PageModel;
    siteId: string;
    search: string;
}

export function HierarchyPane({ model, siteId, search }: HierarchyPaneProps): ReactElement {
    const hierarchy = model.hierarchy;
    const label =
        hierarchy.parentTitle !== "" ? hierarchy.parentTitle : hierarchy.parentPath !== "" ? hierarchy.parentPath : hierarchy.parentId;

    return (
        <Panel>
            <PanelHeader title={said.title} />
            {model.path === siteRoot ? (
                <Note>{said.front}</Note>
            ) : hierarchy.parentId === "" ? (
                <NoParent model={model} siteId={siteId} />
            ) : (
                <div className="flex flex-col gap-1 px-3 py-2">
                    <div className="flex min-w-0 items-center gap-2">
                        <PageRef
                            pageId={hierarchy.parentId}
                            siteId={siteId}
                            search={search}
                            label={label}
                            path={hierarchy.parentPath === "" ? undefined : hierarchy.parentPath}
                        />
                        {hierarchy.parentPath === siteRoot ? (
                            <StatusBadge tone="muted" dot={false}>
                                {said.root}
                            </StatusBadge>
                        ) : null}
                    </div>
                    <span className="text-2xs text-ink-dim">{parentReasonSentence(hierarchy.reason, hierarchy.method)}</span>
                </div>
            )}
        </Panel>
    );
}
