import type { ReactElement } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import type { ImportCounts } from "../../data/types.js";
import {
    Banner,
    Button,
    CheckCircleIcon,
    Field,
    Input,
    Spinner,
    TableRowsIcon,
    UploadFileIcon,
} from "../../ui/index.js";
import type { SheetsRead } from "./workbook.js";
import { savedNames, sheetsOf } from "./workbook.js";

interface TileProps {
    label: string;
    value: number;
    strong?: boolean;
}

function Tile({ label, value, strong = false }: TileProps): ReactElement {
    return (
        <div className="flex min-w-28 flex-col gap-0.5 rounded-md border border-hairline bg-inset px-3 py-2">
            <span className={strong && value > 0 ? "font-mono text-xl text-accent" : "font-mono text-xl text-ink"}>
                {value}
            </span>
            <span className="text-2xs text-ink-dim">{label}</span>
        </div>
    );
}

interface AppliedRequest extends SheetsRead {
    options: { saveMappingAs?: string };
}

export interface StepApplyProps {
    siteId: string;
    rows: number;
    request: SheetsRead | null;
    applied: AppliedRequest | null;
    blocked: string | null;
    saveAs: string;
    counts: ImportCounts | null;
    busy: boolean;
    thrown: unknown;
    onSaveAs: (name: string) => void;
    onBack: () => void;
    onApply: () => void;
    onAgain: () => void;
}

export function StepApply({
    siteId,
    rows,
    request,
    applied,
    blocked,
    saveAs,
    counts,
    busy,
    thrown,
    onSaveAs,
    onBack,
    onApply,
    onAgain,
}: StepApplyProps): ReactElement {
    const failure = thrown === null || thrown === undefined ? null : react(thrown);

    if (busy) {
        return (
            <div className="flex flex-col items-center gap-3 p-8">
                <Spinner size={20} />
                <p className="text-sm text-ink">{copy.imports.apply.working(rows)}</p>
            </div>
        );
    }

    if (counts === null) {
        const sheets = sheetsOf(request);
        const example = saveAs.trim() === "" ? copy.imports.apply.savePlaceholder : saveAs;
        return (
            <div className="flex flex-col gap-3 p-4">
                {failure === null || failure.kind === "silent" || failure.kind === "unlock" ? null : (
                    <Banner tone="danger" title={failure.message} />
                )}
                <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed border-hairline bg-panel px-4 py-8">
                    <TableRowsIcon size={20} className="text-ink-faint" />
                    <p className="text-sm font-semibold text-ink">
                        {sheets.length > 1 ? copy.imports.apply.titleWorkbook : copy.imports.apply.title}
                    </p>
                    <p className="text-xs text-ink-dim">
                        {sheets.length === 0
                            ? copy.imports.file.rows(rows)
                            : `${copy.imports.apply.reads(sheets)} ${copy.imports.file.rows(rows)}.`}
                    </p>
                    <Field
                        className="w-full max-w-120"
                        label={copy.imports.apply.saveAs}
                        hint={
                            sheets.length > 1
                                ? copy.imports.apply.saveEach(savedNames(example, request))
                                : copy.imports.apply.saveOne
                        }
                    >
                        {(control) => (
                            <Input
                                id={control.id}
                                aria-describedby={control["aria-describedby"]}
                                data-mapping-name={true}
                                value={saveAs}
                                placeholder={copy.imports.apply.savePlaceholder}
                                onChange={(event) => {
                                    onSaveAs(event.target.value);
                                }}
                            />
                        )}
                    </Field>
                    {blocked === null ? null : <Banner tone="warn" title={blocked} className="w-full max-w-120" />}
                    <div className="flex gap-2">
                        <Button variant="secondary" onClick={onBack}>
                            {copy.imports.back}
                        </Button>
                        <Button
                            variant="primary"
                            data-import-apply={true}
                            disabled={blocked !== null}
                            title={blocked ?? undefined}
                            onClick={onApply}
                        >
                            {copy.imports.apply.start}
                        </Button>
                    </div>
                </div>
            </div>
        );
    }

    const done = sheetsOf(applied);
    const saved = savedNames(applied?.options.saveMappingAs ?? "", applied);
    return (
        <div className="flex flex-col gap-4 p-4">
            <div className="flex items-center gap-2">
                <CheckCircleIcon size={16} className="text-ok" />
                <h2 className="text-sm font-semibold text-ink">{copy.imports.apply.done}</h2>
            </div>
            {done.length === 0 && saved.length === 0 ? null : (
                <div className="flex flex-col gap-0.5 text-xs text-ink-dim">
                    {done.length === 0 ? null : <p>{copy.imports.apply.applied(done)}</p>}
                    {saved.length === 0 ? null : <p>{copy.imports.apply.saved(saved)}</p>}
                </div>
            )}
            <div className="flex flex-wrap gap-2">
                <Tile label={copy.imports.apply.pagesCreated} value={counts.pagesCreated} strong={true} />
                <Tile label={copy.imports.apply.pagesUpdated} value={counts.pagesUpdated} />
                <Tile label={copy.imports.apply.entitiesCreated} value={counts.entitiesCreated} strong={true} />
                <Tile label={copy.imports.apply.entitiesUpdated} value={counts.entitiesUpdated} />
                <Tile label={copy.imports.apply.edgesCreated} value={counts.edgesCreated} />
                <Tile label={copy.imports.apply.categoriesCreated} value={counts.categoriesCreated} strong={true} />
                <Tile label={copy.imports.apply.categoriesDeleted} value={counts.categoriesDeleted} />
                <Tile label={copy.imports.apply.skipped} value={counts.skipped} />
            </div>
            <div className="flex flex-wrap gap-2">
                <Button variant="primary" icon={UploadFileIcon} onClick={onAgain}>
                    {copy.imports.apply.again}
                </Button>
                <Link
                    to={`/s/${siteId}/graph`}
                    className="flex h-7 items-center rounded-md border border-hairline px-2.5 text-sm text-ink hover:bg-raised"
                >
                    {copy.imports.apply.openGraph}
                </Link>
                <Link
                    to={`/s/${siteId}/pages`}
                    className="flex h-7 items-center rounded-md border border-hairline px-2.5 text-sm text-ink hover:bg-raised"
                >
                    {copy.imports.apply.openPages}
                </Link>
            </div>
        </div>
    );
}
