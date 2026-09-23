import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { useDeleteOverride } from "../../../data/hooks/templates.js";
import type { TemplateOverride } from "../../../data/types.js";
import { Banner, Button, Dialog, EmptyState, TuneIcon, SectionLabel } from "../../../ui/index.js";
import { diffRows } from "../diff.js";
import type { Layer } from "../patch.js";
import type { SpecDraft } from "../spec.js";

interface ClearProps {
    label: string;
    title: string;
    body: string;
    busy: boolean;
    onConfirm: () => void;
}

function Clear({ label, title, body, busy, onConfirm }: ClearProps): ReactElement {
    const [open, setOpen] = useState(false);
    return (
        <>
            <Button
                variant="danger"
                onClick={() => {
                    setOpen(true);
                }}
            >
                {label}
            </Button>
            <Dialog
                open={open}
                onOpenChange={setOpen}
                title={title}
                description={body}
                confirmLabel={copy.templates.overrides.clearConfirm}
                cancelLabel={copy.templates.overrides.cancel}
                destructive={true}
                busy={busy}
                onConfirm={() => {
                    setOpen(false);
                    onConfirm();
                }}
            />
        </>
    );
}

export interface OverridesGroupProps {
    layer: Layer;
    below: SpecDraft;
    draft: SpecDraft;
    siteOverride: TemplateOverride | null;
    pageOverrides: readonly TemplateOverride[];
    onChange: (patch: Partial<SpecDraft>) => void;
}

export function OverridesGroup({
    layer,
    below,
    draft,
    siteOverride,
    pageOverrides,
    onChange,
}: OverridesGroupProps): ReactElement {
    const drop = useDeleteOverride();
    const rows = diffRows(below, draft);

    if (layer === "global") {
        return (
            <div className="flex flex-col gap-3">
                <Banner
                    tone="info"
                    title={copy.templates.overrides.globalTitle}
                    body={copy.templates.overrides.globalBody}
                />
                <section className="flex flex-col gap-2">
                    <SectionLabel>{copy.templates.overrides.siteTitle}</SectionLabel>
                    {siteOverride === null ? (
                        <p className="text-xs text-ink-faint">{copy.templates.overrides.siteNone}</p>
                    ) : (
                        <Clear
                            label={copy.templates.overrides.clearSite}
                            title={copy.templates.overrides.clearSiteTitle}
                            body={copy.templates.overrides.clearSiteBody}
                            busy={drop.isPending}
                            onConfirm={() => {
                                drop.mutate({ id: siteOverride.id });
                            }}
                        />
                    )}
                </section>
                <section className="flex flex-col gap-2">
                    <SectionLabel>{copy.templates.overrides.pagesTitle}</SectionLabel>
                    <p className="text-xs text-ink-dim">
                        {pageOverrides.length === 0
                            ? copy.templates.overrides.pagesNone
                            : copy.templates.overrides.pageCount(pageOverrides.length)}
                    </p>
                </section>
            </div>
        );
    }

    if (rows.length === 0) {
        return (
            <EmptyState
                icon={TuneIcon}
                title={layer === "site" ? copy.templates.overrides.siteNone : copy.templates.overrides.pageNone}
            />
        );
    }

    return (
        <section className="overflow-hidden rounded-lg border border-hairline bg-panel">
            <header className="grid h-7 grid-cols-[minmax(0,1fr)_160px_160px_74px] items-center gap-3 border-b border-hairline bg-inset px-3 text-2xs font-semibold tracking-label text-ink-faint uppercase">
                <div>{copy.templates.overrides.columns.field}</div>
                <div>{copy.templates.overrides.columns.below}</div>
                <div>{copy.templates.overrides.columns.here}</div>
                <div />
            </header>
            {rows.map((row) => (
                <div
                    key={row.key}
                    className="grid h-9 grid-cols-[minmax(0,1fr)_160px_160px_74px] items-center gap-3 border-b border-inset px-3 shadow-[inset_2px_0_0_var(--color-accent)] last:border-b-0"
                >
                    <span className="min-w-0 truncate text-xs text-ink-soft">{row.label}</span>
                    <span className="min-w-0 truncate font-mono text-2xs text-ink-faint line-through">
                        {row.below}
                    </span>
                    <span className="min-w-0 truncate font-mono text-2xs text-accent">{row.here}</span>
                    <Button
                        size="sm"
                        onClick={() => {
                            onChange(row.revert);
                        }}
                    >
                        {copy.templates.overrides.revert}
                    </Button>
                </div>
            ))}
        </section>
    );
}
