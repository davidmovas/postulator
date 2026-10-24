import type { ReactElement, ReactNode } from "react";

import type { Tone } from "../../../ui/index.js";
import { cx, toneClasses } from "../../../ui/index.js";
import { familyOf } from "../conversation/model/tools.js";
import { familyIcon, familyLabel } from "../labels.js";

export interface CardBodyProps {
    tool: string;
    tone: Tone | null;
    title: ReactNode;
    lines?: ReactNode;
    summary?: string | null;
}

export function CardBody({ tool, tone, title, lines, summary = null }: CardBodyProps): ReactElement {
    const family = familyOf(tool);
    const FamilyIcon = familyIcon(family);
    return (
        <>
            <div className="flex items-start gap-2">
                <FamilyIcon size={16} className={cx("mt-0.5 shrink-0", tone === null ? "text-ink-faint" : toneClasses[tone].ink)} />
                <div className="flex min-w-0 flex-col">
                    <span className="text-2xs font-semibold tracking-label text-ink-faint uppercase">{familyLabel(family)}</span>
                    <span className="text-sm leading-snug font-semibold text-ink">{title}</span>
                </div>
            </div>
            {lines === undefined ? null : <div className="flex flex-col gap-1.5">{lines}</div>}
            {summary === null ? null : (
                <p className="border-l-2 border-hairline pl-2.5 text-xs leading-snug text-ink-soft">{summary}</p>
            )}
        </>
    );
}
