import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { Checkbox, cx, SectionLabel, StatusBadge } from "../../../ui/index.js";
import { blockKindLabel } from "../../pages/model/labels.js";
import { fieldLabel } from "../../settings/types/labels.js";
import type { BlockChangeView, FieldChangeView } from "../artifacts.js";
import { blockExcerpt, fieldText } from "../artifacts.js";
import { blockState, blockTone } from "../diff-labels.js";

const said = copy.runs.review.diff;

const shortcode = "shortcode";

export interface Ticks {
    ticked: (key: string) => boolean;
    onToggle: (key: string) => void;
}

function Value({ label, fieldKey, value }: { label: string; fieldKey: string; value: string }): ReactElement {
    const text = fieldText(fieldKey, value);
    return (
        <div className="grid grid-cols-[4rem_minmax(0,1fr)] gap-x-2">
            <span className="text-2xs text-ink-faint">{label}</span>
            <p className={cx("text-xs whitespace-pre-wrap", text === "" ? "text-ink-faint italic" : "text-ink-soft")}>
                {text === "" ? said.empty : text}
            </p>
        </div>
    );
}

export interface FieldDiffProps {
    fields: readonly FieldChangeView[];
    empty?: string;
    ticks?: Ticks | null;
}

export function FieldDiff({ fields, empty, ticks = null }: FieldDiffProps): ReactElement | null {
    if (fields.length === 0 && empty === undefined) {
        return null;
    }
    return (
        <section aria-label={said.fields} className="flex flex-col">
            <SectionLabel className="px-3">{said.fields}</SectionLabel>
            {fields.length === 0 ? (
                <p className="px-3 py-1.5 text-xs text-ink-dim">{empty}</p>
            ) : (
                <ul className="flex flex-col">
                    {fields.map((change) => (
                        <li
                            key={change.key}
                            title={said.hashes(change.beforeHash, change.afterHash)}
                            className="flex flex-col gap-0.5 border-b border-inset px-3 py-1.5 last:border-b-0"
                        >
                            {ticks === null ? (
                                <span className="text-2xs font-medium text-ink-dim">{fieldLabel(change.key)}</span>
                            ) : (
                                <Checkbox
                                    className="text-xs"
                                    label={fieldLabel(change.key)}
                                    checked={ticks.ticked(change.key)}
                                    onChange={() => {
                                        ticks.onToggle(change.key);
                                    }}
                                />
                            )}
                            <Value label={said.before} fieldKey={change.key} value={change.before} />
                            <Value label={said.after} fieldKey={change.key} value={change.after} />
                        </li>
                    ))}
                </ul>
            )}
        </section>
    );
}

export interface BodyTick {
    key: string;
    ticks: Ticks;
}

export interface BlockDiffProps {
    blocks: readonly BlockChangeView[];
    empty?: string;
    tick?: BodyTick | null;
}

export function BlockDiff({ blocks, empty, tick = null }: BlockDiffProps): ReactElement | null {
    if (blocks.length === 0 && empty === undefined) {
        return null;
    }
    return (
        <section aria-label={said.body} className="flex flex-col">
            {tick === null ? (
                <SectionLabel className="px-3">{said.body}</SectionLabel>
            ) : (
                <Checkbox
                    className="px-3 text-xs"
                    label={fieldLabel(tick.key)}
                    checked={tick.ticks.ticked(tick.key)}
                    onChange={() => {
                        tick.ticks.onToggle(tick.key);
                    }}
                />
            )}
            {blocks.length === 0 ? (
                <p className="px-3 py-1.5 text-xs text-ink-dim">{empty}</p>
            ) : (
                <ol className="flex flex-col">
                    {blocks.map((block, index) => {
                        const excerpt = blockExcerpt(block.raw);
                        return (
                            <li
                                key={`${block.hash}:${String(index)}`}
                                className="flex min-w-0 flex-col gap-0.5 border-b border-inset px-3 py-1.5 last:border-b-0"
                            >
                                <div className="flex min-w-0 items-center gap-2">
                                    <span className="shrink-0 text-xs text-ink">{blockKindLabel(block.kind)}</span>
                                    <span className="ml-auto flex shrink-0 items-center gap-1">
                                        {block.kind === shortcode ? (
                                            <span title={said.renderedHint}>
                                                <StatusBadge tone="info" dot={false}>
                                                    {said.rendered}
                                                </StatusBadge>
                                            </span>
                                        ) : null}
                                        <StatusBadge tone={blockTone(block.change, block.owner)} dot={false}>
                                            {blockState(block.change, block.owner)}
                                        </StatusBadge>
                                    </span>
                                </div>
                                <p
                                    className={cx(
                                        "line-clamp-2 text-2xs",
                                        excerpt === "" ? "text-ink-faint italic" : "text-ink-dim",
                                        block.change === "removed" && "line-through",
                                    )}
                                >
                                    {excerpt === "" ? said.noText : excerpt}
                                </p>
                            </li>
                        );
                    })}
                </ol>
            )}
        </section>
    );
}

export function Declined({ keys }: { keys: readonly string[] }): ReactElement | null {
    if (keys.length === 0) {
        return null;
    }
    return (
        <dl className="grid grid-cols-[minmax(0,10rem)_1fr] gap-x-3 px-3 py-1 text-xs">
            <dt className="truncate text-ink-faint">{said.declined}</dt>
            <dd className="min-w-0 text-ink-soft">{keys.map((key) => fieldLabel(key)).join(", ")}</dd>
        </dl>
    );
}
