import type { ReactElement } from "react";
import { useRef, useState } from "react";

import { EntityContextMenu } from "./actions/context-menu.js";
import type { MenuTarget } from "./actions/context-menu.js";
import { Controls } from "./canvas/controls.js";
import { Legend } from "./canvas/legend.js";
import { GraphMap } from "./canvas/map.js";
import type { GraphMapProps, MapHandle } from "./canvas/map.js";
import { collapseToDepth, expandAll, liftMore } from "./model/fold.js";
import type { FoldState } from "./model/fold.js";
import { NodeCard } from "./node-card.js";
import type { Hovered } from "./node-card.js";
import type { GraphSession } from "./state.js";

interface Card extends Hovered {
    width: number;
}

type Passed =
    | "siteId"
    | "index"
    | "rows"
    | "fold"
    | "selectedId"
    | "matched"
    | "highlightEdgeId"
    | "pulse"
    | "tint"
    | "revealVersion"
    | "onSelect"
    | "onPick"
    | "onToggle"
    | "onCreateChild"
    | "onConnect"
    | "onDelete"
    | "onReparent";

export interface GraphMapViewProps extends Pick<GraphMapProps, Passed> {
    session: GraphSession;
    proof: boolean;
    hostWidth: () => number;
    onPatchSession: (patch: Partial<GraphSession>) => void;
    onFold: (next: FoldState) => void;
    onReveal: (id: string) => void;
}

export function GraphMapView(props: GraphMapViewProps): ReactElement {
    const { siteId, index, rows, fold, session, proof, hostWidth, onPatchSession, onFold, onReveal } = props;
    const { onCreateChild, onConnect, onDelete, onToggle } = props;
    const [menu, setMenu] = useState<MenuTarget | null>(null);
    const [card, setCard] = useState<Card | null>(null);
    const map = useRef<MapHandle | null>(null);

    return (
        <>
            <GraphMap
                siteId={siteId}
                index={index}
                rows={rows}
                fold={fold}
                selectedId={props.selectedId}
                matched={props.matched}
                showRelated={session.showRelated}
                highlightEdgeId={props.highlightEdgeId}
                pulse={props.pulse}
                tint={props.tint}
                revealVersion={props.revealVersion}
                onSelect={props.onSelect}
                onPick={props.onPick}
                onToggle={onToggle}
                onLiftMore={(parentId, count) => {
                    onFold(liftMore(fold, parentId, count));
                }}
                onContextMenu={(id, at) => {
                    setCard(null);
                    if (id === null) {
                        setMenu(null);
                        return;
                    }
                    const row = rows.find((held) => held.id === id);
                    setMenu({
                        id,
                        at,
                        expanded: row?.kind === "entity" && row.expanded,
                        hasChildren: row?.kind === "entity" && row.childCount > 0,
                    });
                }}
                onHover={(id, at) => {
                    setCard(id === null ? null : { id, at, width: hostWidth() });
                }}
                onCreateChild={onCreateChild}
                onConnect={onConnect}
                onDelete={onDelete}
                onReparent={props.onReparent}
                ref={(handle) => {
                    map.current = handle;
                }}
                minimap={session.minimap}
            />
            <Controls
                siteId={siteId}
                showRelated={session.showRelated}
                onZoom={(factor) => map.current?.zoomBy(factor)}
                onFit={() => map.current?.fit()}
                onExpandAll={() => {
                    onFold(expandAll(index));
                }}
                onCollapse={() => {
                    onFold(collapseToDepth(index, 1));
                }}
                onToggleRelated={() => {
                    onPatchSession({ showRelated: !session.showRelated });
                }}
            />
            <Legend
                open={session.legend}
                proof={proof}
                onToggle={() => {
                    onPatchSession({ legend: !session.legend });
                }}
            />
            {card === null || menu !== null ? null : <NodeCard index={index} hovered={card} hostWidth={card.width} />}
            <EntityContextMenu
                target={menu}
                onClose={() => {
                    setMenu(null);
                }}
                onAddChild={(id) => {
                    onCreateChild(id);
                }}
                onConnect={onConnect}
                onToggle={onToggle}
                onCenter={onReveal}
                onDelete={onDelete}
            />
        </>
    );
}
