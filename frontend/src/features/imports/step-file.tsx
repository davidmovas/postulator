import type { DragEvent, ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { relativeTime } from "../../domain/format.js";
import {
    Button,
    CloseIcon,
    cx,
    DescriptionIcon,
    IconButton,
    SectionLabel,
    Spinner,
    UploadFileIcon,
} from "../../ui/index.js";
import { fileName } from "./recent.js";
import type { RecentFile } from "./recent.js";

export interface StepFileProps {
    path: string;
    rows: number | null;
    columns: number | null;
    busy: boolean;
    recent: readonly RecentFile[];
    onChoose: () => void;
    onOpen: (path: string) => void;
    onForget: (path: string) => void;
    onNext: () => void;
}

export function StepFile({
    path,
    rows,
    columns,
    busy,
    recent,
    onChoose,
    onOpen,
    onForget,
    onNext,
}: StepFileProps): ReactElement {
    const [over, setOver] = useState(false);
    const leave = (event: DragEvent<HTMLDivElement>): void => {
        event.preventDefault();
        setOver(false);
    };

    return (
        <div className="flex flex-col gap-4 p-4">
            <div
                data-drop-target={true}
                onDragEnter={(event) => {
                    event.preventDefault();
                    setOver(true);
                }}
                onDragOver={(event) => {
                    event.preventDefault();
                    setOver(true);
                }}
                onDragLeave={leave}
                onDrop={leave}
                className={cx(
                    "flex flex-col items-center gap-3 rounded-lg border border-dashed px-4 py-8 transition-colors duration-100",
                    over ? "border-accent bg-accent-soft" : "border-hairline bg-panel",
                )}
            >
                <UploadFileIcon size={20} className={over ? "text-accent" : "text-ink-faint"} />
                {over ? <p className="text-sm font-semibold text-accent">{copy.imports.dropped.hint}</p> : null}
                {path === "" ? (
                    <>
                        <p className="text-sm font-semibold text-ink">{copy.imports.file.title}</p>
                        <p className="text-xs text-ink-dim">{copy.imports.file.body}</p>
                        <Button variant="primary" onClick={onChoose}>
                            {copy.imports.file.choose}
                        </Button>
                    </>
                ) : (
                    <>
                        <p className="max-w-160 truncate font-mono text-sm text-ink" title={path}>
                            {fileName(path)}
                        </p>
                        <p className="flex items-center gap-2 text-xs text-ink-dim">
                            {busy ? (
                                <>
                                    <Spinner size={12} />
                                    {copy.imports.file.reading}
                                </>
                            ) : rows === null ? (
                                copy.imports.file.gone
                            ) : (
                                `${copy.imports.file.rows(rows)} · ${copy.imports.file.columns(columns ?? 0)}`
                            )}
                        </p>
                        <div className="flex gap-2">
                            <Button variant="secondary" onClick={onChoose}>
                                {copy.imports.file.change}
                            </Button>
                            <Button variant="primary" disabled={busy || rows === null} onClick={onNext}>
                                {copy.imports.next}
                            </Button>
                        </div>
                    </>
                )}
            </div>
            {recent.length === 0 ? null : (
                <section className="flex flex-col gap-1">
                    <SectionLabel>{copy.imports.file.recent}</SectionLabel>
                    <ul className="flex flex-col overflow-hidden rounded-lg border border-hairline">
                        {recent.map((file) => (
                            <li
                                key={file.path}
                                className="flex h-7 items-center gap-2 border-b border-hairline px-3 last:border-b-0 hover:bg-raised"
                            >
                                <button
                                    type="button"
                                    title={file.path}
                                    onClick={() => {
                                        onOpen(file.path);
                                    }}
                                    className="flex min-w-0 flex-1 items-center gap-2 text-left"
                                >
                                    <DescriptionIcon size={13} className="shrink-0 text-ink-faint" />
                                    <span className="truncate font-mono text-xs text-ink">{fileName(file.path)}</span>
                                </button>
                                <span className="shrink-0 font-mono text-2xs text-ink-faint">
                                    {copy.imports.file.rows(file.rows)}
                                </span>
                                <span className="w-20 shrink-0 text-right text-2xs text-ink-faint">
                                    {relativeTime(file.at)}
                                </span>
                                <IconButton
                                    icon={CloseIcon}
                                    label={copy.imports.file.forget}
                                    size="sm"
                                    variant="ghost"
                                    onClick={() => {
                                        onForget(file.path);
                                    }}
                                />
                            </li>
                        ))}
                    </ul>
                </section>
            )}
        </div>
    );
}
