import type { ReactElement } from "react";

import type { IconComponent } from "../../../ui/index.js";
import {
    AdsClickIcon,
    AttachFileIcon,
    CalculateIcon,
    cx,
    EditNoteIcon,
    KeyIcon,
    LightbulbIcon,
    LinkIcon,
    PlayArrowIcon,
    ScheduleIcon,
    TableRowsIcon,
    toneClasses,
    TuneIcon,
    WarningIcon,
} from "../../../ui/index.js";
import type { Line, LineIcon, Part } from "./model/card.js";
import { RefLabel } from "./ref.js";

const icons: Readonly<Record<LineIcon, IconComponent>> = {
    target: AdsClickIcon,
    field: EditNoteIcon,
    list: TableRowsIcon,
    money: CalculateIcon,
    warn: WarningIcon,
    link: LinkIcon,
    key: KeyIcon,
    file: AttachFileIcon,
    clock: ScheduleIcon,
    step: PlayArrowIcon,
    rule: TuneIcon,
    note: LightbulbIcon,
};

export function Parts({ parts }: { parts: readonly Part[] }): ReactElement {
    return (
        <>
            {parts.map((part, index) =>
                typeof part === "string" ? (
                    <span key={index}>{part}</span>
                ) : (
                    <RefLabel key={`${part.kind}:${part.id}:${index}`} kind={part.kind} id={part.id} />
                ),
            )}
        </>
    );
}

export function Lines({ lines }: { lines: readonly Line[] }): ReactElement | null {
    if (lines.length === 0) {
        return null;
    }
    return (
        <ul className="flex flex-col gap-1">
            {lines.map((line, index) => {
                const Icon = icons[line.icon];
                const classes = toneClasses[line.tone];
                return (
                    <li key={index} className="flex items-start gap-2 text-xs leading-relaxed text-ink-soft">
                        <Icon size={14} className={cx("mt-0.5 shrink-0", line.tone === "muted" ? "text-ink-faint" : classes.ink)} />
                        <span className={cx("min-w-0 break-words", line.tone !== "muted" && classes.ink)}>
                            <Parts parts={line.parts} />
                        </span>
                    </li>
                );
            })}
        </ul>
    );
}
