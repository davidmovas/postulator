import type { KeyboardEvent, ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import {
    Button,
    CallSplitIcon,
    ChevronRightIcon,
    CountBadge,
    cx,
    DenseTable,
    LinkOffIcon,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
    toneClasses,
    VirtualRows,
} from "../../../ui/index.js";
import { entityIcon, formatScore, kindTone } from "../labels.js";
import type { VisibleRow } from "../model/fold.js";
import { childLimit } from "../model/fold.js";
import type { GraphIndex } from "../model/index.js";
import { move } from "../model/navigation.js";
import type { NavKey } from "../model/navigation.js";

const columns = "minmax(180px, 2fr) 84px 56px 72px minmax(120px, 1fr)";
const rowHeight = 28;
const indentStep = 14;

const navKeys: Readonly<Record<string, NavKey>> = {
    ArrowUp: "up",
    ArrowDown: "down",
    ArrowLeft: "left",
    ArrowRight: "right",
    Home: "home",
    End: "end",
};

export interface OutlineViewProps {
    siteId: string;
    index: GraphIndex;
    rows: readonly VisibleRow[];
    selectedId: string | null;
    matched: ReadonlySet<string> | null;
    onSelect: (id: string | null) => void;
    onPick: (id: string) => void;
    onToggle: (id: string) => void;
    onLiftMore: (parentId: string, count: number) => void;
}

export function OutlineView({ siteId, index, rows, selectedId, matched, onSelect, onPick, onToggle, onLiftMore }: OutlineViewProps): ReactElement {
    const onKeyDown = (event: KeyboardEvent<HTMLDivElement>): void => {
        const nav = navKeys[event.key];
        if (nav !== undefined) {
            event.preventDefault();
            const next = move(rows, selectedId, nav);
            if (next === null) {
                return;
            }
            if (next.expand) {
                onToggle(next.id);
            } else {
                onSelect(next.id);
            }
        } else if (event.key === " " && selectedId !== null) {
            event.preventDefault();
            onToggle(selectedId);
        } else if (event.key === "Escape") {
            onSelect(null);
        }
    };

    const row = (position: number): ReactElement => {
        const held = rows[position];
        if (held.kind === "more") {
            return (
                <TableRow key={held.id} className="text-ink-dim">
                    <TableCell className="col-span-5">
                        <span className="flex items-center gap-2" style={{ paddingLeft: held.depth * indentStep }}>
                            <Button
                                size="sm"
                                variant="ghost"
                                onClick={() => {
                                    onLiftMore(held.parentId, childLimit);
                                }}
                            >
                                {copy.graph.node.showMore(Math.min(childLimit, held.hidden))}
                            </Button>
                            <span className="font-mono text-2xs text-ink-faint">{copy.graph.node.more(held.shown, held.hidden)}</span>
                        </span>
                    </TableCell>
                </TableRow>
            );
        }
        const entity = index.byId.get(held.id);
        const flags = index.problems.get(held.id);
        const Icon = entityIcon(entity?.kind ?? "");
        const dim = matched !== null && !matched.has(held.id);
        return (
            <TableRow
                key={held.id}
                interactive={true}
                selected={held.id === selectedId}
                className={cx(dim && "opacity-40")}
                onClick={() => {
                    onPick(held.id);
                }}
            >
                <TableCell>
                    <span className="flex min-w-0 items-center gap-1" style={{ paddingLeft: held.depth * indentStep }}>
                        {held.childCount > 0 ? (
                            <button
                                type="button"
                                aria-expanded={held.expanded}
                                aria-label={held.expanded ? copy.graph.outline.fold : copy.graph.outline.unfold}
                                className="flex h-5 w-5 shrink-0 items-center justify-center rounded-sm text-ink-dim hover:bg-raised hover:text-ink"
                                onClick={(event) => {
                                    event.stopPropagation();
                                    onToggle(held.id);
                                }}
                            >
                                <ChevronRightIcon size={14} className={cx("transition-transform duration-100", held.expanded && "rotate-90")} />
                            </button>
                        ) : (
                            <span className="w-5 shrink-0" aria-hidden={true} />
                        )}
                        <Icon size={14} className={cx("shrink-0", toneClasses[kindTone(entity?.kind ?? "")].ink)} />
                        <span className="truncate">{entity?.name ?? held.id}</span>
                        {held.childCount > 0 && !held.expanded ? (
                            <span className="shrink-0 font-mono text-2xs text-ink-faint">{copy.graph.outline.children(held.hiddenChildren)}</span>
                        ) : null}
                    </span>
                </TableCell>
                <TableCell muted={true}>{entity?.kind ?? ""}</TableCell>
                <TableCell mono={true} align="right">
                    {entity === undefined ? "" : formatScore(entity.score)}
                </TableCell>
                <TableCell>
                    {flags?.noPage ? (
                        <StatusBadge tone="danger" icon={LinkOffIcon} dot={false}>
                            {copy.graph.outline.noPage}
                        </StatusBadge>
                    ) : (
                        <span className="text-2xs text-ink-faint">{copy.graph.outline.page}</span>
                    )}
                </TableCell>
                <TableCell>
                    <span className="flex items-center gap-1">
                        {(flags?.proposed ?? 0) > 0 ? <CountBadge tone="info" count={flags?.proposed ?? 0} /> : null}
                        {flags?.orphan ? (
                            <StatusBadge tone="warn" dot={false}>
                                {copy.graph.lens.orphan}
                            </StatusBadge>
                        ) : null}
                        {flags?.multiParent ? <CallSplitIcon size={12} className="text-ink-dim" /> : null}
                    </span>
                </TableCell>
            </TableRow>
        );
    };

    return (
        <div className="flex min-h-0 flex-1 flex-col outline-none" tabIndex={0} onKeyDown={onKeyDown}>
            <DenseTable columns={columns} label={copy.graph.views.outline} className="flex min-h-0 flex-1 flex-col">
                <TableHead>
                    <TableCell>{copy.graph.outline.columns.name}</TableCell>
                    <TableCell>{copy.graph.outline.columns.kind}</TableCell>
                    <TableCell align="right">{copy.graph.outline.columns.score}</TableCell>
                    <TableCell>{copy.graph.outline.columns.page}</TableCell>
                    <TableCell>{copy.graph.outline.columns.attention}</TableCell>
                </TableHead>
                <VirtualRows count={rows.length} rowHeight={rowHeight} row={row} scrollKey={`${siteId}:graph-outline`} />
            </DenseTable>
        </div>
    );
}
