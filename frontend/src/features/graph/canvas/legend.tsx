import type { ReactElement, ReactNode } from "react";

import { copy } from "../../../copy/index.js";
import { entityKinds } from "../../../generated/vocab.js";
import { CallSplitIcon, cx, IconButton, LinkOffIcon, toneClasses, UnfoldLessIcon, UnfoldMoreIcon } from "../../../ui/index.js";
import { entityIcon, kindTone } from "../labels.js";

interface RowProps {
    swatch: ReactNode;
    label: string;
}

function Row({ swatch, label }: RowProps): ReactElement {
    return (
        <li className="flex items-center gap-2 text-2xs text-ink-soft">
            <span className="flex w-9 shrink-0 items-center justify-center">{swatch}</span>
            <span className="truncate">{label}</span>
        </li>
    );
}

function Pill({ className }: { className: string }): ReactElement {
    return <span className={cx("block h-3 w-8 rounded-sm border bg-inset", className)} aria-hidden={true} />;
}

function Line({ className, dashed }: { className: string; dashed?: boolean }): ReactElement {
    return (
        <svg width="36" height="8" viewBox="0 0 36 8" aria-hidden={true} className={className}>
            <path d="M1 4 C 12 4, 24 4, 35 4" fill="none" stroke="currentColor" strokeWidth="1.5" strokeDasharray={dashed ? "4 3" : undefined} />
        </svg>
    );
}

export interface LegendProps {
    open: boolean;
    proof: boolean;
    onToggle: () => void;
}

export function Legend({ open, proof, onToggle }: LegendProps): ReactElement {
    return (
        <section
            aria-label={copy.graph.legend.title}
            className="absolute bottom-3 left-3 w-56 rounded-lg border border-hairline bg-panel/90 backdrop-blur"
        >
            <header className="flex h-7 items-center justify-between pr-1 pl-3">
                <span className="text-2xs font-semibold text-ink-dim">{copy.graph.legend.title}</span>
                <IconButton
                    icon={open ? UnfoldLessIcon : UnfoldMoreIcon}
                    label={copy.graph.legend.title}
                    variant="ghost"
                    size="sm"
                    onClick={onToggle}
                />
            </header>
            {open ? (
                <div className="flex flex-col gap-3 px-3 pb-3">
                    {proof ? (
                        <ul className="flex flex-col gap-1">
                            <Row swatch={<Pill className="border-ok bg-ok-soft" />} label={copy.graph.legend.proofOk} />
                            <Row swatch={<Pill className="border-warn bg-warn-soft" />} label={copy.graph.legend.proofWarn} />
                            <Row swatch={<Pill className="border-danger bg-danger-soft" />} label={copy.graph.legend.proofDanger} />
                            <Row swatch={<Pill className="border-hairline bg-muted-soft" />} label={copy.graph.legend.proofMuted} />
                        </ul>
                    ) : null}
                    <ul className="flex flex-col gap-1">
                        {entityKinds.map((kind) => {
                            const Icon = entityIcon(kind);
                            return (
                                <Row
                                    key={kind}
                                    swatch={<Icon size={14} className={toneClasses[kindTone(kind)].ink} />}
                                    label={kind}
                                />
                            );
                        })}
                    </ul>
                    <ul className="flex flex-col gap-1">
                        <Row swatch={<Pill className="border-hairline" />} label={copy.graph.legend.nodes} />
                        <Row swatch={<Pill className="border-dashed border-danger" />} label={copy.graph.legend.noPage} />
                        <Row swatch={<Pill className="border-dashed border-info" />} label={copy.graph.legend.proposedPlacement} />
                        <Row swatch={<Pill className="border-accent bg-accent-soft" />} label={copy.graph.legend.selected} />
                        <Row
                            swatch={
                                <span className="rounded-sm bg-raised px-1 font-mono text-2xs text-ink-soft" aria-hidden={true}>
                                    ▸ 12
                                </span>
                            }
                            label={copy.graph.legend.collapsed}
                        />
                    </ul>
                    <ul className="flex flex-col gap-1">
                        <Row swatch={<Line className="text-edge" />} label={copy.graph.legend.parent} />
                        <Row swatch={<Line className="text-info" />} label={copy.graph.legend.related} />
                        <Row swatch={<Line className="text-info" dashed={true} />} label={copy.graph.legend.relatedProposed} />
                    </ul>
                    <ul className="flex flex-col gap-1">
                        <Row swatch={<LinkOffIcon size={12} className="text-danger" />} label={copy.graph.legend.noPage} />
                        <Row swatch={<CallSplitIcon size={12} className="text-ink-dim" />} label={copy.graph.legend.extraParent} />
                        <Row
                            swatch={
                                <span className="rounded-sm bg-info-soft px-1 font-mono text-2xs text-info" aria-hidden={true}>
                                    2
                                </span>
                            }
                            label={copy.graph.legend.proposedBadge}
                        />
                    </ul>
                    <p className="text-2xs text-ink-faint">{copy.graph.legend.order}</p>
                </div>
            ) : null}
        </section>
    );
}
