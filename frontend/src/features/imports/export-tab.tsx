import type { ReactElement } from "react";
import { useState } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useExportSite } from "../../data/hooks/imports.js";
import { useSite } from "../../data/hooks/sites.js";
import { pickSaveFile } from "../../data/host.js";
import { Banner, Button, CheckCircleIcon, CloudUploadIcon, Field, Input, Select } from "../../ui/index.js";
import type { SelectOption } from "../../ui/index.js";
import type { ExportFormat } from "../../generated/vocab.js";
import { exportFormats } from "../../generated/vocab.js";
import { exportFormatLabel } from "./labels.js";
import { fileName } from "./recent.js";

function suggested(name: string, format: ExportFormat): string {
    const slug = name
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-+|-+$/g, "");
    return `${slug === "" ? "postulator" : slug}-export.${format}`;
}

const formatOptions: readonly SelectOption<ExportFormat>[] = exportFormats.map((value) => ({
    value,
    label: exportFormatLabel(value),
}));

export interface ExportTabProps {
    siteId: string;
}

export function ExportTab({ siteId }: ExportTabProps): ReactElement {
    const site = useSite(siteId === "" ? null : siteId);
    const exporting = useExportSite();
    const [path, setPath] = useState("");
    const [format, setFormat] = useState<ExportFormat>(exportFormats[0]);
    const written = exporting.data ?? null;
    const failure = exporting.error === null ? null : react(exporting.error);

    const choose = (): void => {
        void pickSaveFile({
            title: copy.imports.export.dialogTitle,
            filename: suggested(site.data?.site.name ?? "", format),
            filters: [{ displayName: exportFormatLabel(format), pattern: `*.${format}` }],
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
                                placeholder={suggested(site.data?.site.name ?? "", format)}
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
                <Field label={copy.imports.export.format}>
                    {(control) => (
                        <div className="w-64">
                            <Select
                                id={control.id}
                                data-export-format={true}
                                value={format}
                                options={formatOptions}
                                onValueChange={(picked) => {
                                    setFormat(picked);
                                    exporting.reset();
                                }}
                            />
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
                            exporting.mutate({ siteId, path, format });
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
