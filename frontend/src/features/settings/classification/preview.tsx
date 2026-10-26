import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router";

import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { detailOf, failure, reasonOf } from "../../../data/errors.js";
import { useApplyClassification, useClassificationPages, useClassificationPreview } from "../../../data/hooks/classification.js";
import { pushToast } from "../../../data/toasts.js";
import type { ClassificationGroup } from "../../../data/types.js";
import { Banner, Button, Checkbox, ChevronRightIcon, cx, Dialog, EmptyState, SkeletonRows, TaskAltIcon, Workspace } from "../../../ui/index.js";
import { findingsOfCode } from "../../findings/index.js";
import { closedOver, groupKey, keyOf, pagesIn } from "./model.js";

const said = copy.classification.previewDialog;

const typeByDefault = "type_by_default";

const keptKey = "kept";
const waitingKey = "waiting";

interface Naming {
    typeName: (typeId: string) => string;
    ruleName: (ruleId: string) => string;
}

function named(naming: Naming, typeId: string): string {
    return typeId === "" ? said.untyped : naming.typeName(typeId);
}

function labelOf(group: ClassificationGroup, naming: Naming): string {
    return group.retyped ? said.group(named(naming, group.from), named(naming, group.to)) : said.stays(named(naming, group.to));
}

interface PagesProps {
    siteId: string;
    from: string;
    to: string;
    status: "" | "kept" | "waiting";
    naming: Naming;
}

function GroupPages({ siteId, from, to, status, naming }: PagesProps): ReactElement {
    const listed = useClassificationPages({ siteId, from, to, status }, true);
    const rows = flatten(listed.data?.pages);
    if (listed.isPending) {
        return <SkeletonRows rows={3} label={said.loading} className="px-9 py-2" />;
    }
    return (
        <div className="flex flex-col pb-2 pl-9">
            <ul className="flex flex-col">
                {rows.map((row) => (
                    <li key={row.pageId} data-preview-page={row.pageId} className="flex h-6 min-w-0 items-center gap-3 pr-3 text-xs">
                        <span className="min-w-0 truncate font-mono text-ink-soft" title={row.path}>
                            {row.path}
                        </span>
                        <span className="ml-auto shrink-0 truncate text-ink-faint">
                            {status === "" ? (row.ruleId === "" ? said.noRule : naming.ruleName(row.ruleId)) : row.to === "" ? "" : named(naming, row.to)}
                        </span>
                    </li>
                ))}
            </ul>
            {listed.hasNextPage ? (
                <Button
                    size="sm"
                    variant="ghost"
                    className="self-start"
                    busy={listed.isFetchingNextPage}
                    onClick={() => {
                        void listed.fetchNextPage();
                    }}
                >
                    {said.more}
                </Button>
            ) : null}
        </div>
    );
}

interface FoldProps {
    label: string;
    open: boolean;
    onToggle: () => void;
}

function Fold({ label, open, onToggle }: FoldProps): ReactElement {
    return (
        <button
            type="button"
            aria-expanded={open}
            aria-label={open ? said.collapse(label) : said.expand(label)}
            onClick={onToggle}
            className="inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-sm text-ink-dim hover:bg-inset hover:text-ink"
        >
            <ChevronRightIcon size={14} className={cx("transition-transform duration-100", open && "rotate-90")} />
        </button>
    );
}

export interface ClassificationPreviewProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: string;
    fallbackType: string;
    typeName: (typeId: string) => string;
    ruleName: (ruleId: string) => string;
}

export function ClassificationPreview({ open, onOpenChange, siteId, fallbackType, typeName, ruleName }: ClassificationPreviewProps): ReactElement | null {
    const preview = useClassificationPreview(siteId, open);
    const apply = useApplyClassification();
    const [chosen, setChosen] = useState<ReadonlySet<string>>(new Set());
    const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set());
    const [confirming, setConfirming] = useState(false);
    const [notice, setNotice] = useState<string | null>(null);
    const naming: Naming = useMemo(() => ({ typeName, ruleName }), [typeName, ruleName]);
    const answered = preview.data;
    const groups = useMemo(() => answered?.groups ?? [], [answered]);

    useEffect(() => {
        if (open) {
            setChosen(new Set());
            setExpanded(new Set());
            setConfirming(false);
            setNotice(null);
            apply.reset();
        }
    }, [open, siteId]);

    if (!open) {
        return null;
    }

    const toggle = (key: string): void => {
        setExpanded((held) => {
            const next = new Set(held);
            if (!next.delete(key)) {
                next.add(key);
            }
            return next;
        });
    };

    const whole = chosen.size === 0;
    const counted = pagesIn(groups, whole ? null : chosen);

    const run = (): void => {
        if (answered === undefined) {
            return;
        }
        apply.mutate(
            { siteId, hash: answered.hash, groups: whole ? [] : [...chosen].map(keyOf) },
            {
                onSuccess: (result) => {
                    pushToast("info", said.applied(result.moved, result.changed));
                    setConfirming(false);
                    onOpenChange(false);
                },
                onError: (thrown) => {
                    setConfirming(false);
                    const reason = reasonOf(thrown);
                    if (reason === "preview_moved") {
                        setNotice(said.moved);
                        setChosen(new Set());
                        void preview.refetch();
                        return;
                    }
                    if (reason === "group_depends") {
                        const from = detailOf(thrown, "from");
                        const to = detailOf(thrown, "to");
                        if (typeof from === "string" && typeof to === "string") {
                            setChosen((held) => closedOver(held, groups, groupKey({ from, to }), true));
                        }
                        setNotice(said.depends);
                        return;
                    }
                    setNotice(failure(thrown).message);
                },
            },
        );
    };

    const body = ((): ReactElement => {
        if (preview.isPending || answered === undefined) {
            return <SkeletonRows rows={6} label={said.loading} className="p-4" />;
        }
        const summary = [
            answered.moving > 0 ? said.moving(answered.moving) : null,
            answered.refreshed > 0 ? said.refreshed(answered.refreshed) : null,
            answered.kept > 0 ? said.kept(answered.kept) : null,
            answered.waiting > 0 ? said.waiting(answered.waiting) : null,
        ].filter((line): line is string => line !== null);
        return (
            <div className="flex min-h-0 flex-1 flex-col overflow-auto">
                <div className="flex flex-col gap-1 border-b border-hairline px-4 py-3 text-sm text-ink-soft">
                    {summary.map((line) => (
                        <p key={line}>{line}</p>
                    ))}
                    {answered.byDefault > 0 ? (
                        <p data-preview-default={answered.byDefault}>
                            {said.byDefault(answered.byDefault, fallbackType)}{" "}
                            <Link to={findingsOfCode(siteId, typeByDefault)} className="text-accent hover:underline">
                                {said.byDefaultLink}
                            </Link>
                        </p>
                    ) : null}
                </div>
                {notice === null ? null : (
                    <div className="px-4 pt-3">
                        <Banner tone="warn" title={notice} />
                    </div>
                )}
                {groups.length === 0 ? (
                    <div className="p-6">
                        <EmptyState icon={TaskAltIcon} title={said.nothing} />
                    </div>
                ) : (
                    <ul aria-label={said.title} className="flex flex-col">
                        {groups.map((group) => {
                            const key = groupKey(group);
                            const label = labelOf(group, naming);
                            const needs = (group.dependsOn ?? []).map((held) => said.group(named(naming, held.from), named(naming, held.to)));
                            return (
                                <li key={key} data-preview-group={key} className="border-b border-inset">
                                    <div className="flex items-start gap-2 px-3 py-2">
                                        <Fold label={label} open={expanded.has(key)} onToggle={() => toggle(key)} />
                                        <Checkbox
                                            aria-label={said.choose(label)}
                                            checked={chosen.has(key)}
                                            onChange={(event) => {
                                                setNotice(null);
                                                setChosen((held) => closedOver(held, groups, key, event.target.checked));
                                            }}
                                        />
                                        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                                            <span className="flex flex-wrap items-baseline gap-x-2 text-sm">
                                                <span className="font-mono text-xs text-ink-soft">{said.groupPages(group.pages)}</span>
                                                <span className="text-ink">{label}</span>
                                            </span>
                                            <span className="flex flex-wrap gap-x-3 text-2xs text-ink-faint">
                                                {(group.rules ?? []).map((held) => (
                                                    <span key={held.ruleId}>
                                                        {held.ruleId === "" ? said.noRule : ruleName(held.ruleId)}: {held.pages}
                                                    </span>
                                                ))}
                                            </span>
                                            {needs.length === 0 ? null : <span className="text-2xs text-info">{said.needs(needs)}</span>}
                                        </div>
                                    </div>
                                    {expanded.has(key) ? <GroupPages siteId={siteId} from={group.from} to={group.to} status="" naming={naming} /> : null}
                                </li>
                            );
                        })}
                    </ul>
                )}
                {(
                    [
                        [keptKey, said.keptTitle, answered.kept],
                        [waitingKey, said.waitingTitle, answered.waiting],
                    ] as const
                ).map(([key, title, count]) =>
                    count === 0 ? null : (
                        <section key={key} data-preview-section={key} className="border-b border-inset">
                            <div className="flex items-center gap-2 px-3 py-2">
                                <Fold label={title} open={expanded.has(key)} onToggle={() => toggle(key)} />
                                <span className="text-sm text-ink-soft">{title}</span>
                                <span className="font-mono text-xs text-ink-faint">{said.groupPages(count)}</span>
                            </div>
                            {expanded.has(key) ? <GroupPages siteId={siteId} from="" to="" status={key} naming={naming} /> : null}
                        </section>
                    ),
                )}
            </div>
        );
    })();

    return (
        <Workspace
            open={open}
            onOpenChange={onOpenChange}
            title={said.title}
            description={said.description}
            closeLabel={said.close}
            footer={
                <>
                    <span className="mr-auto" />
                    <Button variant="secondary" onClick={() => onOpenChange(false)}>
                        {said.close}
                    </Button>
                    <Button
                        variant="primary"
                        disabled={answered === undefined || groups.length === 0 || apply.isPending}
                        busy={apply.isPending}
                        onClick={() => {
                            setConfirming(true);
                        }}
                    >
                        {whole ? said.applyAll(counted) : said.applyChosen(chosen.size, counted)}
                    </Button>
                </>
            }
        >
            {body}
            <Dialog
                open={confirming}
                onOpenChange={setConfirming}
                title={said.confirmTitle}
                description={whole ? said.confirmAll(counted) : said.confirmChosen(counted)}
                confirmLabel={said.confirm}
                cancelLabel={said.cancel}
                busy={apply.isPending}
                onConfirm={run}
            />
        </Workspace>
    );
}
