import type { ReactElement } from "react";
import { useCallback, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router";

import { readLastSite } from "../../../app/site-memory.js";
import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { failure } from "../../../data/errors.js";
import { useReorderRules, useRules, useSaveRule } from "../../../data/hooks/classification.js";
import { usePagePaths } from "../../../data/hooks/pages.js";
import { useSites } from "../../../data/hooks/sites.js";
import { usePageTypes } from "../../../data/hooks/templates.js";
import { pushToast } from "../../../data/toasts.js";
import type { PageType, Rule } from "../../../data/types.js";
import {
    AddIcon,
    ArrowDownwardIcon,
    ArrowUpwardIcon,
    Banner,
    Button,
    cx,
    EmptyState,
    IconButton,
    Panel,
    PanelHeader,
    PreviewIcon,
    PublicIcon,
    Select,
    SkeletonRows,
    StatusBadge,
    Switch,
    toneClasses,
} from "../../../ui/index.js";
import { presetIcon, presetTone } from "../types/labels.js";
import { orderedTypes, pickSite } from "../types/model.js";
import type { Names } from "./model.js";
import { reordered, sentence } from "./model.js";
import { ClassificationPreview } from "./preview.js";
import { RuleDrawer } from "./rule-drawer.js";

const said = copy.classification;

interface RuleRowProps {
    rule: Rule;
    first: boolean;
    last: boolean;
    names: Names;
    type: PageType | undefined;
    busy: boolean;
    onOpen: () => void;
    onEnable: (enabled: boolean) => void;
    onMove: (delta: -1 | 1) => void;
}

function TypeMark({ type, fallback }: { type: PageType | undefined; fallback: string }): ReactElement {
    if (type === undefined) {
        return <span className="text-ink-faint">{fallback}</span>;
    }
    const Icon = presetIcon(type.preset);
    return (
        <span className="inline-flex min-w-0 items-center gap-1 text-ink-soft">
            <Icon size={13} className={cx("shrink-0", toneClasses[presetTone(type.preset)].ink)} />
            <span className="truncate">{type.name}</span>
        </span>
    );
}

function RuleRow({ rule, first, last, names, type, busy, onOpen, onEnable, onMove }: RuleRowProps): ReactElement {
    const name = rule.name === "" ? said.unnamed : rule.name;
    return (
        <li
            data-rule-row={rule.id}
            data-rule-enabled={rule.enabled}
            className="relative grid grid-cols-[1.5rem_minmax(0,1fr)_auto] items-start gap-3 border-b border-inset px-3 py-2.5"
        >
            <span
                aria-hidden={true}
                className={cx(
                    "relative z-[1] mt-px inline-flex h-6 w-6 items-center justify-center rounded-full border bg-panel font-mono text-2xs",
                    rule.enabled ? "border-edge text-ink-soft" : "border-dashed border-hairline text-ink-faint",
                )}
            >
                {rule.number}
            </span>
            <button
                type="button"
                aria-label={said.open(name)}
                onClick={onOpen}
                className={cx(
                    "flex min-w-0 flex-col items-start gap-0.5 rounded-sm text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent",
                    !rule.enabled && "opacity-60",
                )}
            >
                <span className="flex min-w-0 flex-wrap items-center gap-x-2 text-sm">
                    <span className="font-medium text-ink hover:text-accent">{name}</span>
                    <span className="inline-flex min-w-0 items-center gap-1 text-xs text-ink-dim">
                        {said.gives}
                        <TypeMark type={type} fallback={said.lostType} />
                    </span>
                    {rule.enabled ? null : (
                        <StatusBadge tone="muted" dot={false}>
                            {said.off}
                        </StatusBadge>
                    )}
                </span>
                <span data-rule-sentence={true} className="text-xs leading-snug text-ink-dim">
                    {sentence(rule.conditions, names)}
                </span>
            </button>
            <span className="flex items-center gap-1 pt-0.5">
                <Switch
                    aria-label={said.enable(name)}
                    checked={rule.enabled}
                    disabled={busy}
                    onChange={(event) => {
                        onEnable(event.target.checked);
                    }}
                />
                <IconButton size="sm" variant="ghost" icon={ArrowUpwardIcon} label={said.moveUp(name)} disabled={first || busy} onClick={() => onMove(-1)} />
                <IconButton size="sm" variant="ghost" icon={ArrowDownwardIcon} label={said.moveDown(name)} disabled={last || busy} onClick={() => onMove(1)} />
            </span>
        </li>
    );
}

interface FallbackRowProps {
    rule: Rule;
    types: readonly PageType[];
    busy: boolean;
    onType: (typeId: string) => void;
}

function FallbackRow({ rule, types, busy, onType }: FallbackRowProps): ReactElement {
    return (
        <li data-rule-fallback={rule.id} className="relative grid grid-cols-[1.5rem_minmax(0,1fr)_auto] items-start gap-3 px-3 py-2.5">
            <span aria-hidden={true} className="relative z-[1] mt-1.5 ml-1.5 inline-flex h-3 w-3 rounded-full border-2 border-edge bg-panel" />
            <span className="flex min-w-0 flex-col gap-0.5">
                <span className="text-sm font-medium text-ink">{said.fallback.name}</span>
                <span className="text-xs leading-snug text-ink-dim">{said.fallback.body}</span>
            </span>
            <span className="w-48">
                <Select
                    size="sm"
                    aria-label={said.fallback.type}
                    disabled={busy}
                    value={rule.typeId === "" ? null : rule.typeId}
                    options={types.map((type) => ({ value: type.id, label: type.name }))}
                    onValueChange={onType}
                />
            </span>
        </li>
    );
}

function SiteRules({ siteId }: { siteId: string }): ReactElement {
    const listed = useRules(siteId);
    const typed = usePageTypes(siteId);
    const save = useSaveRule();
    const reorder = useReorderRules();
    const [editing, setEditing] = useState<Rule | null>(null);
    const [creating, setCreating] = useState(false);
    const [previewing, setPreviewing] = useState(false);
    const rules = useMemo(() => listed.data?.rules ?? [], [listed.data]);
    const fallback = listed.data?.fallback ?? null;
    const types = useMemo(() => orderedTypes(typed.data?.types ?? []), [typed.data]);
    const byType = useMemo(() => new Map(types.map((type) => [type.id, type] as const)), [types]);
    const underIds = useMemo(
        () => [...new Set(rules.map((rule) => rule.conditions.ancestor?.pageId ?? "").filter((id) => id !== ""))],
        [rules],
    );
    const paths = usePagePaths(underIds, underIds.length > 0);
    const busy = save.isPending || reorder.isPending;

    const typeName = useCallback((typeId: string): string => byType.get(typeId)?.name ?? said.lostType, [byType]);
    const ruleName = useCallback(
        (ruleId: string): string => {
            const held = rules.find((rule) => rule.id === ruleId);
            if (held === undefined) {
                return ruleId === fallback?.id ? said.fallback.name : said.previewDialog.noRule;
            }
            return said.ruleName(held.number, held.name === "" ? said.unnamed : held.name);
        },
        [rules, fallback],
    );
    const names: Names = useMemo(() => ({ type: typeName, page: (pageId) => paths.get(pageId) ?? said.lostPage }), [typeName, paths]);

    const enable = (rule: Rule, enabled: boolean): void => {
        save.mutate({ id: rule.id, siteId, name: rule.name, enabled, typeId: rule.typeId, conditions: rule.conditions });
    };

    const move = (rule: Rule, delta: -1 | 1): void => {
        const order = reordered(
            rules.map((held) => held.id),
            rule.id,
            delta,
        );
        if (order !== null) {
            reorder.mutate({ siteId, ruleIds: order });
        }
    };

    const retype = (typeId: string): void => {
        if (fallback === null) {
            return;
        }
        save.mutate(
            { id: fallback.id, siteId, name: fallback.name, enabled: true, typeId, conditions: {} },
            {
                onSuccess: () => {
                    pushToast("info", said.fallback.saved(typeName(typeId)));
                },
            },
        );
    };

    const failed = listed.error ?? typed.error;

    return (
        <Panel>
            <PanelHeader title={said.rulesTitle}>
                <Button
                    size="sm"
                    icon={PreviewIcon}
                    disabled={listed.data === undefined}
                    onClick={() => {
                        setPreviewing(true);
                    }}
                >
                    {said.preview}
                </Button>
                <Button
                    size="sm"
                    variant="primary"
                    icon={AddIcon}
                    disabled={listed.data === undefined}
                    onClick={() => {
                        setCreating(true);
                    }}
                >
                    {said.add}
                </Button>
            </PanelHeader>
            {listed.isPending || typed.isPending ? (
                <SkeletonRows rows={6} label={said.loading} className="p-3" />
            ) : failed !== null ? (
                <div className="p-3">
                    <Banner
                        tone="danger"
                        title={failure(failed).message}
                        actions={
                            <Button
                                size="sm"
                                onClick={() => {
                                    void listed.refetch();
                                    void typed.refetch();
                                }}
                            >
                                {copy.app.retry}
                            </Button>
                        }
                    />
                </div>
            ) : (
                <ol aria-label={said.rulesTitle} className="relative flex flex-col">
                    <span aria-hidden={true} className="absolute top-5 bottom-5 left-[1.5625rem] w-px bg-hairline" />
                    {rules.length === 0 ? <li className="px-12 py-3 text-xs text-ink-dim">{said.noRules}</li> : null}
                    {rules.map((rule, at) => (
                        <RuleRow
                            key={rule.id}
                            rule={rule}
                            first={at === 0}
                            last={at === rules.length - 1}
                            names={names}
                            type={byType.get(rule.typeId)}
                            busy={busy}
                            onOpen={() => {
                                setEditing(rule);
                            }}
                            onEnable={(enabled) => {
                                enable(rule, enabled);
                            }}
                            onMove={(delta) => {
                                move(rule, delta);
                            }}
                        />
                    ))}
                    {fallback === null ? null : <FallbackRow rule={fallback} types={types} busy={busy} onType={retype} />}
                </ol>
            )}
            {save.error === null ? null : <p className="px-3 pb-3 text-xs text-danger">{failure(save.error).message}</p>}
            {editing === null && !creating ? null : (
                <RuleDrawer
                    key={editing?.id ?? "new"}
                    siteId={siteId}
                    rule={editing}
                    types={types}
                    typeName={typeName}
                    ruleName={ruleName}
                    onClose={() => {
                        setEditing(null);
                        setCreating(false);
                    }}
                />
            )}
            <ClassificationPreview
                open={previewing}
                onOpenChange={setPreviewing}
                siteId={siteId}
                fallbackType={fallback === null ? said.lostType : typeName(fallback.typeId)}
                typeName={typeName}
                ruleName={ruleName}
            />
        </Panel>
    );
}

export function ClassificationScreen(): ReactElement {
    const [searchParams, setSearchParams] = useSearchParams();
    const sites = useSites();
    const rows = flatten(sites.data?.pages);
    const siteId = pickSite(searchParams.get("site"), readLastSite(), rows.map((site) => site.id));

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
                                setSearchParams(new URLSearchParams({ site }), { replace: true });
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
                <SiteRules key={siteId} siteId={siteId} />
            )}
        </div>
    );
}
