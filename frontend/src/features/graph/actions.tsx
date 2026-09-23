import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { AddIcon, AddLinkIcon, Button, CalculateIcon, CloseIcon, CountBadge, Menu, PolylineIcon, Stars2Icon } from "../../ui/index.js";
import type { MenuEntry } from "../../ui/index.js";
import type { GraphIndex } from "./model/index.js";

export interface GraphActionsProps {
    index: GraphIndex;
    selectedId: string | null;
    connecting: boolean;
    reviewing: boolean;
    recomputing: boolean;
    onCreate: () => void;
    onConnect: () => void;
    onStopConnect: () => void;
    onReview: () => void;
    onProposeFromPages: () => void;
    onProposeRelated: () => void;
    onRecompute: () => void;
}

export function GraphActions({
    index,
    selectedId,
    connecting,
    reviewing,
    recomputing,
    onCreate,
    onConnect,
    onStopConnect,
    onReview,
    onProposeFromPages,
    onProposeRelated,
    onRecompute,
}: GraphActionsProps): ReactElement {
    const proposed = index.counts.proposedEdges;
    const selectedName = selectedId === null ? null : (index.byId.get(selectedId)?.name ?? null);
    const model: readonly MenuEntry[] = [
        { key: "pages", label: copy.graph.ai.fromPages, icon: Stars2Icon, onSelect: onProposeFromPages },
        {
            key: "related",
            label: selectedName === null ? copy.graph.ai.related : copy.graph.ai.relatedFor(selectedName),
            icon: PolylineIcon,
            onSelect: onProposeRelated,
        },
        { kind: "separator", key: "s1" },
        { key: "recompute", label: copy.graph.ai.recompute, icon: CalculateIcon, disabled: recomputing, onSelect: onRecompute },
    ];

    return (
        <>
            {proposed > 0 || reviewing ? (
                <Button variant={reviewing ? "primary" : "secondary"} icon={PolylineIcon} aria-pressed={reviewing} onClick={onReview}>
                    {copy.graph.queue.open}
                    <CountBadge
                        tone="info"
                        variant={reviewing ? "contrast" : "soft"}
                        count={proposed}
                        className="ml-1"
                    />
                </Button>
            ) : null}
            {connecting ? (
                <Button variant="secondary" icon={CloseIcon} onClick={onStopConnect}>
                    {copy.graph.connect.stop}
                </Button>
            ) : (
                <Button
                    variant="secondary"
                    icon={AddLinkIcon}
                    disabled={selectedId === null}
                    title={selectedId === null ? copy.graph.actions.connectNeedsSelection : copy.graph.actions.connectKey}
                    onClick={onConnect}
                >
                    {copy.graph.connect.start}
                </Button>
            )}
            <Menu
                label={copy.graph.ai.menuLabel}
                trigger={
                    <Button variant="secondary" icon={Stars2Icon} busy={recomputing}>
                        {copy.graph.ai.menu}
                    </Button>
                }
                items={model}
            />
            <Button variant="primary" icon={AddIcon} title={copy.graph.actions.createKey} onClick={onCreate}>
                {copy.graph.actions.create}
            </Button>
        </>
    );
}
