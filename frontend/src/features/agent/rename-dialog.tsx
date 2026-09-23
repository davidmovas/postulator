import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { useRenameConversation } from "../../data/hooks/agent.js";
import type { Conversation } from "../../data/types.js";
import { Dialog, EditIcon, Field, Input } from "../../ui/index.js";

export interface RenameDialogProps {
    conversation: Conversation | null;
    onClose: () => void;
}

export function RenameDialog({ conversation, onClose }: RenameDialogProps): ReactElement {
    const rename = useRenameConversation();
    const [title, setTitle] = useState("");

    useEffect(() => {
        if (conversation !== null) {
            setTitle(conversation.title);
        }
    }, [conversation]);

    return (
        <Dialog
            open={conversation !== null}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={copy.agent.header.renameAction}
            description={copy.agent.header.renameBody}
            confirmLabel={copy.agent.header.renameSave}
            cancelLabel={copy.agent.screen.cancel}
            icon={EditIcon}
            busy={rename.isPending}
            onConfirm={() => {
                if (conversation === null || title.trim() === "") {
                    return;
                }
                rename.mutate(
                    { conversationId: conversation.id, title: title.trim() },
                    {
                        onSuccess: onClose,
                    },
                );
            }}
        >
            <Field label={copy.agent.header.renameTitle}>
                {(control) => (
                    <Input
                        id={control.id}
                        autoFocus={true}
                        value={title}
                        maxLength={120}
                        onChange={(event) => {
                            setTitle(event.target.value);
                        }}
                    />
                )}
            </Field>
        </Dialog>
    );
}
