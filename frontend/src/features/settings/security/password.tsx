import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { formErrorOf } from "../../../data/errors.js";
import { useRuns } from "../../../data/hooks/runs.js";
import { useLock, useSetMasterPassword } from "../../../data/hooks/settings.js";
import { useLockGate } from "../../../data/lock.js";
import { Banner, Button, Dialog, Field, Input, LockIcon, Panel, PanelHeader, ShieldIcon, StatusBadge } from "../../../ui/index.js";
import { PasswordDialog } from "./password-dialog.js";

const said = copy.settings.security;

function RemoveDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }): ReactElement {
    const change = useSetMasterPassword();
    const [current, setCurrent] = useState("");

    return (
        <Dialog
            open={open}
            onOpenChange={(staying) => {
                if (!staying) {
                    setCurrent("");
                    onOpenChange(false);
                }
            }}
            title={said.password.removeTitle}
            description={said.password.removeBody}
            confirmLabel={said.password.removeConfirm}
            cancelLabel={copy.app.cancel}
            destructive={true}
            busy={change.isPending}
            onConfirm={() => {
                if (current === "") {
                    return;
                }
                change.mutate(
                    { current, new: "" },
                    {
                        onSuccess: () => {
                            setCurrent("");
                            onOpenChange(false);
                        },
                    },
                );
            }}
        >
            <div className="flex flex-col gap-2">
                <Field label={said.password.current} required={true}>
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
                {formErrorOf(change.error) === null ? null : (
                    <Banner tone="danger" title={formErrorOf(change.error) ?? ""} />
                )}
            </div>
        </Dialog>
    );
}

export function MasterPasswordPanel(): ReactElement {
    const gate = useLockGate();
    const lock = useLock();
    const running = useRuns({ status: "running" }, null, 20);
    const [changing, setChanging] = useState(false);
    const [removing, setRemoving] = useState(false);
    const [locking, setLocking] = useState(false);

    const active = flatten(running.data?.pages).length;
    const held = gate.protectedByPassword;

    return (
        <Panel>
            <PanelHeader title={said.password.title}>
                <StatusBadge tone={held ? "ok" : "warn"} icon={ShieldIcon}>
                    {held ? said.lock.protected : said.lock.unprotected}
                </StatusBadge>
            </PanelHeader>
            <div className="flex flex-wrap items-center gap-2 p-3">
                <Button
                    variant={held ? "secondary" : "primary"}
                    onClick={() => {
                        setChanging(true);
                    }}
                >
                    {held ? said.password.change : said.password.set}
                </Button>
                {held ? (
                    <Button
                        variant="ghost"
                        onClick={() => {
                            setRemoving(true);
                        }}
                    >
                        {said.password.remove}
                    </Button>
                ) : null}
                <div className="ml-auto">
                    <Button
                        variant="secondary"
                        icon={LockIcon}
                        disabled={!held}
                        title={held ? undefined : said.lock.cannotLock}
                        onClick={() => {
                            setLocking(true);
                        }}
                    >
                        {said.lock.confirm}
                    </Button>
                </div>
            </div>

            <PasswordDialog open={changing} protectedByPassword={held} onOpenChange={setChanging} />
            <RemoveDialog open={removing} onOpenChange={setRemoving} />

            <Dialog
                open={locking}
                onOpenChange={setLocking}
                title={said.lock.confirmTitle}
                description={said.lock.confirmBody(active)}
                confirmLabel={said.lock.confirm}
                cancelLabel={copy.app.cancel}
                destructive={true}
                icon={LockIcon}
                busy={lock.isPending}
                onConfirm={() => {
                    lock.mutate(undefined);
                }}
            />
        </Panel>
    );
}
