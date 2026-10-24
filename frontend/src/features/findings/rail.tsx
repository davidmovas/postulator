import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { FindingCounts } from "../../data/types.js";
import { cx, SectionLabel, SkeletonRows, toneClasses } from "../../ui/index.js";
import { findingCodeLabel, severityLabel, severityTone } from "../pages/model/labels.js";

const said = copy.findings;

interface CodeRowProps {
    label: string;
    count: number;
    severity: string | null;
    active: boolean;
    onSelect: () => void;
}

function CodeRow({ label, count, severity, active, onSelect }: CodeRowProps): ReactElement {
    return (
        <button
            type="button"
            aria-pressed={active}
            onClick={onSelect}
            title={severity === null ? undefined : severityLabel(severity)}
            className={cx(
                "flex h-6 w-full items-center gap-2 rounded-sm px-1.5 text-xs transition-colors duration-100",
                active ? "bg-accent-soft text-ink" : "text-ink-dim hover:bg-inset hover:text-ink",
            )}
        >
            {severity === null ? null : (
                <span aria-hidden={true} className={cx("size-1.5 shrink-0 rounded-full", toneClasses[severityTone(severity)].solid)} />
            )}
            <span className="min-w-0 flex-1 truncate text-left">{label}</span>
            <span className="shrink-0 font-mono text-2xs text-ink-faint">{count}</span>
        </button>
    );
}

export interface FindingsRailProps {
    counts: FindingCounts | undefined;
    code: string;
    onCode: (code: string) => void;
}

export function FindingsRail({ counts, code, onCode }: FindingsRailProps): ReactElement {
    const codes = counts?.codes ?? [];
    return (
        <nav aria-label={said.codes} className="flex flex-col gap-1 p-3">
            <SectionLabel>{said.codes}</SectionLabel>
            {counts === undefined ? (
                <SkeletonRows rows={6} label={said.loading} />
            ) : (
                <>
                    <CodeRow
                        label={said.all}
                        count={counts.total}
                        severity={null}
                        active={code === ""}
                        onSelect={() => {
                            onCode("");
                        }}
                    />
                    {codes.map((held) => (
                        <CodeRow
                            key={held.code}
                            label={findingCodeLabel(held.code)}
                            count={held.count}
                            severity={held.severity}
                            active={code === held.code}
                            onSelect={() => {
                                onCode(code === held.code ? "" : held.code);
                            }}
                        />
                    ))}
                </>
            )}
        </nav>
    );
}
