import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router";

import { readLastSite } from "../../../app/site-memory.js";
import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { failure, reasonOf } from "../../../data/errors.js";
import { useSetPageTemplates } from "../../../data/hooks/pages.js";
import { useSites } from "../../../data/hooks/sites.js";
import { usePageTypes, useSavePageType, useTemplates } from "../../../data/hooks/templates.js";
import { pushToast } from "../../../data/toasts.js";
import type { PageType } from "../../../data/types.js";
import {
    AddIcon,
    Banner,
    Button,
    CategoryIcon,
    cx,
    Dialog,
    EmptyState,
    Panel,
    PanelHeader,
    PublicIcon,
    Select,
    SkeletonRows,
    StatusBadge,
    toneClasses,
} from "../../../ui/index.js";
import { presetHint, presetIcon, presetTone, roleLabel } from "./labels.js";
import { orderedTypes, pickSite } from "./model.js";
import type { TemplateChoice } from "./type-dialogs.js";
import { TypeFormDialog } from "./type-dialogs.js";
import { TypeDrawer } from "./type-drawer.js";

const said = copy.pageTypes.screen;

const columns = "minmax(0,1.7fr) minmax(0,0.8fr) 4.5rem minmax(0,1.5fr)";

const templatePage = 100;

const selectionMoved = "selection_moved";

interface TypeRowProps {
    type: PageType;
    selected: boolean;
    template: string;
    suggestion: string | null;
    onOpen: () => void;
    onSuggest: () => void;
    onFollow: () => void;
}

function TypeRow({ type, selected, template, suggestion, onOpen, onSuggest, onFollow }: TypeRowProps): ReactElement {
    const Icon = presetIcon(type.preset);
    const hint = presetHint(type.preset);
    return (
        <div
            role="row"
            tabIndex={0}
            aria-label={said.open(type.name)}
            aria-selected={selected}
            data-page-type-row={type.preset}
            style={{ gridTemplateColumns: columns }}
            onClick={onOpen}
            onKeyDown={(event) => {
                if (event.key === "Enter" && event.target === event.currentTarget) {
                    event.preventDefault();
                    onOpen();
                }
            }}
            className={cx(
                "grid cursor-pointer items-start gap-3 border-b border-inset px-3 py-2.5 text-sm transition-colors duration-100",
                "focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-accent",
                selected ? "bg-accent-soft shadow-[inset_2px_0_0_var(--color-accent)]" : "hover:bg-inset",
            )}
        >
            <div role="cell" className="flex min-w-0 flex-col gap-1">
                <span className="flex min-w-0 items-center gap-1.5">
                    <Icon size={14} className={`shrink-0 ${toneClasses[presetTone(type.preset)].ink}`} />
                    <span data-type-name={true} className="truncate font-medium text-ink">
                        {type.name}
                    </span>
                </span>
                {hint === "" ? null : <span className="line-clamp-2 text-2xs leading-snug text-ink-faint">{hint}</span>}
            </div>
            <div role="cell" className="flex min-w-0 flex-wrap items-center gap-1.5 text-ink-soft">
                <span data-type-role={true}>{roleLabel(type.role)}</span>
                {type.writable ? null : (
                    <StatusBadge tone="muted" dot={false}>
                        {said.readOnly}
                    </StatusBadge>
                )}
            </div>
            <div role="cell" className={cx("text-right font-mono text-xs", type.pages === 0 ? "text-ink-faint" : "text-ink-soft")}>
                {String(type.pages)}
            </div>
            <div role="cell" className="flex min-w-0 flex-col items-start gap-1">
                <span data-type-template={true} title={template} className={cx("max-w-full truncate", type.templateId === null ? "text-ink-dim" : "text-ink-soft")}>
                    {template}
                </span>
                {suggestion === null ? null : (
                    <span data-type-suggestion={true} className="flex max-w-full flex-wrap items-center gap-x-2 text-2xs text-ink-dim">
                        <span className="truncate">{said.suggested(suggestion)}</span>
                        <Button
                            size="sm"
                            variant="ghost"
                            onClick={(event) => {
                                event.stopPropagation();
                                onSuggest();
                            }}
                        >
                            {said.useSuggested}
                        </Button>
                    </span>
                )}
                {type.overrides === 0 ? null : (
                    <span data-type-overrides={type.overrides} className="flex max-w-full flex-wrap items-center gap-x-2 text-2xs text-ink-dim">
                        <span>{said.overrides(type.overrides)}</span>
                        <Button
                            size="sm"
                            variant="ghost"
                            onClick={(event) => {
                                event.stopPropagation();
                                onFollow();
                            }}
                        >
                            {said.follow}
                        </Button>
                    </span>
                )}
            </div>
        </div>
    );
}

interface FollowDialogProps {
    siteId: string;
    type: PageType | null;
    template: string;
    onOpenChange: (open: boolean) => void;
}

function FollowDialog({ siteId, type, template, onOpenChange }: FollowDialogProps): ReactElement {
    const follow = useSetPageTemplates();
    const words = copy.pageTypes.follow;
    const name = type?.name ?? "";
    const problem = follow.error === null ? null : reasonOf(follow.error) === selectionMoved ? words.moved : failure(follow.error).message;
    return (
        <Dialog
            open={type !== null}
            onOpenChange={(next) => {
                if (!next) {
                    follow.reset();
                }
                onOpenChange(next);
            }}
            title={words.title(name)}
            description={words.body(type?.overrides ?? 0, template)}
            confirmLabel={words.confirm}
            cancelLabel={words.cancel}
            busy={follow.isPending}
            onConfirm={() => {
                if (type === null) {
                    return;
                }
                follow.mutate(
                    { selection: { siteId, filter: { pageTypeId: type.id } }, templateId: null },
                    {
                        onSuccess: (answered) => {
                            pushToast("info", answered.changed === 0 ? words.none : words.done(answered.changed));
                            onOpenChange(false);
                        },
                    },
                );
            }}
        >
            {problem === null ? null : <p className="text-xs text-danger">{problem}</p>}
        </Dialog>
    );
}

interface SiteTypesProps {
    siteId: string;
    typeId: string | null;
    onOpen: (typeId: string | null) => void;
}

function SiteTypes({ siteId, typeId, onOpen }: SiteTypesProps): ReactElement {
    const listed = usePageTypes(siteId);
    const globals = useTemplates({ scope: "global" }, null, templatePage);
    const locals = useTemplates({ siteId }, null, templatePage);
    const [creating, setCreating] = useState(false);
    const [following, setFollowing] = useState<PageType | null>(null);
    const save = useSavePageType();

    const templates: TemplateChoice[] = useMemo(
        () =>
            [...flatten(globals.data?.pages), ...flatten(locals.data?.pages)].map((held) => ({ id: held.id, name: held.name })),
        [globals.data, locals.data],
    );
    const names = useMemo(() => new Map(templates.map((held) => [held.id, held.name] as const)), [templates]);
    const types = useMemo(() => orderedTypes(listed.data?.types ?? []), [listed.data]);
    const open = types.find((held) => held.id === typeId) ?? null;

    const templateOf = (type: PageType): string =>
        type.templateId === null ? said.siteDefault : (names.get(type.templateId) ?? said.siteDefault);

    const suggestionOf = (type: PageType): string | null =>
        type.templateId !== null || type.suggestedTemplateId === null
            ? null
            : (names.get(type.suggestedTemplateId) ?? said.suggestedBuiltIn);

    const acceptSuggestion = (type: PageType): void => {
        const templateId = type.suggestedTemplateId;
        if (templateId === null) {
            return;
        }
        save.mutate(
            { id: type.id, templateId },
            {
                onSuccess: () => {
                    pushToast("info", said.usedSuggested(names.get(templateId) ?? said.suggestedBuiltIn, type.name));
                },
            },
        );
    };

    return (
        <Panel>
            <PanelHeader title={said.title}>
                <Button
                    size="sm"
                    variant="primary"
                    icon={AddIcon}
                    data-page-type-new={true}
                    onClick={() => {
                        setCreating(true);
                    }}
                >
                    {said.create}
                </Button>
            </PanelHeader>
            {listed.isPending ? (
                <SkeletonRows rows={6} label={said.loading} className="p-3" />
            ) : listed.error !== null ? (
                <div className="p-3">
                    <Banner
                        tone="danger"
                        title={failure(listed.error).message}
                        actions={
                            <Button
                                size="sm"
                                onClick={() => {
                                    void listed.refetch();
                                }}
                            >
                                {copy.app.retry}
                            </Button>
                        }
                    />
                </div>
            ) : types.length === 0 ? (
                <div className="p-3">
                    <EmptyState icon={CategoryIcon} title={said.empty} body={said.emptyBody} />
                </div>
            ) : (
                <div role="table" aria-label={said.title} className="flex min-w-0 flex-col">
                    <div
                        role="row"
                        style={{ gridTemplateColumns: columns }}
                        className="grid h-6 items-center gap-3 border-b border-hairline bg-inset px-3 text-2xs font-semibold tracking-label text-ink-faint uppercase"
                    >
                        <span role="columnheader">{said.columns.name}</span>
                        <span role="columnheader">{said.columns.preset}</span>
                        <span role="columnheader" className="text-right">
                            {said.columns.pages}
                        </span>
                        <span role="columnheader">{said.columns.template}</span>
                    </div>
                    {types.map((type) => (
                        <TypeRow
                            key={type.id}
                            type={type}
                            selected={type.id === typeId}
                            template={templateOf(type)}
                            suggestion={suggestionOf(type)}
                            onOpen={() => {
                                onOpen(type.id);
                            }}
                            onSuggest={() => {
                                acceptSuggestion(type);
                            }}
                            onFollow={() => {
                                setFollowing(type);
                            }}
                        />
                    ))}
                </div>
            )}
            <FollowDialog
                siteId={siteId}
                type={following}
                template={following === null ? said.siteDefault : templateOf(following)}
                onOpenChange={(next) => {
                    if (!next) {
                        setFollowing(null);
                    }
                }}
            />
            <TypeFormDialog
                open={creating}
                siteId={siteId}
                type={null}
                templates={templates}
                onOpenChange={setCreating}
                onCreated={(id) => {
                    onOpen(id);
                }}
            />
            {open === null ? null : (
                <TypeDrawer
                    key={open.id}
                    siteId={siteId}
                    type={open}
                    types={types}
                    catalog={listed.data?.fields ?? []}
                    templates={templates}
                    templateName={templateOf(open)}
                    onClose={() => {
                        onOpen(null);
                    }}
                />
            )}
        </Panel>
    );
}

export function PageTypesScreen(): ReactElement {
    const [searchParams, setSearchParams] = useSearchParams();
    const sites = useSites();
    const rows = flatten(sites.data?.pages);
    const siteId = pickSite(searchParams.get("site"), readLastSite(), rows.map((site) => site.id));
    const typeId = searchParams.get("type");

    const go = (site: string, type: string | null): void => {
        const next = new URLSearchParams({ site });
        if (type !== null) {
            next.set("type", type);
        }
        setSearchParams(next, { replace: true });
    };

    return (
        <div className="mx-auto flex w-full max-w-4xl flex-col gap-4">
            <div className="flex flex-wrap items-end justify-between gap-3">
                <p className="max-w-xl text-sm text-ink-soft">{said.body}</p>
                {siteId === null ? null : (
                    <div className="w-64">
                        <Select
                            aria-label={said.site}
                            value={siteId}
                            options={rows.map((site) => ({ value: site.id, label: site.name }))}
                            onValueChange={(site) => {
                                go(site, null);
                            }}
                        />
                    </div>
                )}
            </div>
            {sites.isPending ? (
                <Panel>
                    <SkeletonRows rows={4} label={said.loading} className="p-3" />
                </Panel>
            ) : siteId === null ? (
                <Panel>
                    <div className="p-3">
                        <EmptyState
                            icon={PublicIcon}
                            title={said.noSites}
                            actions={
                                <Link to="/sites">
                                    <Button size="sm">{said.openSites}</Button>
                                </Link>
                            }
                        />
                    </div>
                </Panel>
            ) : (
                <SiteTypes
                    key={siteId}
                    siteId={siteId}
                    typeId={typeId}
                    onOpen={(type) => {
                        go(siteId, type);
                    }}
                />
            )}
        </div>
    );
}
