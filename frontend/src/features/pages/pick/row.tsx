import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import type { Page } from "../../../data/types.js";
import { AccountTreeIcon, Checkbox, ChevronRightIcon, cx, IconButton, StatusBadge } from "../../../ui/index.js";
import { pageStatusLabel, statusTone as pageStatusTone } from "../labels.js";
import type { Tick } from "./model.js";

const indentPx = 14;

export interface PageRowProps {
    page: Page;
    depth: number;
    childCount: number;
    expanded: boolean;
    tick: Tick;
    refusal: string | null;
    note: string | null;
    onToggle: () => void;
    onExpand: () => void;
    onBranch: () => void;
}

export function segmentOf(path: string): string {
    const parts = path.split("/").filter((part) => part !== "");
    return parts.length === 0 ? "/" : `${parts[parts.length - 1]}/`;
}

export function PageRow({
    page,
    depth,
    childCount,
    expanded,
    tick,
    refusal,
    note,
    onToggle,
    onExpand,
    onBranch,
}: PageRowProps): ReactElement {
    return (
        <div
            data-page-pick={page.id}
            className="flex h-7 shrink-0 items-center gap-1.5 rounded-sm pr-1 hover:bg-raised"
            style={{ paddingLeft: depth * indentPx }}
            title={refusal ?? page.path}
        >
            {childCount > 0 ? (
                <button
                    type="button"
                    aria-label={expanded ? copy.pages.pick.collapse : copy.pages.pick.expand}
                    aria-expanded={expanded}
                    className="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-sm text-ink-dim hover:bg-raised hover:text-ink"
                    onClick={onExpand}
                >
                    <ChevronRightIcon size={14} className={cx("transition-transform", expanded && "rotate-90")} />
                </button>
            ) : (
                <span aria-hidden={true} className="w-4 shrink-0" />
            )}
            <Checkbox
                checked={tick === "on"}
                indeterminate={tick === "some"}
                disabled={refusal !== null}
                label={segmentOf(page.path)}
                onChange={onToggle}
            />
            <span className="min-w-0 flex-1 truncate text-xs text-ink-faint">{page.title}</span>
            {note === null ? null : <span className="shrink-0 text-2xs text-accent">{note}</span>}
            <StatusBadge tone={pageStatusTone(page.status)} dot={false}>
                {pageStatusLabel(page.status)}
            </StatusBadge>
            {childCount > 0 ? (
                <IconButton
                    size="sm"
                    variant="ghost"
                    icon={AccountTreeIcon}
                    label={copy.pages.pick.branch}
                    title={copy.pages.pick.branch}
                    onClick={onBranch}
                />
            ) : null}
        </div>
    );
}
