import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { useDeleteEntity } from "../../../data/hooks/graph.js";
import { pushToast } from "../../../data/toasts.js";
import { DeleteIcon, Dialog } from "../../../ui/index.js";
import type { GraphIndex } from "../model/index.js";

export interface DeleteEntityDialogProps {
    index: GraphIndex;
    entityId: string | null;
    onClose: () => void;
    onDeleted: () => void;
}

export function DeleteEntityDialog({ index, entityId, onClose, onDeleted }: DeleteEntityDialogProps): ReactElement {
    const remove = useDeleteEntity();
    const entity = entityId === null ? undefined : index.byId.get(entityId);
    const edges = entity === undefined ? 0 : index.edges.filter((edge) => edge.fromEntityId === entity.id || edge.toEntityId === entity.id).length;

    return (
        <Dialog
            open={entity !== undefined}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={entity === undefined ? "" : copy.graph.remove.title(entity.name)}
            description={edges === 0 ? copy.graph.remove.alone : copy.graph.remove.withEdges(edges)}
            confirmLabel={copy.graph.remove.confirm}
            cancelLabel={copy.graph.remove.cancel}
            destructive={true}
            icon={DeleteIcon}
            busy={remove.isPending}
            onConfirm={() => {
                if (entity === undefined) {
                    return;
                }
                remove.mutate(
                    { id: entity.id },
                    {
                        onSuccess: () => {
                            pushToast("info", copy.graph.remove.deleted(entity.name));
                            onClose();
                            onDeleted();
                        },
                    },
                );
            }}
        />
    );
}
