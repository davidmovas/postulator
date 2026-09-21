import type { ReactElement } from "react";

import { PlanPageDialog } from "../pages/create.js";
import type { EntityIndex } from "../pages/entities.js";
import { ConnectDrawer } from "./actions/connect.js";
import { CreateEntityDrawer } from "./actions/create-entity.js";
import { DeleteEntityDialog } from "./actions/delete-entity.js";
import { MoveEntityDialog } from "./actions/move-entity.js";
import type { MoveRequest } from "./actions/move-entity.js";
import { ProposeFromPagesDialog, ProposeRelatedDialog } from "./actions/propose.js";
import type { GraphIndex } from "./model/index.js";

export type Proposing = "pages" | "related" | null;

export interface Creating {
    parentId: string | null;
}

export interface GraphOverlaysProps {
    siteId: string;
    index: GraphIndex;
    entityIndex: EntityIndex;
    selectedId: string | null;
    creating: Creating | null;
    connectFrom: string | null;
    connectTo: string | null;
    deleting: string | null;
    moving: MoveRequest | null;
    proposing: Proposing;
    planning: string | null;
    onCreatingChange: (next: Creating | null) => void;
    onStopConnect: () => void;
    onDeletingChange: (next: string | null) => void;
    onMovingChange: (next: MoveRequest | null) => void;
    onProposingChange: (next: Proposing) => void;
    onPlanningChange: (next: string | null) => void;
    onSelect: (id: string | null) => void;
    onReviewQueue: () => void;
}

export function GraphOverlays({
    siteId,
    index,
    entityIndex,
    selectedId,
    creating,
    connectFrom,
    connectTo,
    deleting,
    moving,
    proposing,
    planning,
    onCreatingChange,
    onStopConnect,
    onDeletingChange,
    onMovingChange,
    onProposingChange,
    onPlanningChange,
    onSelect,
    onReviewQueue,
}: GraphOverlaysProps): ReactElement {
    return (
        <>
            <CreateEntityDrawer
                open={creating !== null}
                onOpenChange={(open) => {
                    if (!open) {
                        onCreatingChange(null);
                    }
                }}
                siteId={siteId}
                index={index}
                parentId={creating?.parentId ?? null}
                onCreated={onSelect}
            />
            <ConnectDrawer siteId={siteId} index={index} fromId={connectFrom} toId={connectTo} onClose={onStopConnect} />
            <DeleteEntityDialog
                index={index}
                entityId={deleting}
                onClose={() => {
                    onDeletingChange(null);
                }}
                onDeleted={() => {
                    if (deleting === selectedId) {
                        onSelect(null);
                    }
                }}
            />
            <MoveEntityDialog
                siteId={siteId}
                index={index}
                move={moving}
                onClose={() => {
                    onMovingChange(null);
                }}
            />
            <ProposeFromPagesDialog
                open={proposing === "pages"}
                onOpenChange={(open) => {
                    if (!open) {
                        onProposingChange(null);
                    }
                }}
                siteId={siteId}
                onReview={onReviewQueue}
            />
            <ProposeRelatedDialog
                open={proposing === "related"}
                onOpenChange={(open) => {
                    if (!open) {
                        onProposingChange(null);
                    }
                }}
                siteId={siteId}
                index={index}
                selectedId={selectedId}
                onReview={onReviewQueue}
            />
            <PlanPageDialog
                open={planning !== null}
                onOpenChange={(open) => {
                    if (!open) {
                        onPlanningChange(null);
                    }
                }}
                siteId={siteId}
                index={entityIndex}
                search=""
                initialEntityId={planning ?? undefined}
            />
        </>
    );
}
