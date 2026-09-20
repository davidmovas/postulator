import type { ReactElement } from "react";
import { useState } from "react";
import { useNavigate } from "react-router";

import { copy } from "../../copy/index.js";
import { formErrorOf } from "../../data/errors.js";
import { pickOpenFile, pickSaveFile } from "../../data/host.js";
import { useExportBackup, useImportBackup, useLock, useSetMasterPassword } from "../../data/hooks/settings.js";
import { useRuns } from "../../data/hooks/runs.js";
import { useLockGate } from "../../data/lock.js";
import { flatten } from "../../data/call.js";
import { bytes } from "../../domain/format.js";
import {
    Banner,
    Button,
    Dialog,
    Field,
    Input,
    LockIcon,
    Panel,
    PanelHeader,
    ShieldIcon,
    StatusBadge,
    UploadFileIcon,
} from "../../ui/index.js";

const archiveFilter = { displayName: copy.settings.security.backup.filter, pattern: "*.pstx" };

function LockPanel(): ReactElement {
    const gate = useLockGate();
    const running = useRuns({ status: "running" }, null, 20);
    const lock = useLock();
    const [asking, setAsking] = useState(false);

    const active = flatten(running.data?.pages).length;

    return (
        <Panel>
            <PanelHeader title={copy.settings.security.lock.title} />
            <div className="flex flex-col gap-2 p-3">
                <p className="text-xs text-ink-dim">{copy.settings.security.lock.blurb}</p>
                <StatusBadge tone={gate.protectedByPassword ? "ok" : "warn"} icon={ShieldIcon}>
                    {gate.protectedByPassword
                        ? copy.settings.security.lock.protected
                        : copy.settings.security.lock.unprotected}
                </StatusBadge>
                <div className="flex items-center gap-2">
                    <Button
                        variant="danger"
                        icon={LockIcon}
                        disabled={!gate.protectedByPassword}
                        onClick={() => {
                            setAsking(true);
                        }}
                    >
                        {copy.lock.lockNow}
                    </Button>
                    {gate.protectedByPassword ? null : (
                        <span className="text-xs text-ink-dim">{copy.settings.security.lock.cannotLock}</span>
                    )}
                </div>
            </div>

            <Dialog
                open={asking}
                onOpenChange={setAsking}
                title={copy.settings.security.lock.confirmTitle}
                description={copy.settings.security.lock.confirmBody(active)}
                confirmLabel={copy.settings.security.lock.confirm}
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

function MasterPasswordPanel(): ReactElement {
    const gate = useLockGate();
    const change = useSetMasterPassword();

    const [current, setCurrent] = useState("");
    const [next, setNext] = useState("");
    const [repeat, setRepeat] = useState("");
    const [removing, setRemoving] = useState(false);

    const mismatch = next !== "" && repeat !== "" && next !== repeat;
    const unchanged = next !== "" && next === current;
    const ready = next !== "" && next === repeat && !unchanged && (!gate.protectedByPassword || current !== "");

    const clear = (): void => {
        setCurrent("");
        setNext("");
        setRepeat("");
    };

    return (
        <Panel>
            <PanelHeader title={copy.settings.security.password.title} />
            <div className="flex flex-col gap-2 p-3">
                <p className="text-xs text-ink-dim">
                    {gate.protectedByPassword
                        ? copy.settings.security.password.changeBlurb
                        : copy.settings.security.password.setBlurb}
                </p>
                {gate.protectedByPassword ? (
                    <Field label={copy.settings.security.password.current} required={true}>
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
                <Field label={copy.settings.security.password.next} required={true} error={unchanged ? copy.settings.security.password.unchanged : null}>
                    {(binding) => (
                        <Input
                            id={binding.id}
                            type="password"
                            autoComplete="new-password"
                            value={next}
                            onChange={(event) => {
                                setNext(event.target.value);
                            }}
                        />
                    )}
                </Field>
                <Field
                    label={copy.settings.security.password.repeat}
                    required={true}
                    error={mismatch ? copy.settings.security.password.mismatch : null}
                >
                    {(binding) => (
                        <Input
                            id={binding.id}
                            type="password"
                            autoComplete="new-password"
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
                <div className="flex items-center gap-2">
                    <Button
                        variant="primary"
                        disabled={!ready}
                        busy={change.isPending}
                        onClick={() => {
                            change.mutate({ current, new: next }, { onSuccess: clear });
                        }}
                    >
                        {gate.protectedByPassword
                            ? copy.settings.security.password.change
                            : copy.settings.security.password.set}
                    </Button>
                    {gate.protectedByPassword ? (
                        <Button
                            variant="ghost"
                            disabled={current === ""}
                            onClick={() => {
                                setRemoving(true);
                            }}
                        >
                            {copy.settings.security.password.remove}
                        </Button>
                    ) : null}
                </div>
            </div>

            <Dialog
                open={removing}
                onOpenChange={setRemoving}
                title={copy.settings.security.password.removeTitle}
                description={copy.settings.security.password.removeBody}
                confirmLabel={copy.settings.security.password.removeConfirm}
                cancelLabel={copy.app.cancel}
                destructive={true}
                busy={change.isPending}
                onConfirm={() => {
                    change.mutate(
                        { current, new: "" },
                        {
                            onSuccess: () => {
                                clear();
                                setRemoving(false);
                            },
                        },
                    );
                }}
            />
        </Panel>
    );
}

function BackupPanel(): ReactElement {
    const navigate = useNavigate();
    const write = useExportBackup();
    const read = useImportBackup();

    const [exportPath, setExportPath] = useState<string | null>(null);
    const [exportPassword, setExportPassword] = useState("");
    const [importPath, setImportPath] = useState<string | null>(null);
    const [importPassword, setImportPassword] = useState("");
    const [replacing, setReplacing] = useState(false);

    return (
        <Panel>
            <PanelHeader title={copy.settings.security.backup.title} />
            <div className="flex flex-col gap-3 p-3">
                <p className="text-xs text-ink-dim">{copy.settings.security.backup.blurb}</p>

                <div className="flex flex-col gap-2 rounded-md border border-hairline p-3">
                    <p className="text-sm font-semibold text-ink">{copy.settings.security.backup.export}</p>
                    <div className="flex items-center gap-2">
                        <Button
                            variant="secondary"
                            onClick={() => {
                                void pickSaveFile({
                                    title: copy.settings.security.backup.export,
                                    filters: [archiveFilter],
                                }).then(setExportPath);
                            }}
                        >
                            {copy.settings.security.backup.choose}
                        </Button>
                        <span className="min-w-0 truncate font-mono text-2xs text-ink-faint">
                            {exportPath ?? copy.settings.security.backup.noFile}
                        </span>
                    </div>
                    <Field
                        label={copy.settings.security.backup.password}
                        hint={copy.settings.security.backup.passwordHint}
                        required={true}
                    >
                        {(binding) => (
                            <Input
                                id={binding.id}
                                aria-describedby={binding["aria-describedby"]}
                                type="password"
                                autoComplete="off"
                                value={exportPassword}
                                onChange={(event) => {
                                    setExportPassword(event.target.value);
                                }}
                            />
                        )}
                    </Field>
                    {formErrorOf(write.error) === null ? null : (
                        <Banner tone="danger" title={formErrorOf(write.error) ?? ""} />
                    )}
                    {write.data === undefined ? null : (
                        <Banner
                            tone="ok"
                            title={copy.settings.security.backup.written(write.data.path, bytes(write.data.bytes))}
                        />
                    )}
                    <div>
                        <Button
                            variant="primary"
                            disabled={exportPath === null || exportPassword === ""}
                            busy={write.isPending}
                            onClick={() => {
                                if (exportPath === null) {
                                    return;
                                }
                                write.mutate({ path: exportPath, password: exportPassword });
                            }}
                        >
                            {write.isPending
                                ? copy.settings.security.backup.exporting
                                : copy.settings.security.backup.export}
                        </Button>
                    </div>
                </div>

                <div className="flex flex-col gap-2 rounded-md border border-danger-border p-3">
                    <p className="text-sm font-semibold text-ink">{copy.settings.security.backup.import}</p>
                    <div className="flex items-center gap-2">
                        <Button
                            variant="secondary"
                            icon={UploadFileIcon}
                            onClick={() => {
                                void pickOpenFile({
                                    title: copy.settings.security.backup.import,
                                    filters: [archiveFilter],
                                }).then(setImportPath);
                            }}
                        >
                            {copy.settings.security.backup.choose}
                        </Button>
                        <span className="min-w-0 truncate font-mono text-2xs text-ink-faint">
                            {importPath ?? copy.settings.security.backup.noFile}
                        </span>
                    </div>
                    <Field label={copy.settings.security.backup.password} required={true}>
                        {(binding) => (
                            <Input
                                id={binding.id}
                                type="password"
                                autoComplete="off"
                                value={importPassword}
                                onChange={(event) => {
                                    setImportPassword(event.target.value);
                                }}
                            />
                        )}
                    </Field>
                    {formErrorOf(read.error) === null ? null : (
                        <Banner tone="danger" title={formErrorOf(read.error) ?? ""} />
                    )}
                    <div>
                        <Button
                            variant="danger"
                            disabled={importPath === null || importPassword === ""}
                            busy={read.isPending}
                            onClick={() => {
                                setReplacing(true);
                            }}
                        >
                            {read.isPending
                                ? copy.settings.security.backup.importing
                                : copy.settings.security.backup.import}
                        </Button>
                    </div>
                </div>
            </div>

            <Dialog
                open={replacing}
                onOpenChange={setReplacing}
                title={copy.settings.security.backup.importTitle}
                description={copy.settings.security.backup.importBody}
                confirmLabel={copy.settings.security.backup.importConfirm}
                cancelLabel={copy.app.cancel}
                destructive={true}
                busy={read.isPending}
                onConfirm={() => {
                    if (importPath === null) {
                        return;
                    }
                    read.mutate(
                        { path: importPath, password: importPassword },
                        {
                            onSuccess: () => {
                                setReplacing(false);
                                setImportPassword("");
                                void navigate("/sites");
                            },
                        },
                    );
                }}
            />
        </Panel>
    );
}

export function SecurityScreen(): ReactElement {
    return (
        <div className="flex max-w-2xl flex-col gap-3">
            <LockPanel />
            <MasterPasswordPanel />
            <BackupPanel />
        </div>
    );
}
