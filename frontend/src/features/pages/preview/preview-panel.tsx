import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { useSite } from "../../../data/hooks/sites.js";
import type { Page } from "../../../data/types.js";
import { Button, ExtensionOffIcon, Panel, PanelHeader, PreviewIcon, PublicIcon } from "../../../ui/index.js";
import type { PreviewState } from "./model.js";
import { editorPreviewUrl, previewState } from "./model.js";
import { SitePreviewDrawer } from "./site-preview.js";

function sentenceOf(state: PreviewState): { title: string; body: string } {
    const said = copy.pages.preview;
    switch (state) {
        case "public":
            return { title: said.public, body: said.publicBody };
        case "draft":
            return { title: said.draft, body: said.draftBody };
        case "draft-needs-plugin":
            return { title: said.needsPlugin, body: said.needsPluginBody };
        case "draft-plugin-outdated":
            return { title: said.outdatedPlugin, body: said.outdatedPluginBody };
        case "archived":
            return { title: said.archived, body: said.archivedBody };
        default:
            return { title: said.notOnSite, body: said.notOnSiteBody };
    }
}

export interface PreviewPanelProps {
    page: Page;
    siteId: string;
}

export function PreviewPanel({ page, siteId }: PreviewPanelProps): ReactElement {
    const site = useSite(siteId);
    const [open, setOpen] = useState(false);
    const plugin = site.data?.site.plugin ?? null;
    const editorUrl = editorPreviewUrl(site.data?.site.baseUrl ?? "", page.wpType, page.wpId);
    const state = previewState({ status: page.status, wpId: page.wpId }, plugin);
    const sentence = sentenceOf(state);
    const canOpen = state === "public" || state === "draft";
    const Icon = state === "draft-needs-plugin" || state === "draft-plugin-outdated" ? ExtensionOffIcon : state === "public" ? PublicIcon : PreviewIcon;

    return (
        <Panel>
            <PanelHeader title={copy.pages.preview.panel} />
            <div className="flex items-start gap-3 p-3">
                <Icon size={18} className={canOpen ? "mt-px shrink-0 text-accent" : "mt-px shrink-0 text-ink-faint"} />
                <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                    <p className="text-sm font-semibold text-ink">{sentence.title}</p>
                    <p className="text-xs text-ink-dim">{sentence.body}</p>
                </div>
                <Button
                    variant={canOpen ? "primary" : "secondary"}
                    disabled={!canOpen}
                    onClick={() => {
                        setOpen(true);
                    }}
                >
                    {state === "public" ? copy.pages.preview.open : copy.pages.preview.openDraft}
                </Button>
            </div>
            {open ? (
                <SitePreviewDrawer
                    page={page}
                    editorUrl={editorUrl}
                    onClose={() => {
                        setOpen(false);
                    }}
                />
            ) : null}
        </Panel>
    );
}
