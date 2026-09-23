import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useCreateConversation } from "../../data/hooks/agent.js";
import type { Conversation } from "../../data/types.js";
import type { SelectOption } from "../../ui/index.js";
import { AddCommentIcon, Dialog, Field, Select } from "../../ui/index.js";

const noSite = "no-site";

export interface NewConversationDialogProps {
    open: boolean;
    sites: readonly SelectOption<string>[];
    onOpenChange: (open: boolean) => void;
    onCreated: (conversation: Conversation) => void;
}

export function NewConversationDialog({ open, sites, onOpenChange, onCreated }: NewConversationDialogProps): ReactElement {
    const create = useCreateConversation();
    const [siteId, setSiteId] = useState(noSite);
    const thrown = create.error;
    const reaction = thrown === null ? null : react(thrown);

    useEffect(() => {
        if (open) {
            setSiteId(sites[0]?.value ?? noSite);
            create.reset();
        }
    }, [open, sites]);

    return (
        <Dialog
            open={open}
            onOpenChange={onOpenChange}
            title={copy.agent.screen.newTitle}
            description={copy.agent.subtitle}
            confirmLabel={copy.agent.screen.newStart}
            cancelLabel={copy.agent.screen.cancel}
            icon={AddCommentIcon}
            busy={create.isPending}
            onConfirm={() => {
                create.mutate(
                    { siteId: siteId === noSite ? undefined : siteId, mode: "confirm" },
                    {
                        onSuccess: (answered) => {
                            onOpenChange(false);
                            onCreated(answered.conversation);
                        },
                    },
                );
            }}
        >
            <Field
                label={copy.agent.screen.newSite}
                hint={siteId === noSite ? copy.agent.screen.newEverywhere : undefined}
            >
                {(control) => (
                    <Select
                        id={control.id}
                        value={siteId}
                        options={[...sites, { value: noSite, label: copy.agent.header.everywhere }]}
                        onValueChange={setSiteId}
                    />
                )}
            </Field>
            {reaction === null || reaction.kind === "silent" || reaction.kind === "unlock" ? null : (
                <p className="text-xs text-danger">{reaction.message}</p>
            )}
        </Dialog>
    );
}
