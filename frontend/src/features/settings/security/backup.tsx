import type { ReactElement } from "react";
import { useState } from "react";
import { Link, useNavigate } from "react-router";

import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { formErrorOf } from "../../../data/errors.js";
import { pickOpenFile, pickSaveFile } from "../../../data/host.js";
import { useRuns } from "../../../data/hooks/runs.js";
import { useExportBackup, useImportBackup } from "../../../data/hooks/settings.js";
import { bytes } from "../../../domain/format.js";
import { Banner, Button, Dialog, Field, Input, Panel, PanelHeader, UploadFileIcon } from "../../../ui/index.js";

const said = copy.settings.security.backup;

const archiveFilter = { displayName: said.filter, pattern: "*.pstx" };

function ExportPanel(): ReactElement {
    const write = useExportBackup();
    const [path, setPath] = useState<string | null>(null);
    const [password, setPassword] = useState("");

    return (
        <Panel>
            <PanelHeader title={said.exportTitle} />
            <div className="flex flex-col gap-2 p-3">
                <p className="text-xs text-ink-dim">{said.exportBody}</p>
                <div className="flex items-center gap-2">
                    <Button
                        variant="secondary"
                        onClick={() => {
                            void pickSaveFile({ title: said.exportTitle, filters: [archiveFilter] }).then(setPath);
                        }}
                    >
                        {said.choose}
                    </Button>
                    <span className="min-w-0 truncate font-mono text-2xs text-ink-faint">{path ?? said.noFile}</span>
                </div>
                <Field label={said.password} hint={said.passwordHint} required={true}>
                    {(binding) => (
                        <Input
                            id={binding.id}
                            aria-describedby={binding["aria-describedby"]}
                            type="password"
                            autoComplete="off"
                            value={password}
                            onChange={(event) => {
                                setPassword(event.target.value);
                            }}
                        />
                    )}
                </Field>
                {formErrorOf(write.error) === null ? null : (
                    <Banner tone="danger" title={formErrorOf(write.error) ?? ""} />
                )}
                {write.data === undefined ? null : (
                    <Banner tone="ok" title={said.written(write.data.path, bytes(write.data.bytes))} />
                )}
                <div>
                    <Button
                        variant="primary"
                        disabled={path === null || password === ""}
                        busy={write.isPending}
                        onClick={() => {
                            if (path === null) {
                                return;
                            }
                            write.mutate({ path, password });
                        }}
                    >
                        {write.isPending ? said.exporting : said.export}
                    </Button>
                </div>
            </div>
        </Panel>
    );
}

function ImportPanel(): ReactElement {
    const navigate = useNavigate();
    const read = useImportBackup();
    const running = useRuns({ status: "running" }, null, 20);
    const [path, setPath] = useState<string | null>(null);
    const [password, setPassword] = useState("");
    const [asking, setAsking] = useState(false);

    const active = flatten(running.data?.pages)[0] ?? null;

    return (
        <Panel>
            <PanelHeader title={said.importTitle} />
            <div className="flex flex-col gap-2 p-3">
                <p className="text-xs text-ink-dim">{said.importBody}</p>
                {active === null ? null : (
                    <Banner
                        tone="warn"
                        title={said.running}
                        actions={
                            <Link
                                to={`/s/${active.siteId}/runs/${active.id}`}
                                className="text-xs font-semibold text-accent hover:underline"
                            >
                                {said.openRun}
                            </Link>
                        }
                    />
                )}
                <div className="flex items-center gap-2">
                    <Button
                        variant="secondary"
                        icon={UploadFileIcon}
                        onClick={() => {
                            void pickOpenFile({ title: said.importTitle, filters: [archiveFilter] }).then(setPath);
                        }}
                    >
                        {said.choose}
                    </Button>
                    <span className="min-w-0 truncate font-mono text-2xs text-ink-faint">{path ?? said.noFile}</span>
                </div>
                <Field label={said.password} required={true}>
                    {(binding) => (
                        <Input
                            id={binding.id}
                            type="password"
                            autoComplete="off"
                            value={password}
                            onChange={(event) => {
                                setPassword(event.target.value);
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
                        disabled={path === null || password === ""}
                        busy={read.isPending}
                        onClick={() => {
                            setAsking(true);
                        }}
                    >
                        {read.isPending ? said.importing : said.import}
                    </Button>
                </div>
            </div>

            <Dialog
                open={asking}
                onOpenChange={setAsking}
                title={said.importConfirmTitle}
                description={said.importConfirmBody}
                confirmLabel={said.importConfirm}
                cancelLabel={copy.app.cancel}
                destructive={true}
                busy={read.isPending}
                onConfirm={() => {
                    if (path === null) {
                        return;
                    }
                    read.mutate(
                        { path, password },
                        {
                            onSuccess: () => {
                                setAsking(false);
                                setPassword("");
                                void navigate("/sites");
                            },
                        },
                    );
                }}
            />
        </Panel>
    );
}

export function BackupPanels(): ReactElement {
    return (
        <>
            <ExportPanel />
            <ImportPanel />
        </>
    );
}
