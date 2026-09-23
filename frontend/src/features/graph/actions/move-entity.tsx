import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../../copy/index.js";
import { react } from "../../../data/errors.js";
import { useMoveEntity } from "../../../data/hooks/graph.js";
import { pushToast } from "../../../data/toasts.js";
import { AltRouteIcon, Checkbox, Dialog } from "../../../ui/index.js";
import { cycleOf } from "../model/cycle.js";
import type { GraphIndex } from "../model/index.js";

export interface MoveRequest {
    childId: string;
    parentId: string;
}

export function parentBeingReplaced(index: GraphIndex, childId: string): string | undefined {
    return index.placementProposed.has(childId) ? undefined : index.placementParent.get(childId);
}

export interface MoveEntityDialogProps {
    index: GraphIndex;
    move: MoveRequest | null;
    onClose: () => void;
}

export function MoveEntityDialog({ index, move, onClose }: MoveEntityDialogProps): ReactElement {
    const moveEntity = useMoveEntity();
    const [keepBoth, setKeepBoth] = useState(false);
    const child = move === null ? undefined : index.byId.get(move.childId);
    const parent = move === null ? undefined : index.byId.get(move.parentId);
    const previousId = child === undefined ? undefined : parentBeingReplaced(index, child.id);
    const previous = previousId === undefined ? undefined : index.byId.get(previousId);

    useEffect(() => {
        if (move !== null) {
            setKeepBoth(false);
            moveEntity.reset();
        }
    }, [move?.childId, move?.parentId]);

    const run = async (): Promise<void> => {
        if (child === undefined || parent === undefined) {
            return;
        }
        try {
            await moveEntity.mutateAsync({ entityId: child.id, newParentId: parent.id, keepBoth });
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
            busy={moveEntity.isPending}
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
