import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import type { SiteMapNode } from "../../../data/types.js";
import { Checkbox, ChevronRightIcon, cx, shiftHeld } from "../../../ui/index.js";
import { PageStatusBadge } from "../badges.js";
import type { Tick } from "./model.js";

const indentPx = 14;

export interface PageRowProps {
    node: SiteMapNode;
    label: string;
    depth: number;
    expanded: boolean | null;
    tick: Tick;
    refusal: string | null;
    note: string | null;
    onToggle: (ranged: boolean) => void;
    onExpand: () => void;
}

export function PageRow({ node, label, depth, expanded, tick, refusal, note, onToggle, onExpand }: PageRowProps): ReactElement {
    return (
        <div
            data-page-pick={node.page.pageId}
            className="flex h-7 shrink-0 items-center gap-1.5 rounded-sm pr-1 hover:bg-raised"
            style={{ paddingLeft: depth * indentPx }}
            title={refusal ?? node.page.path}
            onMouseDown={(event) => {
                if (event.shiftKey) {
                    event.preventDefault();
                }
            }}
        >
            {expanded === null ? (
                <span aria-hidden={true} className="w-4 shrink-0" />
            ) : (
                <button
                    type="button"
                    aria-label={expanded ? copy.pages.pick.collapse : copy.pages.pick.expand}
                    aria-expanded={expanded}
                    className="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-sm text-ink-dim hover:bg-raised hover:text-ink"
                    onClick={onExpand}
                >
                    <ChevronRightIcon size={14} className={cx("transition-transform", expanded && "rotate-90")} />
                </button>
            )}
            <Checkbox
                checked={tick === "on"}
                indeterminate={tick === "some"}
                disabled={refusal !== null}
                label={label}
                onChange={(event) => {
                    onToggle(shiftHeld(event));
                }}
            />
            <span className="min-w-0 flex-1 truncate text-xs text-ink-faint">{node.page.title}</span>
            {note === null ? null : <span className="shrink-0 text-2xs text-accent">{note}</span>}
            <PageStatusBadge status={node.status} dot={false} />
        </div>
    );
}
