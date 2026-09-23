import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { Page } from "../../data/types.js";
import { AccountTreeIcon, Checkbox, ChevronRightIcon, cx, IconButton, StatusBadge } from "../../ui/index.js";
import { pageStatusLabel, statusTone as pageStatusTone } from "../pages/labels.js";
import type { Tick } from "./start-selection.js";

const indentPx = 14;

export interface TargetRowProps {
    page: Page;
    depth: number;
    childCount: number;
    expanded: boolean;
    tick: Tick;
    neededBy: string | null;
    writable: boolean;
    onToggle: () => void;
    onExpand: () => void;
    onBranch: () => void;
}

function segmentOf(path: string): string {
    const parts = path.split("/").filter((part) => part !== "");
    return parts.length === 0 ? "/" : `${parts[parts.length - 1]}/`;
}

function refusalOf(writable: boolean, neededBy: string | null): string | undefined {
    if (!writable) {
        return copy.runs.start.unmapped;
    }
    if (neededBy !== null) {
        return copy.runs.start.required(neededBy);
    }
    return undefined;
}

export function TargetRow({
    page,
    depth,
    childCount,
    expanded,
    tick,
    neededBy,
    writable,
    onToggle,
    onExpand,
    onBranch,
}: TargetRowProps): ReactElement {
    const refusal = refusalOf(writable, neededBy);
    return (
        <div
            data-run-target={page.id}
            className="flex h-7 shrink-0 items-center gap-1.5 rounded-sm pr-1 hover:bg-raised"
            style={{ paddingLeft: depth * indentPx }}
            title={refusal ?? page.path}
        >
            {childCount > 0 ? (
                <button
                    type="button"
                    aria-label={expanded ? copy.runs.start.collapse : copy.runs.start.expand}
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
                disabled={!writable || neededBy !== null}
                label={segmentOf(page.path)}
                onChange={onToggle}
            />
            <span className="min-w-0 flex-1 truncate text-xs text-ink-faint">{page.title}</span>
            {neededBy === null ? null : (
                <span className="shrink-0 text-2xs text-accent">{copy.runs.start.writtenFirst}</span>
            )}
            <StatusBadge tone={pageStatusTone(page.status)} dot={false}>
                {pageStatusLabel(page.status)}
            </StatusBadge>
            {childCount > 0 ? (
                <IconButton
                    size="sm"
                    variant="ghost"
                    icon={AccountTreeIcon}
                    label={copy.runs.start.branch}
                    title={copy.runs.start.branch}
                    onClick={onBranch}
                />
            ) : null}
        </div>
    );
}
