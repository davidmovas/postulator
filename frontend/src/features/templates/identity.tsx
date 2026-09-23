import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { Dialog, Field, Input } from "../../ui/index.js";
import { fieldErrorOf } from "./controls.js";

export interface IdentityDialogProps {
    open: boolean;
    name: string;
    pageKind: string;
    busy: boolean;
    error: unknown;
    onOpenChange: (open: boolean) => void;
    onApply: (identity: { name: string; pageKind: string }) => void;
}

export function IdentityDialog({
    open,
    name,
    pageKind,
    busy,
    error,
    onOpenChange,
    onApply,
}: IdentityDialogProps): ReactElement {
    const [draftName, setDraftName] = useState(name);
    const [draftKind, setDraftKind] = useState(pageKind);

    return (
        <Dialog
            open={open}
            onOpenChange={(next) => {
                if (next) {
                    setDraftName(name);
                    setDraftKind(pageKind);
                }
                onOpenChange(next);
            }}
            title={copy.templates.editor.changeKindTitle}
            description={copy.templates.editor.changeKindBody}
            confirmLabel={copy.templates.editor.changeKindConfirm}
            cancelLabel={copy.templates.editor.cancel}
            busy={busy}
            onConfirm={() => {
                onApply({ name: draftName.trim(), pageKind: draftKind.trim() });
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
                <Field
                    label={copy.templates.editor.pageKind}
                    hint={copy.templates.create.pageKindHint}
                    required={true}
                    error={fieldErrorOf(error, "pageKind")}
                >
                    {(control) => (
                        <Input
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            mono={true}
                            value={draftKind}
                            onChange={(event) => {
                                setDraftKind(event.target.value);
                            }}
                        />
                    )}
                </Field>
            </div>
        </Dialog>
    );
}
