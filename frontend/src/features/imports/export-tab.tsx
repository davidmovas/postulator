import type { ReactElement } from "react";
import { useState } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useExportSite } from "../../data/hooks/imports.js";
import { useSite } from "../../data/hooks/sites.js";
import { pickSaveFile } from "../../data/host.js";
import { Banner, Button, CheckCircleIcon, CloudUploadIcon, Field, Input } from "../../ui/index.js";
import { fileName } from "./recent.js";

function suggested(name: string): string {
    const slug = name
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-+|-+$/g, "");
    return `${slug === "" ? "postulator" : slug}-export.xlsx`;
}

export interface ExportTabProps {
    siteId: string;
}

export function ExportTab({ siteId }: ExportTabProps): ReactElement {
    const site = useSite(siteId === "" ? null : siteId);
    const exporting = useExportSite();
    const [path, setPath] = useState("");
    const written = exporting.data ?? null;
    const failure = exporting.error === null ? null : react(exporting.error);

    const choose = (): void => {
        void pickSaveFile({
            title: copy.imports.export.dialogTitle,
            filename: suggested(site.data?.site.name ?? ""),
            filters: [{ displayName: copy.imports.file.workbooks, pattern: "*.xlsx" }],
        }).then((picked) => {
            if (picked !== null) {
                setPath(picked);
                exporting.reset();
            }
        });
    };

    return (
        <div className="flex flex-col gap-4 p-4">
            <div className="flex max-w-160 flex-col gap-3 rounded-lg border border-hairline bg-panel p-4">
                <div className="flex items-center gap-2">
                    <CloudUploadIcon size={16} className="text-ink-faint" />
                    <h2 className="text-sm font-semibold text-ink">{copy.imports.export.title}</h2>
                </div>
                <p className="max-w-120 text-xs text-ink-dim">{copy.imports.export.body}</p>
                <Field label={copy.imports.export.destination}>
                    {(control) => (
                        <div className="flex items-center gap-2">
                            <Input
                                id={control.id}
                                data-export-path={true}
                                mono={true}
                                value={path}
                                placeholder={suggested(site.data?.site.name ?? "")}
                                title={path}
                                onChange={(event) => {
                                    setPath(event.target.value);
                                    exporting.reset();
                                }}
                            />
                            <Button variant="secondary" onClick={choose}>
                                {copy.imports.export.choose}
                            </Button>
                        </div>
                    )}
                </Field>
                {failure === null || failure.kind === "silent" || failure.kind === "unlock" ? null : (
                    <Banner tone="danger" title={failure.message} />
                )}
                <div className="flex items-center gap-2">
                    <Button
                        variant="primary"
                        data-export-start={true}
                        busy={exporting.isPending}
                        disabled={path === ""}
                        title={path === "" ? copy.imports.export.noDestination : undefined}
                        onClick={() => {
                            exporting.mutate({ siteId, path });
                        }}
                    >
                        {copy.imports.export.start}
                    </Button>
                    {written === null ? null : (
                        <span className="flex min-w-0 items-center gap-1.5 text-xs text-ink-dim">
                            <CheckCircleIcon size={14} className="shrink-0 text-ok" />
                            <span>{copy.imports.export.done(written.pages, written.entities)}</span>
                            <span className="truncate font-mono text-ink" title={written.path}>
                                {fileName(written.path)}
                            </span>
                        </span>
                    )}
                </div>
                {written === null ? null : (
                    <Link
                        to={`/s/${siteId}/pages`}
                        className="flex h-7 w-fit items-center rounded-md border border-hairline px-2.5 text-sm text-ink hover:bg-raised"
                    >
                        {copy.imports.apply.openPages}
                    </Link>
                )}
            </div>
        </div>
    );
}
