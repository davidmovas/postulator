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
            <span className="min-w-0 flex-1">{label}</span>
        </li>
    );
}

function Group({ title, children }: { title: string; children: ReactNode }): ReactElement {
    return (
        <div className="flex flex-col gap-1">
            <span className="text-2xs tracking-label text-ink-faint uppercase">{title}</span>
            <ul className="flex flex-col gap-1">{children}</ul>
        </div>
    );
}

function Pill({ className }: { className: string }): ReactElement {
    return <span className={cx("block h-3 w-8 rounded-sm border bg-inset", className)} aria-hidden={true} />;
}

function Line({ className, dashed }: { className: string; dashed?: boolean }): ReactElement {
    return (
        <svg width="36" height="8" viewBox="0 0 36 8" aria-hidden={true} className={className}>
            <path
                d="M1 4 C 12 4, 24 4, 35 4"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeDasharray={dashed ? "4 3" : undefined}
            />
        </svg>
    );
}

export interface LegendProps {
    open: boolean;
    proof: boolean;
    onToggle: () => void;
}

export function Legend({ open, proof, onToggle }: LegendProps): ReactElement {
    const said = copy.graph.legend;

    return (
        <section
            aria-label={said.title}
            className="absolute bottom-3 left-3 max-h-[70%] w-60 overflow-y-auto rounded-lg border border-hairline bg-panel/90 backdrop-blur"
        >
            <header className="sticky top-0 flex h-7 items-center justify-between bg-panel/90 pr-1 pl-3 backdrop-blur">
                <span className="text-2xs font-semibold text-ink-dim">{said.title}</span>
                <IconButton
                    icon={open ? UnfoldLessIcon : UnfoldMoreIcon}
                    label={open ? said.hide : said.show}
                    variant="ghost"
                    size="sm"
                    onClick={onToggle}
                />
            </header>
            {open ? (
                <div className="flex flex-col gap-3 px-3 pb-3">
                    {proof ? (
                        <Group title={said.proof}>
                            <Row swatch={<Pill className="border-ok bg-ok-soft" />} label={said.proofOk} />
                            <Row swatch={<Pill className="border-warn bg-warn-soft" />} label={said.proofWarn} />
                            <Row swatch={<Pill className="border-danger bg-danger-soft" />} label={said.proofDanger} />
                            <Row swatch={<Pill className="border-hairline bg-muted-soft" />} label={said.proofMuted} />
                        </Group>
                    ) : null}
                    <Group title={said.kinds}>
                        {entityKinds.map((kind) => {
                            const Icon = entityIcon(kind);
                            return (
                                <Row
                                    key={kind}
                                    swatch={<Icon size={14} className={toneClasses[kindTone(kind)].ink} />}
                                    label={said.kind(kind)}
                                />
                            );
                        })}
                    </Group>
                    <Group title={said.states}>
                        <Row swatch={<Pill className="border-danger bg-danger-soft" />} label={said.state.mismatch} />
                        <Row swatch={<Pill className="border-accent bg-accent-soft" />} label={said.state.working} />
                        <Row swatch={<Pill className="border-ok bg-ok-soft" />} label={said.state.published} />
                        <Row swatch={<Pill className="border-warn bg-warn-soft" />} label={said.state.exists} />
                        <Row swatch={<Pill className="border-info bg-info-soft" />} label={said.state.planned} />
                        <Row swatch={<Pill className="border-muted bg-muted-soft" />} label={said.state.archived} />
                        <Row swatch={<Pill className="border-hairline" />} label={said.state.noPage} />
                    </Group>
                    <Group title={said.nodes}>
                        <Row swatch={<Pill className="border-hairline" />} label={said.plain} />
                        <Row swatch={<Pill className="border-dashed border-danger" />} label={said.noPage} />
                        <Row swatch={<Pill className="border-dashed border-info" />} label={said.proposedPlacement} />
                        <Row swatch={<Pill className="border-accent bg-accent-soft" />} label={said.selected} />
                        <Row swatch={<Pill className="border-info" />} label={said.neighbour} />
                        <Row
                            swatch={
                                <span className="rounded-sm bg-raised px-1 font-mono text-2xs text-ink-soft" aria-hidden={true}>
                                    ▸ 12
                                </span>
                            }
                            label={said.collapsed}
                        />
                    </Group>
                    <Group title={said.edges}>
                        <Row swatch={<Line className="text-edge" />} label={said.parent} />
                        <Row swatch={<Line className="text-info" dashed={true} />} label={said.parentProposed} />
                        <Row swatch={<Line className="text-info" />} label={said.related} />
                        <Row swatch={<Line className="text-info" dashed={true} />} label={said.relatedProposed} />
                    </Group>
                    <Group title={said.badges}>
                        <Row swatch={<LinkOffIcon size={12} className="text-danger" />} label={said.noPageBadge} />
                        <Row swatch={<CallSplitIcon size={12} className="text-ink-dim" />} label={said.extraParent} />
                        <Row
                            swatch={
                                <span className="rounded-sm bg-info-soft px-1 font-mono text-2xs text-info" aria-hidden={true}>
                                    2
                                </span>
                            }
                            label={said.proposedBadge}
                        />
                    </Group>
                    <p className="text-2xs text-ink-faint">{said.order}</p>
                </div>
            ) : null}
        </section>
    );
}
