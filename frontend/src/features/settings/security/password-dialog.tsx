import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { formErrorOf } from "../../../data/errors.js";
import { useSetMasterPassword } from "../../../data/hooks/settings.js";
import { Banner, Dialog, Field, Input, ShieldIcon } from "../../../ui/index.js";

const said = copy.settings.security.password;

export interface PasswordDialogProps {
    open: boolean;
    protectedByPassword: boolean;
    onOpenChange: (open: boolean) => void;
}

export function PasswordDialog({ open, protectedByPassword, onOpenChange }: PasswordDialogProps): ReactElement {
    const change = useSetMasterPassword();
    const [current, setCurrent] = useState("");
    const [next, setNext] = useState("");
    const [repeat, setRepeat] = useState("");

    const mismatch = next !== "" && repeat !== "" && next !== repeat;
    const unchanged = next !== "" && next === current;
    const ready = next !== "" && next === repeat && !unchanged && (!protectedByPassword || current !== "");

    const close = (): void => {
        setCurrent("");
        setNext("");
        setRepeat("");
        onOpenChange(false);
    };

    return (
        <Dialog
            open={open}
            onOpenChange={(staying) => {
                if (!staying) {
                    close();
                }
            }}
            title={protectedByPassword ? said.changeTitle : said.setTitle}
            description={protectedByPassword ? said.changeBody : said.setBody}
            confirmLabel={said.save}
            cancelLabel={copy.app.cancel}
            icon={ShieldIcon}
            busy={change.isPending}
            onConfirm={() => {
                if (!ready) {
                    return;
                }
                change.mutate({ current, new: next }, { onSuccess: close });
            }}
        >
            <div className="flex flex-col gap-2">
                {protectedByPassword ? (
                    <Field label={said.current} required={true}>
                        {(binding) => (
                            <Input
                                id={binding.id}
                                type="password"
                                autoComplete="off"
                                value={current}
                                onChange={(event) => {
                                    setCurrent(event.target.value);
                                }}
                            />
                        )}
                    </Field>
                ) : null}
                <Field label={said.next} required={true} error={unchanged ? said.unchanged : null}>
                    {(binding) => (
                        <Input
                            id={binding.id}
                            type="password"
                            autoComplete="new-password"
                            invalid={binding.invalid}
                            value={next}
                            onChange={(event) => {
                                setNext(event.target.value);
                            }}
                        />
                    )}
                </Field>
                <Field label={said.repeat} required={true} error={mismatch ? said.mismatch : null}>
                    {(binding) => (
                        <Input
                            id={binding.id}
                            type="password"
                            autoComplete="new-password"
                            invalid={binding.invalid}
                            value={repeat}
                            onChange={(event) => {
                                setRepeat(event.target.value);
                            }}
                        />
                    )}
                </Field>
                {formErrorOf(change.error) === null ? null : (
                    <Banner tone="danger" title={formErrorOf(change.error) ?? ""} />
                )}
            </div>
        </Dialog>
    );
}
