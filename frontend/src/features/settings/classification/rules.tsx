import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { failure } from "../../../data/errors.js";
import { useReorderRules, useSaveRule } from "../../../data/hooks/classification.js";
import { pushToast } from "../../../data/toasts.js";
import type { PageType, Rule } from "../../../data/types.js";
import {
    AddIcon,
    ArrowDownwardIcon,
    ArrowUpwardIcon,
    Banner,
    Button,
    cx,
    IconButton,
    Panel,
    PanelHeader,
    PreviewIcon,
    Select,
    SkeletonRows,
    StatusBadge,
    Switch,
    toneClasses,
} from "../../../ui/index.js";
import { presetIcon, presetTone } from "../types/labels.js";
import type { Names } from "./model.js";
import { reordered, sentence } from "./model.js";
import type { RuleNaming } from "./naming.js";
import { RuleDrawer } from "./rule-drawer.js";

const said = copy.classification;

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

function PageCount({ count }: { count: number }): ReactElement {
    return (
        <span data-rule-pages={count} className={cx("shrink-0 font-mono text-2xs", count === 0 ? "text-ink-faint" : "text-ink-dim")}>
            {said.pages(count)}
        </span>
    );
}

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
                    <PageCount count={rule.pages} />
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
                <span className="flex flex-wrap items-center gap-x-2">
                    <span className="text-sm font-medium text-ink">{said.fallback.name}</span>
                    <PageCount count={rule.pages} />
                </span>
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

export interface RulesViewProps {
    siteId: string;
    naming: RuleNaming;
    onPreview: () => void;
}

export function RulesView({ siteId, naming, onPreview }: RulesViewProps): ReactElement {
    const { listed, typed, rules, fallback, types, byType, typeName, ruleName, names } = naming;
    const save = useSaveRule();
    const reorder = useReorderRules();
    const [editing, setEditing] = useState<Rule | null>(null);
    const [creating, setCreating] = useState(false);
    const busy = save.isPending || reorder.isPending;

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
        <div data-rules-view={true} className="flex flex-col gap-3">
            <p className="max-w-3xl text-sm leading-relaxed text-ink-soft">{said.body}</p>
            <Panel>
                <PanelHeader title={said.rulesTitle}>
                    <span className="flex items-center gap-2">
                        <Button size="sm" icon={PreviewIcon} disabled={listed.data === undefined} onClick={onPreview}>
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
                    </span>
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
            </Panel>
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
        </div>
    );
}
