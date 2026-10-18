import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import type { PageModel } from "../../../data/types.js";
import { Panel, PanelHeader, StatusBadge } from "../../../ui/index.js";
import { Note, PageRef } from "./facts.js";
import { parentReasonSentence } from "./labels.js";

const said = copy.pages.model.hierarchy;

const siteRoot = "/";

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
                <Note>{said.top}</Note>
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
