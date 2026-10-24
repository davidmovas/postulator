import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { fieldErrorOf } from "../../data/errors.js";
import { Dialog, Field, Input } from "../../ui/index.js";

export interface IdentityDialogProps {
    open: boolean;
    name: string;
    busy: boolean;
    error: unknown;
    onOpenChange: (open: boolean) => void;
    onApply: (name: string) => void;
}

export function IdentityDialog({ open, name, busy, error, onOpenChange, onApply }: IdentityDialogProps): ReactElement {
    const [draftName, setDraftName] = useState(name);

    return (
        <Dialog
            open={open}
            onOpenChange={(next) => {
                if (next) {
                    setDraftName(name);
                }
                onOpenChange(next);
            }}
            title={copy.templates.editor.renameTitle}
            description={copy.templates.editor.renameBody}
            confirmLabel={copy.templates.editor.renameConfirm}
            cancelLabel={copy.templates.editor.cancel}
            busy={busy}
            onConfirm={() => {
                onApply(draftName.trim());
            }}
        >
            <div className="flex flex-col gap-3">
                <Field label={copy.templates.editor.name} required={true} error={fieldErrorOf(error, "name")}>
                    {(control) => (
                        <Input
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            value={draftName}
                            onChange={(event) => {
                                setDraftName(event.target.value);
                            }}
                        />
                    )}
                </Field>
            </div>
        </Dialog>
    );
}
