import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { usePage } from "../../../data/hooks/pages.js";
import { useSite } from "../../../data/hooks/sites.js";
import { Button, PreviewIcon } from "../../../ui/index.js";
import { editorPreviewUrl, previewState } from "./model.js";
import { SitePreviewDrawer } from "./site-preview.js";

const reasons = {
    "draft-needs-plugin": copy.pages.preview.needsPlugin,
    "draft-plugin-outdated": copy.pages.preview.outdatedPlugin,
    "not-on-site": copy.pages.preview.notOnSite,
    archived: copy.pages.preview.archived,
} as const;

export interface PreviewButtonProps {
    pageId: string;
    siteId: string;
}

export function PreviewButton({ pageId, siteId }: PreviewButtonProps): ReactElement {
    const detail = usePage(pageId);
    const site = useSite(siteId);
    const [open, setOpen] = useState(false);
    const page = detail.data?.page;

    const state = page === undefined ? null : previewState({ status: page.status, wpId: page.wpId }, site.data?.site.plugin ?? null);
    const canOpen = state === "public" || state === "draft";
    const reason = state === null || canOpen ? undefined : reasons[state];

    return (
        <>
            <Button
                size="sm"
                icon={PreviewIcon}
                disabled={!canOpen}
                title={reason}
                onClick={() => {
                    setOpen(true);
                }}
            >
                {copy.pages.preview.fromReview}
            </Button>
            {open && page !== undefined ? (
                <SitePreviewDrawer
                    page={page}
                    editorUrl={editorPreviewUrl(site.data?.site.baseUrl ?? "", page.wpType, page.wpId)}
                    onClose={() => {
                        setOpen(false);
                    }}
                />
            ) : null}
        </>
    );
}
