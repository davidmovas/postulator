import type { KeyboardEvent, ReactElement } from "react";
import { useEffect, useMemo, useRef, useState } from "react";

import { copy } from "../../../copy/index.js";
import { useApproveEdge, useRejectEdge } from "../../../data/hooks/graph.js";
import type { Edge } from "../../../data/types.js";
import {
    ArrowRightAltIcon,
    CheckIcon,
    CloseIcon,
    cx,
    DenseTable,
    EmptyState,
    IconButton,
    PolylineIcon,
    Sheet,
    SyncAltIcon,
    TableCell,
    TableHead,
    TableRow,
    toneClasses,
    VirtualRows,
} from "../../../ui/index.js";
import { entityIcon, kindTone } from "../labels.js";
import type { GraphIndex } from "../model/index.js";
import { BulkDialog } from "./bulk.js";
import type { BulkRequest } from "./bulk.js";
import { Threshold } from "./thresholds.js";

const columns = "minmax(140px, 1.4fr) 76px minmax(140px, 1.4fr) 112px minmax(160px, 2fr) 64px";
const rowHeight = 28;
const minSheetHeight = 160;

function confidenceTone(weight: number): "ok" | "warn" | "danger" {
    if (weight >= 0.7) {
        return "ok";
    }
    return weight >= 0.4 ? "warn" : "danger";
}

interface NameProps {
    id: string;
    index: GraphIndex;
    onReveal: (id: string) => void;
}

function Name({ id, index, onReveal }: NameProps): ReactElement {
    const held = index.byId.get(id);
    const Icon = entityIcon(held?.kind ?? "");
    return (
        <button
            type="button"
            className="flex min-w-0 items-center gap-1.5 text-left text-xs text-ink-soft hover:text-ink"
            onClick={(event) => {
                event.stopPropagation();
                onReveal(id);
            }}
        >
            <Icon size={14} className={cx("shrink-0", toneClasses[kindTone(held?.kind ?? "")].ink)} />
            <span className="truncate">{held?.name ?? id}</span>
        </button>
    );
}

export interface ProposalQueueProps {
    index: GraphIndex;
    open: boolean;
    height: number;
    onClose: () => void;
    onHeightChange: (height: number) => void;
    onHover: (edgeId: string | null) => void;
    onReveal: (entityId: string) => void;
}

export function ProposalQueue({ index, open, height, onClose, onHeightChange, onHover, onReveal }: ProposalQueueProps): ReactElement | null {
    const approve = useApproveEdge();
    const reject = useRejectEdge();
    const edges = index.proposedEdges;
    const [active, setActive] = useState(0);
    const [approveAt, setApproveAt] = useState("0.8");
    const [rejectBelow, setRejectBelow] = useState("0.3");
    const [bulk, setBulk] = useState<BulkRequest | null>(null);
    const body = useRef<HTMLDivElement>(null);

    useEffect(() => {
        setActive((held) => Math.min(held, Math.max(edges.length - 1, 0)));
    }, [edges.length]);

    useEffect(() => {
        if (open) {
            body.current?.focus();
        }
    }, [open]);

    const approveThreshold = Number.parseFloat(approveAt);
    const rejectThreshold = Number.parseFloat(rejectBelow);
    const aboveCount = useMemo(
        () => (Number.isFinite(approveThreshold) ? edges.filter((edge) => edge.weight >= approveThreshold).length : 0),
        [edges, approveThreshold],
    );
    const belowCount = useMemo(
        () => (Number.isFinite(rejectThreshold) ? edges.filter((edge) => edge.weight < rejectThreshold).length : 0),
        [edges, rejectThreshold],
    );

    const decide = (edge: Edge, decision: "approve" | "reject"): void => {
        if (decision === "approve") {
            approve.mutate({ id: edge.id });
        } else {
            reject.mutate({ id: edge.id });
        }
    };

    const onKeyDown = (event: KeyboardEvent<HTMLElement>): void => {
        const current = edges[active];
        if (event.key === "ArrowDown") {
            event.preventDefault();
            setActive((held) => Math.min(held + 1, Math.max(edges.length - 1, 0)));
        } else if (event.key === "ArrowUp") {
            event.preventDefault();
            setActive((held) => Math.max(held - 1, 0));
        } else if ((event.key === "a" || event.key === "A") && current !== undefined) {
            decide(current, "approve");
        } else if ((event.key === "r" || event.key === "R") && current !== undefined) {
            decide(current, "reject");
        } else if (event.key === "Enter" && current !== undefined) {
            onReveal(current.fromEntityId);
        }
    };

    if (!open) {
        return null;
    }

    const row = (position: number): ReactElement => {
        const edge = edges[position];
        const tone = confidenceTone(edge.weight);
        return (
            <TableRow
                key={edge.id}
                interactive={true}
                selected={position === active}
                onMouseEnter={() => {
                    onHover(edge.id);
                }}
                onMouseLeave={() => {
                    onHover(null);
                }}
                onClick={() => {
                    setActive(position);
                }}
            >
                <TableCell>
                    <Name id={edge.fromEntityId} index={index} onReveal={onReveal} />
                </TableCell>
                <TableCell>
                    <span className="flex items-center gap-1 text-2xs text-ink-dim">
                        {edge.kind === "parent" ? <ArrowRightAltIcon size={14} /> : <SyncAltIcon size={14} />}
                        {edge.kind === "parent" ? copy.graph.queue.parent : copy.graph.queue.related}
                    </span>
                </TableCell>
                <TableCell>
                    <Name id={edge.toEntityId} index={index} onReveal={onReveal} />
                </TableCell>
                <TableCell>
                    <span className="flex items-center gap-2">
                        <span className="h-1 w-12 overflow-hidden rounded-sm bg-raised" aria-hidden={true}>
                            <span className={cx("block h-full rounded-sm", toneClasses[tone].solid)} style={{ width: `${Math.round(edge.weight * 100)}%` }} />
                        </span>
                        <span className={cx("font-mono text-2xs", toneClasses[tone].ink)}>{edge.weight.toFixed(2)}</span>
                    </span>
                </TableCell>
                <TableCell muted={edge.reason === ""} title={edge.reason}>
                    {edge.reason === "" ? copy.graph.queue.noReason : edge.reason}
                </TableCell>
                <TableCell>
                    <span className="flex items-center">
                        <IconButton
                            icon={CheckIcon}
                            label={copy.graph.edges.approve}
                            variant="ghost"
                            size="sm"
                            className="text-ok"
                            onClick={(event) => {
                                event.stopPropagation();
                                decide(edge, "approve");
                            }}
                        />
                        <IconButton
                            icon={CloseIcon}
                            label={copy.graph.edges.reject}
                            variant="ghost"
                            size="sm"
                            className="text-danger"
                            onClick={(event) => {
                                event.stopPropagation();
                                decide(edge, "reject");
                            }}
                        />
                    </span>
                </TableCell>
            </TableRow>
        );
    };

    return (
        <>
            <Sheet
                open={open}
                title={copy.graph.queue.title(edges.length)}
                closeLabel={copy.graph.queue.close}
                onClose={onClose}
                height={height}
                minHeight={minSheetHeight}
                resizeLabel={copy.graph.queue.resize}
                onHeightChange={onHeightChange}
                onKeyDown={onKeyDown}
                header={
                    <>
                        <span className="hidden font-mono text-2xs text-ink-faint xl:inline">{copy.graph.queue.keys}</span>
                        <span className="flex-1" />
                        <Threshold
                            label={copy.graph.queue.approveFrom}
                            action={copy.graph.queue.approveAbove(aboveCount)}
                            value={approveAt}
                            matched={aboveCount}
                            onChange={setApproveAt}
                            onApply={() => {
                                setBulk({ decision: "approve", threshold: approveThreshold, edges: edges.filter((edge) => edge.weight >= approveThreshold) });
                            }}
                        />
                        <Threshold
                            label={copy.graph.queue.rejectUnder}
                            action={copy.graph.queue.rejectBelow(belowCount)}
                            value={rejectBelow}
                            matched={belowCount}
                            onChange={setRejectBelow}
                            onApply={() => {
                                setBulk({ decision: "reject", threshold: rejectThreshold, edges: edges.filter((edge) => edge.weight < rejectThreshold) });
                            }}
                        />
                    </>
                }
            >
                <div ref={body} tabIndex={0} className="flex min-h-0 flex-1 flex-col outline-none" onKeyDown={onKeyDown}>
                    {edges.length === 0 ? (
                        <div className="p-4">
                            <EmptyState icon={PolylineIcon} title={copy.graph.queue.emptyTitle} body={copy.empty.proposedEdges} />
                        </div>
                    ) : (
                        <DenseTable columns={columns} label={copy.graph.queue.title(edges.length)} className="flex min-h-0 flex-1 flex-col">
                            <TableHead>
                                <TableCell>{copy.graph.queue.columns.from}</TableCell>
                                <TableCell>{copy.graph.queue.columns.kind}</TableCell>
                                <TableCell>{copy.graph.queue.columns.to}</TableCell>
                                <TableCell>{copy.graph.queue.columns.confidence}</TableCell>
                                <TableCell>{copy.graph.queue.columns.reason}</TableCell>
                                <TableCell>{copy.graph.queue.columns.decision}</TableCell>
                            </TableHead>
                            <VirtualRows count={edges.length} rowHeight={rowHeight} row={row} />
                        </DenseTable>
                    )}
                </div>
            </Sheet>
            <BulkDialog
                request={bulk}
                onClose={() => {
                    setBulk(null);
                }}
            />
        </>
    );
}
