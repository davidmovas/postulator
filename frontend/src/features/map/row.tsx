import type { ReactElement } from "react";
import { Fragment } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import type { SiteMapNode } from "../../data/types.js";
import {
    CenterFocusWeakIcon,
    Checkbox,
    ChevronRightIcon,
    cx,
    DescriptionIcon,
    IconButton,
    shiftHeld,
    StatusBadge,
    TableCell,
    TableRow,
} from "../../ui/index.js";
import type { TickState } from "../../ui/selection.js";
import { findingsAddress } from "../findings/index.js";
import {
    parentMethodName,
    parentReasonLabel,
    parentReasonSentence,
    readOnlyReasonLabel,
    technicalReasonLabel,
} from "../pages/model/labels.js";
import { bodyStateLabel, bodyStateTone, bodyStateTooltip } from "./labels.js";
import { useMapColumns } from "./layout.js";
import type { MapColumn } from "./model/columns.js";

const said = copy.map;

export const mapRowHeight = 28;
export const indentStep = 14;

function ParentCell({ node }: { node: SiteMapNode }): ReactElement {
    const reason = parentReasonLabel(node.parentReason);
    const shown = node.parentMethod === "" ? reason : `${reason} · ${parentMethodName(node.parentMethod)}`;
    return (
        <TableCell muted={true} title={parentReasonSentence(node.parentReason, node.parentMethod)}>
            {shown}
        </TableCell>
    );
}

function Marks({ node }: { node: SiteMapNode }): ReactElement | null {
    if (node.technical === "" && node.readOnly === "") {
        return null;
    }
    return (
        <>
            {node.technical === "" ? null : (
                <span title={technicalReasonLabel(node.technical)} className="shrink-0">
                    <StatusBadge tone="muted" dot={false}>
                        {said.technical}
                    </StatusBadge>
                </span>
            )}
            {node.readOnly === "" ? null : (
                <span title={readOnlyReasonLabel(node.readOnly)} className="shrink-0">
                    <StatusBadge tone="muted" dot={false}>
                        {said.readOnly}
                    </StatusBadge>
                </span>
            )}
        </>
    );
}

export interface NodeTick {
    label: string;
    state: TickState;
    onChange: (ranged: boolean) => void;
}

export interface NodeRowProps {
    siteId: string;
    node: SiteMapNode;
    depth: number;
    expanded: boolean;
    tick?: NodeTick;
    onToggle: (pageId: string) => void;
    onOpen: (pageId: string) => void;
    onUnder: (pageId: string) => void;
}

type PageCellProps = Omit<NodeRowProps, "siteId" | "onUnder">;

function PageCell({ node, depth, expanded, tick, onToggle, onOpen }: PageCellProps): ReactElement {
    const { page, twin } = node;
    const trail = (node.trail ?? []).map((held) => held.title);
    return (
        <TableCell>
            <span className="flex min-w-0 items-center gap-1.5" style={{ paddingLeft: `${depth * indentStep}px` }}>
                {tick === undefined ? null : (
                    <span
                        className="flex shrink-0 items-center"
                        onMouseDown={(event) => {
                            if (event.shiftKey) {
                                event.preventDefault();
                            }
                        }}
                    >
                        <Checkbox
                            aria-label={tick.label}
                            checked={tick.state === "on"}
                            indeterminate={tick.state === "some"}
                            onChange={(event) => {
                                tick.onChange(shiftHeld(event));
                            }}
                        />
                    </span>
                )}
                {node.children === 0 ? (
                    <DescriptionIcon size={13} className="shrink-0 text-ink-faint" />
                ) : (
                    <button
                        type="button"
                        aria-expanded={expanded}
                        aria-label={expanded ? said.collapse(page.path) : said.expand(page.path)}
                        className="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-sm text-ink-dim hover:bg-raised hover:text-ink"
                        onClick={() => {
                            onToggle(page.pageId);
                        }}
                    >
                        <ChevronRightIcon size={14} className={cx("transition-transform duration-100", expanded && "rotate-90")} />
                    </button>
                )}
                <button
                    type="button"
                    title={said.open}
                    onClick={() => {
                        onOpen(page.pageId);
                    }}
                    className="flex min-w-0 items-center gap-1.5 rounded-sm hover:text-accent"
                >
                    <span className="truncate">{page.title === "" ? said.untitled : page.title}</span>
                    <span className="truncate font-mono text-xs text-ink-faint">{page.path}</span>
                </button>
                {twin === undefined || twin === null ? null : (
                    <button
                        type="button"
                        title={said.twinTooltip}
                        data-map-twin={twin.pageId}
                        onClick={() => {
                            onOpen(twin.pageId);
                        }}
                        className="flex h-5 min-w-0 shrink items-center rounded-sm bg-inset px-1.5 font-mono text-2xs text-ink-dim hover:text-ink"
                    >
                        <span className="truncate">{said.twin(twin.path)}</span>
                    </button>
                )}
                {trail.length === 0 ? null : <span className="truncate text-2xs text-ink-faint">{said.trail(trail)}</span>}
                <Marks node={node} />
            </span>
        </TableCell>
    );
}

export function NodeRow(props: NodeRowProps): ReactElement {
    const { siteId, node, onUnder } = props;
    const { page } = node;
    const columns = useMapColumns();

    const cellOf = (column: MapColumn): ReactElement => {
        switch (column) {
            case "page":
                return <PageCell {...props} />;
            case "type":
                return <TableCell muted={true}>{page.type === "" ? said.untyped : page.type}</TableCell>;
            case "parent":
                return <ParentCell node={node} />;
            case "body":
                return (
                    <TableCell>
                        <span title={bodyStateTooltip(node.body)}>
                            <StatusBadge tone={bodyStateTone(node.body)} dot={false}>
                                {bodyStateLabel(node.body)}
                            </StatusBadge>
                        </span>
                    </TableCell>
                );
            case "links":
                return (
                    <TableCell mono={true} muted={true} title={said.linksTooltip}>
                        {said.links(node.linksIn, node.linksOut)}
                    </TableCell>
                );
            case "findings":
                return (
                    <TableCell mono={true}>
                        {node.findings === 0 ? null : (
                            <Link to={findingsAddress(siteId, page.pageId)} title={said.findingsTooltip} className="text-warn hover:underline">
                                {node.findings}
                            </Link>
                        )}
                    </TableCell>
                );
            case "children":
                return (
                    <TableCell
                        mono={true}
                        muted={true}
                        title={node.children === 0 ? said.noChildren : said.childrenTooltip(node.children, node.descendants)}
                    >
                        {node.children === 0 ? "" : node.children}
                    </TableCell>
                );
            case "under":
                return (
                    <TableCell>
                        {node.children === 0 ? null : (
                            <IconButton
                                size="sm"
                                variant="ghost"
                                icon={CenterFocusWeakIcon}
                                label={said.under}
                                onClick={() => {
                                    onUnder(page.pageId);
                                }}
                            />
                        )}
                    </TableCell>
                );
        }
    };

    return (
        <TableRow data-map-node={page.pageId} data-map-tick={props.tick?.state}>
            {columns.map((column) => (
                <Fragment key={column}>{cellOf(column)}</Fragment>
            ))}
        </TableRow>
    );
}
