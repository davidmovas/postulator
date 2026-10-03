import type { ReactElement, ReactNode } from "react";
import { useMemo } from "react";
import { useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { pickOpenFile } from "../../data/host.js";
import { Banner, Button, Screen, SkeletonRows, Tabs, Toolbar } from "../../ui/index.js";
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
import { chosenRows, columnsNotice, inUse, rowsOf, settingsOf } from "./workbook.js";

export function ImportScreen(): ReactElement {
    const params = useParams();
    const siteId = params.siteId ?? "";
    const [searchParams, setSearchParams] = useSearchParams();
    const query = useMemo(() => readQuery(searchParams), [searchParams]);
    const flow = useImportFlow(siteId, query.path, query.step === "preview");
    const book = flow.book;

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
                change({ ...query, path: picked, step: "columns" });
            }
        });
    };

    const inspected = flow.inspect.data;
    const previewed = flow.preview.data;
    const inspectFailure = flow.inspect.error === null ? null : react(flow.inspect.error);
    const previewFailure = flow.preview.error === null ? null : react(flow.preview.error);
    const sheets = inspected?.sheets ?? [];
    const fileRows = sheets.length > 1 ? rowsOf(sheets) : (inspected?.rows ?? null);

    const panel = ((): ReactNode => {
        if (query.tab === "export") {
            return undefined;
        }
        switch (query.step) {
            case "file":
                return (
                    <MappingsPanel
                        siteId={siteId}
                        inUse={book === null ? [] : inUse(book)}
                        onUse={(mapping) => {
                            flow.adopt(mapping);
                        }}
                    />
                );
            case "columns":
                return book === null ? undefined : (
                    <OptionsPanel
                        siteId={siteId}
                        sheets={book.sheets}
                        chosen={book.chosen}
                        active={book.active}
                        settings={settingsOf(book)}
                        headers={flow.reading.headers}
                        onChoose={flow.choose}
                        onOptions={flow.setOptions}
                        onHeaderless={flow.setHeaderless}
                        onSaved={flow.adoptHere}
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
                    rows={fileRows}
                    columns={inspected?.headers?.length ?? null}
                    sheets={sheets.length}
                    busy={flow.inspect.isPending}
                    recent={flow.recent}
                    onChoose={choose}
                    onOpen={(path) => {
                        change({ ...query, path, step: "columns" });
                    }}
                    onForget={flow.forgetFile}
                    onNext={() => {
                        goto("columns");
                    }}
                />
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
        if (flow.inspect.isPending || book === null) {
            return (
                <div className="p-4">
                    <SkeletonRows rows={8} label={copy.imports.file.reading} />
                </div>
            );
        }
        if (query.step === "columns") {
            return (
                <StepColumns
                    tabs={
                        book.sheets.length > 1
                            ? book.sheets
                                  .filter((sheet) => book.chosen.includes(sheet.name))
                                  .map((sheet) => ({ name: sheet.name, rows: sheet.rows }))
                            : []
                    }
                    active={book.active}
                    headers={flow.reading.headers}
                    sample={flow.reading.sample}
                    busy={flow.reading.busy}
                    failure={flow.reading.failure}
                    columns={settingsOf(book).columns}
                    detected={flow.detected}
                    notice={columnsNotice(book)}
                    onActivate={flow.activate}
                    onAssign={flow.assign}
                    onBack={() => {
                        goto(stepBefore("columns"));
                    }}
                    onNext={() => {
                        goto(stepAfter("columns"));
                    }}
                />
            );
        }
        if (flow.request === null) {
            return (
                <div className="p-4">
                    <Banner
                        tone="warn"
                        title={copy.imports.columns.noSheet}
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
                    }}
                />
            );
        }
        return (
            <StepApply
                siteId={siteId}
                rows={chosenRows(book)}
                request={flow.request}
                applied={flow.apply.variables ?? null}
                blocked={(previewed?.report.errors ?? []).length > 0 ? copy.imports.preview.blocked : null}
                saveAs={flow.saveAs}
                counts={flow.apply.data?.counts ?? null}
                busy={flow.apply.isPending}
                thrown={flow.apply.error}
                onSaveAs={flow.setSaveAs}
                onBack={() => {
                    goto("preview");
                }}
                onApply={flow.startApply}
                onAgain={() => {
                    change({ tab: "import", step: "file", path: "" });
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
