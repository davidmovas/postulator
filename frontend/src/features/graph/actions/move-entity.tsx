import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../../copy/index.js";
import { react } from "../../../data/errors.js";
import { useAddEdge, useDeleteEdge } from "../../../data/hooks/graph.js";
import { pushToast } from "../../../data/toasts.js";
import { AltRouteIcon, Checkbox, Dialog } from "../../../ui/index.js";
import { cycleOf } from "../model/cycle.js";
import type { GraphIndex } from "../model/index.js";

export interface MoveRequest {
    childId: string;
    parentId: string;
}

export interface MoveEntityDialogProps {
    siteId: string;
    index: GraphIndex;
    move: MoveRequest | null;
    onClose: () => void;
}

export function MoveEntityDialog({ siteId, index, move, onClose }: MoveEntityDialogProps): ReactElement {
    const addEdge = useAddEdge();
    const deleteEdge = useDeleteEdge();
    const [keepBoth, setKeepBoth] = useState(false);
    const child = move === null ? undefined : index.byId.get(move.childId);
    const parent = move === null ? undefined : index.byId.get(move.parentId);
    const previousId = child === undefined || index.placementProposed.has(child.id) ? undefined : index.placementParent.get(child.id);
    const previous = previousId === undefined ? undefined : index.byId.get(previousId);
    const previousEdge =
        child === undefined || previousId === undefined
            ? undefined
            : index.edges.find((edge) => edge.kind === "parent" && edge.status === "approved" && edge.fromEntityId === child.id && edge.toEntityId === previousId);

    useEffect(() => {
        if (move !== null) {
            setKeepBoth(false);
            addEdge.reset();
            deleteEdge.reset();
        }
    }, [move?.childId, move?.parentId]);

    const run = async (): Promise<void> => {
        if (child === undefined || parent === undefined) {
            return;
        }
        try {
            await addEdge.mutateAsync({ siteId, fromEntityId: child.id, toEntityId: parent.id, kind: "parent", weight: 1 });
            if (!keepBoth && previousEdge !== undefined) {
                await deleteEdge.mutateAsync({ id: previousEdge.id });
            }
            pushToast("info", copy.graph.move.moved(child.name, parent.name));
            onClose();
        } catch (thrown) {
            if (cycleOf(thrown) !== null) {
                pushToast("danger", copy.graph.connect.cycle);
                return;
            }
            const reaction = react(thrown);
            if (reaction.kind !== "silent" && reaction.kind !== "unlock") {
                pushToast("danger", reaction.message);
            }
        }
    };

    return (
        <Dialog
            open={child !== undefined && parent !== undefined}
            onOpenChange={(open) => {
                if (!open) {
                    onClose();
                }
            }}
            title={child === undefined || parent === undefined ? "" : copy.graph.move.title(child.name, parent.name)}
            description={previous === undefined ? copy.graph.move.fromRoot : copy.graph.move.fromParent(previous.name)}
            confirmLabel={copy.graph.move.confirm}
            cancelLabel={copy.graph.move.cancel}
            icon={AltRouteIcon}
            busy={addEdge.isPending || deleteEdge.isPending}
            onConfirm={() => {
                void run();
            }}
        >
            {previous === undefined ? null : (
                <Checkbox
                    label={copy.graph.move.keepBoth}
                    checked={keepBoth}
                    onChange={(event) => {
                        setKeepBoth(event.target.checked);
                    }}
                />
            )}
        </Dialog>
    );
}
