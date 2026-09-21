import type { ReactElement, ReactNode } from "react";
import { useMemo } from "react";
import { useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { pickOpenFile } from "../../data/host.js";
import type { ImportField } from "../../generated/vocab.js";
import { Banner, Button, Screen, SkeletonRows, Tabs, Toolbar } from "../../ui/index.js";
import { assign } from "./columns.js";
import { ExportTab } from "./export-tab.js";
import { FindingsPanel } from "./findings.js";
import { useImportFlow } from "./flow.js";
import { MappingsPanel } from "./mappings-panel.js";
import { OptionsPanel } from "./options-panel.js";
import type { ImportQuery, ImportStep, ImportTab } from "./params.js";
import { readQuery, stepAfter, stepBefore, writeQuery } from "./params.js";
import { StepApply } from "./step-apply.js";
import { StepColumns } from "./step-columns.js";
import { StepFile } from "./step-file.js";
import { StepPreview } from "./step-preview.js";
import { Stepper } from "./stepper.js";

export function ImportScreen(): ReactElement {
    const params = useParams();
    const siteId = params.siteId ?? "";
    const [searchParams, setSearchParams] = useSearchParams();
    const query = useMemo(() => readQuery(searchParams), [searchParams]);
    const flow = useImportFlow(siteId, query.path, query.step === "preview");

    const change = (next: ImportQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    const goto = (step: ImportStep): void => {
        change({ ...query, step });
    };

    const choose = (): void => {
        void pickOpenFile({
            title: copy.imports.file.dialogTitle,
            filters: [{ displayName: copy.imports.file.spreadsheets, pattern: "*.csv;*.xlsx" }],
        }).then((picked) => {
            if (picked !== null) {
                change({ ...query, path: picked, mappingId: "", step: "columns" });
            }
        });
    };

    const inspected = flow.inspect.data;
    const previewed = flow.preview.data;
    const inspectFailure = flow.inspect.error === null ? null : react(flow.inspect.error);
    const previewFailure = flow.preview.error === null ? null : react(flow.preview.error);

    const panel = ((): ReactNode => {
        if (query.tab === "export") {
            return undefined;
        }
        switch (query.step) {
            case "file":
                return (
                    <MappingsPanel
                        siteId={siteId}
                        activeId={query.mappingId}
                        onUse={(mapping) => {
                            flow.useSaved(mapping);
                            change({ ...query, mappingId: mapping.id ?? "" });
                        }}
                    />
                );
            case "columns":
                return (
                    <OptionsPanel
                        siteId={siteId}
                        mapping={flow.mapping}
                        options={flow.options}
                        onOptions={flow.setOptions}
                        onSaved={(mapping) => {
                            change({ ...query, mappingId: mapping.id ?? "" });
                        }}
                    />
                );
            case "preview":
                return previewed === undefined ? undefined : (
                    <FindingsPanel
                        errors={previewed.report.errors ?? []}
                        warnings={previewed.report.warnings ?? []}
                        conflicts={previewed.report.cannibalization ?? []}
                    />
                );
            default:
                return undefined;
        }
    })();

    const body = ((): ReactNode => {
        if (query.tab === "export") {
            return <ExportTab siteId={siteId} />;
        }
        if (query.step === "file") {
            return (
                <StepFile
                    path={query.path}
                    rows={inspected?.rows ?? null}
                    columns={inspected?.headers?.length ?? null}
                    busy={flow.inspect.isPending}
                    recent={flow.recent}
                    onChoose={choose}
                    onOpen={(path) => {
                        change({ ...query, path, mappingId: "", step: "columns" });
                    }}
                    onForget={flow.forgetFile}
                    onNext={() => {
                        goto("columns");
                    }}
                />
            );
        }
        if (flow.inspect.isPending) {
            return (
                <div className="p-4">
                    <SkeletonRows rows={8} label={copy.imports.file.reading} />
                </div>
            );
        }
        if (inspectFailure !== null && inspectFailure.kind !== "silent" && inspectFailure.kind !== "unlock") {
            return (
                <div className="p-4">
                    <Banner
                        tone="danger"
                        title={inspectFailure.message}
                        actions={
                            <Button size="sm" variant="secondary" onClick={choose}>
                                {copy.imports.file.change}
                            </Button>
                        }
                    />
                </div>
            );
        }
        if (query.step === "columns") {
            return (
                <StepColumns
                    headers={inspected?.headers ?? []}
                    sample={inspected?.sample ?? []}
                    columns={flow.columns}
                    detected={flow.detected}
                    onAssign={(header: string, field: ImportField | null) => {
                        flow.setColumns(assign(flow.columns, header, field));
                    }}
                    onBack={() => {
                        goto(stepBefore("columns"));
                    }}
                    onNext={() => {
                        goto(stepAfter("columns"));
                    }}
                />
            );
        }
        if (query.step === "preview") {
            if (previewFailure !== null && previewFailure.kind !== "silent" && previewFailure.kind !== "unlock") {
                return (
                    <div className="p-4">
                        <Banner
                            tone="danger"
                            title={previewFailure.message}
                            actions={
                                <Button
                                    size="sm"
                                    variant="secondary"
                                    onClick={() => {
                                        goto("columns");
                                    }}
                                >
                                    {copy.imports.back}
                                </Button>
                            }
                        />
                    </div>
                );
            }
            if (previewed === undefined) {
                return (
                    <div className="p-4">
                        <SkeletonRows rows={8} label={copy.imports.preview.title} />
                    </div>
                );
            }
            return (
                <StepPreview
                    report={previewed.report}
                    onBack={() => {
                        goto("columns");
                    }}
                    onApply={() => {
                        goto("apply");
                        flow.startApply("");
                    }}
                />
            );
        }
        return (
            <StepApply
                siteId={siteId}
                rows={inspected?.rows ?? 0}
                counts={flow.apply.data?.counts ?? null}
                busy={flow.apply.isPending}
                thrown={flow.apply.error}
                onBack={() => {
                    goto("preview");
                }}
                onApply={() => {
                    flow.startApply("");
                }}
                onAgain={() => {
                    change({ tab: "import", step: "file", path: "", mappingId: "" });
                }}
            />
        );
    })();

    return (
        <Screen
            title={copy.imports.title}
            tabs={
                <Tabs<ImportTab>
                    label={copy.imports.tabs.label}
                    value={query.tab}
                    items={[
                        { key: "import", label: copy.imports.tabs.import },
                        { key: "export", label: copy.imports.tabs.export },
                    ]}
                    onValueChange={(tab) => {
                        change({ ...query, tab });
                    }}
                />
            }
            toolbar={
                query.tab === "export" ? undefined : (
                    <Toolbar label={copy.imports.steps.label}>
                        <Stepper
                            step={query.step}
                            path={query.path}
                            locked={flow.apply.isPending}
                            onStep={goto}
                        />
                    </Toolbar>
                )
            }
            variant="split"
            right={panel}
        >
            {body}
        </Screen>
    );
}
