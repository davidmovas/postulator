import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { Page } from "../../data/types.js";
import { StatusBadge, SyncProblemIcon } from "../../ui/index.js";
import { pageStatusLabel, pageStatusTone } from "./labels.js";

export interface PageStatusBadgeProps {
    status: string;
    dot?: boolean;
}

export function PageStatusBadge({ status, dot }: PageStatusBadgeProps): ReactElement {
    return (
        <StatusBadge tone={pageStatusTone(status)} dot={dot}>
            {pageStatusLabel(status)}
        </StatusBadge>
    );
}

export function DriftBadge(): ReactElement {
    return (
        <StatusBadge tone="warn" icon={SyncProblemIcon}>
            {copy.pages.drift.badge}
        </StatusBadge>
    );
}

export interface PageStatusBadgesProps {
    page: Pick<Page, "status" | "drift">;
}

export function PageStatusBadges({ page }: PageStatusBadgesProps): ReactElement {
    return (
        <>
            <PageStatusBadge status={page.status} />
            {page.drift ? <DriftBadge /> : null}
        </>
    );
}
