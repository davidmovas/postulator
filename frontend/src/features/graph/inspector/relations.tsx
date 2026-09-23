import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { useApproveEdge, useRejectEdge } from "../../../data/hooks/graph.js";
import type { Entity } from "../../../data/types.js";
import { Button, CheckIcon, CloseIcon, IconButton, toneClasses } from "../../../ui/index.js";
import { entityIcon, formatScore, kindTone } from "../labels.js";
import type { GraphIndex } from "../model/index.js";

const childrenShown = 10;

interface RowProps {
    id: string;
    index: GraphIndex;
    note?: string;
    tone?: "info" | "muted";
    edgeId?: string;
    proposed?: boolean;
    onPick: (id: string) => void;
}

function Row({ id, index, note, tone, edgeId, proposed, onPick }: RowProps): ReactElement {
    const held = index.byId.get(id);
    const Icon = entityIcon(held?.kind ?? "");
    const approve = useApproveEdge();
    const reject = useRejectEdge();
    return (
        <li className="flex items-center gap-1">
            <button
                type="button"
                className="flex min-w-0 flex-1 items-center gap-2 rounded-md px-1 py-0.5 text-left text-xs text-ink-soft hover:bg-inset hover:text-ink"
                onClick={() => {
                    onPick(id);
                }}
            >
                <Icon size={14} className={toneClasses[kindTone(held?.kind ?? "")].ink} />
                <span className="flex-1 truncate">{held?.name ?? id}</span>
                {note === undefined ? null : <span className={`font-mono text-2xs ${tone === "info" ? "text-info" : "text-ink-faint"}`}>{note}</span>}
            </button>
            {proposed && edgeId !== undefined ? (
                <span className="flex shrink-0 items-center">
                    <IconButton
                        icon={CheckIcon}
                        label={copy.graph.edges.approve}
                        variant="ghost"
                        size="sm"
                        className="text-ok"
                        disabled={approve.isPending || reject.isPending}
                        onClick={() => {
                            approve.mutate({ id: edgeId });
                        }}
                    />
                    <IconButton
                        icon={CloseIcon}
                        label={copy.graph.edges.reject}
                        variant="ghost"
                        size="sm"
                        className="text-danger"
                        disabled={approve.isPending || reject.isPending}
                        onClick={() => {
                            reject.mutate({ id: edgeId });
                        }}
                    />
                </span>
            ) : null}
        </li>
    );
}

export interface RelationsProps {
    entity: Entity;
    index: GraphIndex;
    onReveal: (id: string) => void;
    onConnect: () => void;
}

export function Relations({ entity, index, onReveal, onConnect }: RelationsProps): ReactElement {
    const parents = index.approvedParents.get(entity.id) ?? [];
    const proposedParents = index.proposedParents.get(entity.id) ?? [];
    const children = index.children.get(entity.id) ?? [];
    const related = index.related.get(entity.id) ?? [];
    const proposedParentEdge = (parentId: string): string | undefined =>
        index.edges.find((edge) => edge.kind === "parent" && edge.status === "proposed" && edge.fromEntityId === entity.id && edge.toEntityId === parentId)?.id;

    return (
        <div className="flex flex-col gap-2">
            {parents.length + proposedParents.length + children.length + related.length === 0 ? (
                <p className="text-xs text-ink-dim">{copy.graph.inspector.noRelations}</p>
            ) : null}
            {parents.length + proposedParents.length > 0 ? (
                <div>
                    <p className="text-2xs text-ink-faint">{copy.graph.inspector.parents}</p>
                    <ul>
                        {parents.map((id) => (
                            <Row key={id} id={id} index={index} onPick={onReveal} />
                        ))}
                        {proposedParents.map((id) => (
                            <Row
                                key={id}
                                id={id}
                                index={index}
                                note={copy.graph.inspector.proposedParent}
                                tone="info"
                                edgeId={proposedParentEdge(id)}
                                proposed={true}
                                onPick={onReveal}
                            />
                        ))}
                    </ul>
                </div>
            ) : null}
            {children.length > 0 ? (
                <div>
                    <p className="text-2xs text-ink-faint">
                        {copy.graph.inspector.children} · {children.length}
                    </p>
                    <ul>
                        {children.slice(0, childrenShown).map((id) => (
                            <Row key={id} id={id} index={index} note={formatScore(index.byId.get(id)?.score ?? 0)} onPick={onReveal} />
                        ))}
                    </ul>
                    {children.length > childrenShown ? (
                        <p className="px-1 text-2xs text-ink-faint">{copy.graph.inspector.moreChildren(children.length - childrenShown)}</p>
                    ) : null}
                </div>
            ) : null}
            {related.length > 0 ? (
                <div>
                    <p className="text-2xs text-ink-faint">{copy.graph.inspector.related}</p>
                    <ul>
                        {related.map((link) => (
                            <Row
                                key={link.edgeId}
                                id={link.otherId}
                                index={index}
                                note={link.weight.toFixed(2)}
                                tone={link.status === "proposed" ? "info" : "muted"}
                                edgeId={link.edgeId}
                                proposed={link.status === "proposed"}
                                onPick={onReveal}
                            />
                        ))}
                    </ul>
                </div>
            ) : null}
            <Button size="sm" variant="secondary" onClick={onConnect}>
                {copy.graph.connect.start}
            </Button>
        </div>
    );
}
