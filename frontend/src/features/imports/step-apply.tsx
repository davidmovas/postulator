import type { ReactElement } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import type { ImportCounts } from "../../data/types.js";
import { Banner, Button, CheckCircleIcon, Spinner, TableRowsIcon, UploadFileIcon } from "../../ui/index.js";

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

export interface StepApplyProps {
    siteId: string;
    rows: number;
    counts: ImportCounts | null;
    busy: boolean;
    thrown: unknown;
    onBack: () => void;
    onApply: () => void;
    onAgain: () => void;
}

export function StepApply({
    siteId,
    rows,
    counts,
    busy,
    thrown,
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
        return (
            <div className="flex flex-col gap-3 p-4">
                {failure === null || failure.kind === "silent" || failure.kind === "unlock" ? null : (
                    <Banner tone="danger" title={failure.message} />
                )}
                <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed border-hairline bg-panel px-4 py-8">
                    <TableRowsIcon size={20} className="text-ink-faint" />
                    <p className="text-sm font-semibold text-ink">{copy.imports.apply.title}</p>
                    <p className="text-xs text-ink-dim">{copy.imports.file.rows(rows)}</p>
                    <div className="flex gap-2">
                        <Button variant="secondary" onClick={onBack}>
                            {copy.imports.back}
                        </Button>
                        <Button variant="primary" onClick={onApply}>
                            {copy.imports.apply.start}
                        </Button>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="flex flex-col gap-4 p-4">
            <div className="flex items-center gap-2">
                <CheckCircleIcon size={16} className="text-ok" />
                <h2 className="text-sm font-semibold text-ink">{copy.imports.apply.done}</h2>
            </div>
            <div className="flex flex-wrap gap-2">
                <Tile label={copy.imports.apply.pagesCreated} value={counts.pagesCreated} strong={true} />
                <Tile label={copy.imports.apply.pagesUpdated} value={counts.pagesUpdated} />
                <Tile label={copy.imports.apply.entitiesCreated} value={counts.entitiesCreated} strong={true} />
                <Tile label={copy.imports.apply.entitiesUpdated} value={counts.entitiesUpdated} />
                <Tile label={copy.imports.apply.edgesCreated} value={counts.edgesCreated} />
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
