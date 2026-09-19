import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router";

import { failure } from "../../data/errors.js";
import { usePage } from "../../data/hooks/pages.js";
import { useDeleteOverride, useResolvedTemplate } from "../../data/hooks/templates.js";
import type { TemplateOverride } from "../../data/types.js";
import { copy } from "../../copy/index.js";
import type { JsonObject } from "../../domain/merge-patch.js";
import {
    Button,
    DeleteIcon,
    Dialog,
    Drawer,
    EditIcon,
    Panel,
    PanelHeader,
    SectionLabel,
    SkeletonRows,
    VisibilityIcon,
} from "../../ui/index.js";
import { patchObject } from "./patch.js";
import { sentencesOf } from "./sentences.js";
import { SpecView } from "./spec-view.js";
import type { SpecDraft } from "./spec.js";
import { draftOf } from "./spec.js";

interface SentenceListProps {
    prefix: string;
    sentences: readonly string[];
}

function SentenceList({ prefix, sentences }: SentenceListProps): ReactElement {
    return (
        <ul className="flex flex-col gap-1">
            {sentences.map((sentence) => (
                <li key={sentence} className="flex gap-1.5 text-xs text-ink-soft">
                    <span aria-hidden={true} className="mt-1.5 h-1 w-1 shrink-0 rounded-full bg-accent" />
                    <span>
                        <span className="font-semibold text-ink">{prefix}</span> {sentence}.
                    </span>
                </li>
            ))}
        </ul>
    );
}

interface ResolvedDrawerProps {
    pageId: string;
    title: string;
    site: JsonObject | null;
    page: JsonObject | null;
    onClose: () => void;
}

function ResolvedDrawer({ pageId, title, site, page, onClose }: ResolvedDrawerProps): ReactElement {
    const resolved = useResolvedTemplate(pageId);
    const spec = resolved.data?.spec;
    const draft = useMemo(() => (spec === undefined ? null : draftOf(spec)), [spec]);

    return (
        <Drawer
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={title}
            description={copy.templates.overrides.resolvedBody}
            closeLabel={copy.templates.overrides.close}
            width={520}
        >
            <div className="p-3">
                {resolved.isPending ? (
                    <SkeletonRows rows={10} label={copy.app.loading} />
                ) : draft === null ? (
                    <p className="text-sm text-ink-soft">
                        {failure(resolved.error).code === "NOT_FOUND"
                            ? copy.templates.overrides.resolveFailed
                            : failure(resolved.error).message}
                    </p>
                ) : (
                    <SpecView draft={draft} site={site} page={page} />
                )}
            </div>
        </Drawer>
    );
}

interface PageOverrideRowProps {
    override: TemplateOverride;
    base: SpecDraft;
    site: JsonObject | null;
    onRemove: (id: string) => void;
    removing: boolean;
}

function PageOverrideRow({ override, base, site, onRemove, removing }: PageOverrideRowProps): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const detail = usePage(override.targetId);
    const [open, setOpen] = useState(false);
    const patch = useMemo(() => patchObject(override.patch), [override.patch]);
    const sentences = useMemo(() => sentencesOf(base, patch), [base, patch]);
    const path = detail.data?.page.path ?? copy.templates.overrides.unknownPage;

    return (
        <div className="flex flex-col gap-1.5 rounded-md border border-hairline bg-inset p-2.5">
            <div className="flex items-start justify-between gap-2">
                <span className="min-w-0 truncate font-mono text-xs text-ink">{path}</span>
                <div className="flex shrink-0 gap-1">
                    <Button
                        size="sm"
                        variant="ghost"
                        icon={EditIcon}
                        onClick={() => {
                            void navigate(`/s/${params.siteId ?? ""}/templates/${override.templateId}?page=${override.targetId}`);
                        }}
                    >
                        {copy.pages.detail.templateEdit}
                    </Button>
                    <Button
                        size="sm"
                        variant="ghost"
                        icon={VisibilityIcon}
                        onClick={() => {
                            setOpen(true);
                        }}
                    >
                        {copy.templates.overrides.viewResolved}
                    </Button>
                    <Button
                        size="sm"
                        variant="ghost"
                        icon={DeleteIcon}
                        busy={removing}
                        onClick={() => {
                            onRemove(override.id);
                        }}
                    >
                        {copy.templates.overrides.clearPage}
                    </Button>
                </div>
            </div>
            <SentenceList prefix={copy.templates.overrides.pagePrefix} sentences={sentences} />
            {open ? (
                <ResolvedDrawer
                    pageId={override.targetId}
                    title={path}
                    site={site}
                    page={patch}
                    onClose={() => {
                        setOpen(false);
                    }}
                />
            ) : null}
        </div>
    );
}

export interface OverridesPanelProps {
    base: SpecDraft;
    siteOverride: TemplateOverride | null;
    pageOverrides: readonly TemplateOverride[];
}

export function OverridesPanel({ base, siteOverride, pageOverrides }: OverridesPanelProps): ReactElement {
    const remove = useDeleteOverride();
    const [clearing, setClearing] = useState(false);
    const sitePatch = useMemo(() => patchObject(siteOverride?.patch), [siteOverride]);
    const siteSentences = useMemo(() => sentencesOf(base, sitePatch), [base, sitePatch]);

    return (
        <Panel className="min-w-0">
            <PanelHeader title={copy.templates.overrides.title} />
            <div className="flex flex-col gap-3 p-3">
                <p className="text-xs text-ink-dim">{copy.templates.overrides.base}</p>
                <p className="text-2xs text-ink-faint">{copy.templates.overrides.runOrder}</p>
                <div className="flex flex-col gap-2 border-t border-hairline pt-3">
                    <SectionLabel>{copy.templates.overrides.siteTitle}</SectionLabel>
                    {siteSentences.length === 0 ? (
                        <p className="text-xs text-ink-faint">{copy.templates.overrides.siteNone}</p>
                    ) : (
                        <>
                            <SentenceList prefix={copy.templates.overrides.sitePrefix} sentences={siteSentences} />
                            <Button
                                size="sm"
                                variant="ghost"
                                icon={DeleteIcon}
                                className="self-start"
                                onClick={() => {
                                    setClearing(true);
                                }}
                            >
                                {copy.templates.overrides.clearSite}
                            </Button>
                        </>
                    )}
                </div>
                <div className="flex flex-col gap-2 border-t border-hairline pt-3">
                    <div className="flex items-center justify-between gap-2">
                        <SectionLabel>{copy.templates.overrides.pagesTitle}</SectionLabel>
                        {pageOverrides.length === 0 ? null : (
                            <span className="font-mono text-2xs text-ink-faint">
                                {copy.templates.overrides.pageCount(pageOverrides.length)}
                            </span>
                        )}
                    </div>
                    {pageOverrides.length === 0 ? (
                        <p className="text-xs text-ink-faint">{copy.templates.overrides.pagesNone}</p>
                    ) : (
                        pageOverrides.map((override) => (
                            <PageOverrideRow
                                key={override.id}
                                override={override}
                                base={base}
                                site={sitePatch}
                                removing={remove.isPending && remove.variables?.id === override.id}
                                onRemove={(id) => {
                                    remove.mutate({ id });
                                }}
                            />
                        ))
                    )}
                </div>
            </div>
            <Dialog
                open={clearing}
                onOpenChange={setClearing}
                title={copy.templates.overrides.clearSiteTitle}
                description={copy.templates.overrides.clearSiteBody}
                confirmLabel={copy.templates.overrides.clearConfirm}
                cancelLabel={copy.templates.overrides.cancel}
                destructive={true}
                icon={DeleteIcon}
                busy={remove.isPending}
                onConfirm={() => {
                    if (siteOverride === null) {
                        setClearing(false);
                        return;
                    }
                    remove.mutate(
                        { id: siteOverride.id },
                        {
                            onSuccess: () => {
                                setClearing(false);
                            },
                        },
                    );
                }}
            />
        </Panel>
    );
}
