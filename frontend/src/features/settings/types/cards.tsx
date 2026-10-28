import type { ReactElement, ReactNode } from "react";
import { useMemo, useState } from "react";
import { Link } from "react-router";

import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { failure, reasonOf } from "../../../data/errors.js";
import { useSetPageTemplates } from "../../../data/hooks/pages.js";
import { useSavePageType, useTemplates } from "../../../data/hooks/templates.js";
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
    SkeletonRows,
    StatusBadge,
    toneClasses,
} from "../../../ui/index.js";
import { rememberPages } from "../../pages/views/memory.js";
import type { Names } from "../classification/model.js";
import { sentence } from "../classification/model.js";
import type { RuleNaming } from "../classification/naming.js";
import { presetHint, presetIcon, presetTone } from "./labels.js";
import type { RulesGiving } from "./summary.js";
import { rulesGiving, typeGroups, writtenSummary } from "./summary.js";
import type { TemplateChoice } from "./type-dialogs.js";
import { TypeFormDialog } from "./type-dialogs.js";
import { TypeDrawer } from "./type-drawer.js";

const said = copy.pageTypes.screen;

const templatePage = 100;

const selectionMoved = "selection_moved";

function quiet(run: () => void): (event: { stopPropagation: () => void }) => void {
    return (event) => {
        event.stopPropagation();
        run();
    };
}

function InlineAction({ label, onClick }: { label: string; onClick: () => void }): ReactElement {
    return (
        <button
            type="button"
            onClick={quiet(onClick)}
            className="rounded-sm text-left font-medium text-accent hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
        >
            {label}
        </button>
    );
}

function Fact({ label, children }: { label: string; children: ReactNode }): ReactElement {
    return (
        <div className="flex flex-col gap-1">
            <dt className="text-2xs font-medium text-ink-faint">{label}</dt>
            <dd className="text-xs leading-snug text-ink-soft">{children}</dd>
        </div>
    );
}

function Giving({ giving, names }: { giving: RulesGiving; names: Names }): ReactElement {
    if (giving.rules.length === 0 && !giving.fallback) {
        return <p className="text-ink-faint">{said.noRule}</p>;
    }
    return (
        <ul className="flex flex-col gap-1">
            {giving.rules.map((rule) => {
                const text = sentence(rule.conditions, names);
                return (
                    <li key={rule.id} data-card-rule={rule.id} className="flex items-start gap-2">
                        <span
                            aria-hidden={true}
                            className="mt-px inline-flex h-4 min-w-4 shrink-0 items-center justify-center rounded-full border border-edge px-1 font-mono text-[0.625rem] leading-none text-ink-dim"
                        >
                            {rule.number}
                        </span>
                        <span title={text} className="line-clamp-2">
                            {text}
                        </span>
                    </li>
                );
            })}
            {giving.fallback ? (
                <li data-card-fallback={true} className="flex items-start gap-2">
                    <span aria-hidden={true} className="mt-1 ml-0.5 inline-flex h-2.5 w-2.5 shrink-0 rounded-full border-2 border-edge" />
                    <span>{said.anythingElse}</span>
                </li>
            ) : null}
        </ul>
    );
}

interface TypeCardProps {
    siteId: string;
    type: PageType;
    giving: RulesGiving;
    names: Names;
    selected: boolean;
    template: string;
    suggestion: string | null;
    onOpen: () => void;
    onSuggest: () => void;
    onFollow: () => void;
}

function TypeCard({ siteId, type, giving, names, selected, template, suggestion, onOpen, onSuggest, onFollow }: TypeCardProps): ReactElement {
    const Icon = presetIcon(type.preset);
    const hint = presetHint(type.preset);
    return (
        <article
            aria-label={type.name}
            data-type-card={type.preset}
            data-type-id={type.id}
            onClick={onOpen}
            className={cx(
                "flex min-w-0 cursor-pointer flex-col gap-3 rounded-lg border bg-panel p-3 transition-colors duration-100",
                selected ? "border-accent-border bg-accent-soft" : "border-hairline hover:border-edge",
            )}
        >
            <header className="flex items-start gap-2">
                <Icon size={16} className={cx("mt-0.5 shrink-0", toneClasses[presetTone(type.preset)].ink)} />
                <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                    <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
                        <button
                            type="button"
                            aria-label={said.open(type.name)}
                            onClick={quiet(onOpen)}
                            className="min-w-0 truncate rounded-sm text-left text-sm font-semibold text-ink hover:text-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
                        >
                            {type.name}
                        </button>
                        {type.writable ? null : (
                            <StatusBadge tone="muted" dot={false}>
                                {said.readOnly}
                            </StatusBadge>
                        )}
                    </span>
                    {hint === "" ? null : <p className="text-xs leading-snug text-ink-dim">{hint}</p>}
                </div>
                <Link
                    to={`/s/${siteId}/pages`}
                    aria-label={said.showPages(type.name)}
                    onClick={quiet(() => {
                        rememberPages(siteId, { view: "types", openType: type.id });
                    })}
                    className={cx(
                        "shrink-0 rounded-sm font-mono text-xs hover:text-accent hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent",
                        type.pages === 0 ? "text-ink-faint" : "text-ink-soft",
                    )}
                >
                    {said.pages(type.pages)}
                </Link>
            </header>
            <dl className="flex flex-col gap-2.5 border-t border-inset pt-2.5">
                <Fact label={said.gets}>
                    <Giving giving={giving} names={names} />
                </Fact>
                {type.writable ? (
                    <>
                        <Fact label={said.writes}>{writtenSummary(type.scopes ?? [])}</Fact>
                        <Fact label={said.template}>
                            <span className="flex flex-col items-start gap-1">
                                <span data-type-template={true} className={cx("max-w-full truncate", type.templateId === null && "text-ink-dim")}>
                                    {template}
                                </span>
                                {suggestion === null ? null : (
                                    <span data-type-suggestion={true} className="flex flex-wrap items-baseline gap-x-2 text-2xs text-ink-dim">
                                        <span>{said.suggested(suggestion)}</span>
                                        <InlineAction label={said.useSuggested} onClick={onSuggest} />
                                    </span>
                                )}
                                {type.overrides === 0 ? null : (
                                    <span data-type-overrides={type.overrides} className="flex flex-wrap items-baseline gap-x-2 text-2xs text-ink-dim">
                                        <span>{said.overrides(type.overrides)}</span>
                                        <InlineAction label={said.follow} onClick={onFollow} />
                                    </span>
                                )}
                            </span>
                        </Fact>
                    </>
                ) : null}
            </dl>
        </article>
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

const cardGrid = "grid grid-cols-[repeat(auto-fill,minmax(min(100%,18rem),1fr))] gap-3";

interface CardGroupProps {
    title: string;
    body?: string;
    action?: ReactNode;
    children: ReactNode;
}

function CardGroup({ title, body, action, children }: CardGroupProps): ReactElement {
    return (
        <section aria-label={title} className="flex flex-col gap-2.5">
            <div className="flex flex-wrap items-end justify-between gap-2">
                <div className="flex min-w-0 flex-col gap-0.5">
                    <h2 className="text-sm font-semibold text-ink">{title}</h2>
                    {body === undefined ? null : <p className="max-w-2xl text-xs text-ink-dim">{body}</p>}
                </div>
                {action}
            </div>
            <div className={cardGrid}>{children}</div>
        </section>
    );
}

export interface TypesViewProps {
    siteId: string;
    naming: RuleNaming;
    typeId: string | null;
    onOpen: (typeId: string | null) => void;
}

export function TypesView({ siteId, naming, typeId, onOpen }: TypesViewProps): ReactElement {
    const { typed, types, rules, fallback, names } = naming;
    const globals = useTemplates({ scope: "global" }, null, templatePage);
    const locals = useTemplates({ siteId }, null, templatePage);
    const save = useSavePageType();
    const [creating, setCreating] = useState(false);
    const [following, setFollowing] = useState<PageType | null>(null);

    const templates: TemplateChoice[] = useMemo(
        () =>
            [...flatten(globals.data?.pages), ...flatten(locals.data?.pages)].map((held) => ({ id: held.id, name: held.name })),
        [globals.data, locals.data],
    );
    const templateNames = useMemo(() => new Map(templates.map((held) => [held.id, held.name] as const)), [templates]);
    const groups = useMemo(() => typeGroups(types), [types]);
    const open = types.find((held) => held.id === typeId) ?? null;

    const templateOf = (type: PageType): string =>
        type.templateId === null ? said.siteDefault : (templateNames.get(type.templateId) ?? said.siteDefault);

    const suggestionOf = (type: PageType): string | null =>
        type.templateId !== null || type.suggestedTemplateId === null
            ? null
            : (templateNames.get(type.suggestedTemplateId) ?? said.suggestedBuiltIn);

    const acceptSuggestion = (type: PageType): void => {
        const suggested = type.suggestedTemplateId;
        if (suggested === null) {
            return;
        }
        save.mutate(
            { id: type.id, templateId: suggested },
            {
                onSuccess: () => {
                    pushToast("info", said.usedSuggested(templateNames.get(suggested) ?? said.suggestedBuiltIn, type.name));
                },
            },
        );
    };

    const card = (type: PageType): ReactElement => (
        <TypeCard
            key={type.id}
            siteId={siteId}
            type={type}
            giving={rulesGiving(type.id, rules, fallback)}
            names={names}
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
    );

    const create = (
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
    );

    return (
        <div data-types-view={true} className="flex flex-col gap-5">
            {typed.isPending ? (
                <SkeletonRows rows={6} label={said.loading} />
            ) : typed.error !== null ? (
                <Banner
                    tone="danger"
                    title={failure(typed.error).message}
                    actions={
                        <Button
                            size="sm"
                            onClick={() => {
                                void typed.refetch();
                            }}
                        >
                            {copy.app.retry}
                        </Button>
                    }
                />
            ) : types.length === 0 ? (
                <EmptyState icon={CategoryIcon} title={said.empty} body={said.emptyBody} actions={create} />
            ) : (
                <>
                    <CardGroup title={said.written} action={create}>
                        {groups.written.map(card)}
                    </CardGroup>
                    {groups.never.length === 0 ? null : (
                        <CardGroup title={said.never} body={said.neverBody}>
                            {groups.never.map(card)}
                        </CardGroup>
                    )}
                </>
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
                    catalog={typed.data?.fields ?? []}
                    templates={templates}
                    templateName={templateOf(open)}
                    onClose={() => {
                        onOpen(null);
                    }}
                />
            )}
        </div>
    );
}
