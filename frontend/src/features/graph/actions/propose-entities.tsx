import type { ReactElement } from "react";
import { useEffect, useMemo, useRef, useState } from "react";

import { copy } from "../../../copy/index.js";
import { react } from "../../../data/errors.js";
import { useApplyProposals, usePreviewFromPages, useProposeFromKeywords } from "../../../data/hooks/graph.js";
import type { ApplyProposalsResponse, Page, ProposedEntity } from "../../../data/types.js";
import type { SegmentedOption, SelectOption } from "../../../ui/index.js";
import {
    Button,
    Checkbox,
    DenseTable,
    Drawer,
    Field,
    Segmented,
    Select,
    Stars2Icon,
    TableCell,
    TableHead,
    TableRow,
    Textarea,
} from "../../../ui/index.js";
import { everyPage } from "../../pages/pick/model.js";
import { PageTree } from "../../pages/pick/tree.js";
import { kindLabel } from "../labels.js";
import type { GraphIndex } from "../model/index.js";

const drawerWidth = 688;
const rootPath = "/";
const noParent = "";

export type ProposalSource = "pages" | "keywords";

type Stage = "pick" | "review" | "done";

type PickScope = "unmapped" | "all";

const scopeOptions: readonly SegmentedOption<PickScope>[] = [
    { value: "unmapped", label: copy.graph.ai.pickScopes.unmapped },
    { value: "all", label: copy.graph.ai.pickScopes.all },
];

export function unmapped(page: Page): boolean {
    return (page.entityId === null || page.entityId === "") && page.path !== rootPath;
}

export function keywordLines(text: string): string[] {
    const seen = new Set<string>();
    const out: string[] = [];
    for (const raw of text.split(/\r?\n/)) {
        const line = raw.trim();
        const key = line.toLowerCase();
        if (line === "" || seen.has(key)) {
            continue;
        }
        seen.add(key);
        out.push(line);
    }
    return out;
}

function useElapsed(running: boolean): number {
    const [seconds, setSeconds] = useState(0);
    useEffect(() => {
        if (!running) {
            setSeconds(0);
            return undefined;
        }
        const started = Date.now();
        const timer = window.setInterval(() => {
            setSeconds(Math.round((Date.now() - started) / 1000));
        }, 1000);
        return () => {
            window.clearInterval(timer);
        };
    }, [running]);
    return seconds;
}

function messageOf(thrown: unknown): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    return reaction.kind === "silent" || reaction.kind === "unlock" ? null : reaction.message;
}

function placeOf(proposal: ProposedEntity): string {
    if (proposal.path !== undefined && proposal.path !== "") {
        return proposal.path;
    }
    return proposal.parent ?? "";
}

export interface ProposeEntitiesDrawerProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: string;
    source: ProposalSource;
    index: GraphIndex;
    onReview: () => void;
}

export function ProposeEntitiesDrawer({ open, onOpenChange, siteId, source: initialSource, index, onReview }: ProposeEntitiesDrawerProps): ReactElement | null {
    const [source, setSource] = useState<ProposalSource>(initialSource);
    const [stage, setStage] = useState<Stage>("pick");
    const [selected, setSelected] = useState<ReadonlySet<string>>(new Set());
    const [scope, setScope] = useState<PickScope>("unmapped");
    const [keywords, setKeywords] = useState("");
    const [parentId, setParentId] = useState(noParent);
    const [proposals, setProposals] = useState<readonly ProposedEntity[]>([]);
    const [kept, setKept] = useState<ReadonlySet<number>>(new Set());
    const [outcome, setOutcome] = useState<ApplyProposalsResponse | null>(null);
    const [stopped, setStopped] = useState(false);
    const controller = useRef<AbortController | null>(null);

    const previewPages = usePreviewFromPages();
    const previewKeywords = useProposeFromKeywords();
    const apply = useApplyProposals();
    const previewing = previewPages.isPending || previewKeywords.isPending;
    const seconds = useElapsed(previewing || apply.isPending);

    useEffect(() => {
        if (!open) {
            return;
        }
        setSource(initialSource);
        setStage("pick");
        setSelected(new Set());
        setScope("unmapped");
        setKeywords("");
        setParentId(noParent);
        setProposals([]);
        setKept(new Set());
        setOutcome(null);
        setStopped(false);
        previewPages.reset();
        previewKeywords.reset();
        apply.reset();
    }, [open, siteId, initialSource]);

    const keep = useMemo(() => (scope === "unmapped" ? unmapped : everyPage), [scope]);
    const lines = useMemo(() => keywordLines(keywords), [keywords]);
    const parents: SelectOption<string>[] = useMemo(
        () => [
            { value: noParent, label: copy.graph.ai.parentNone },
            ...[...index.byId.values()].map((entity) => ({ value: entity.id, label: entity.name })),
        ],
        [index],
    );

    if (!open) {
        return null;
    }

    const stop = (): void => {
        controller.current?.abort();
        setStopped(true);
    };

    const received = (answered: readonly ProposedEntity[]): void => {
        setProposals(answered);
        setKept(new Set(answered.map((_proposal, at) => at)));
        setStage("review");
    };

    const preview = (): void => {
        const held = new AbortController();
        controller.current = held;
        setStopped(false);
        if (source === "pages") {
            previewPages.mutate(
                { request: { siteId, pageIds: [...selected] }, signal: held.signal },
                { onSuccess: (answered) => received(answered.entities ?? []) },
            );
            return;
        }
        const request: Parameters<typeof previewKeywords.mutate>[0]["request"] = { siteId, keywords: lines };
        if (parentId !== noParent) {
            request.parentEntityId = parentId;
        }
        previewKeywords.mutate({ request, signal: held.signal }, { onSuccess: (answered) => received(answered.entities ?? []) });
    };

    const create = (): void => {
        const chosen = proposals.filter((_proposal, at) => kept.has(at));
        apply.mutate(
            { request: { siteId, entities: chosen } },
            {
                onSuccess: (answered) => {
                    setOutcome(answered);
                    setStage("done");
                },
            },
        );
    };

    const close = (next: boolean): void => {
        if (!next && previewing) {
            stop();
        }
        onOpenChange(next);
    };

    const error = messageOf(previewPages.error) ?? messageOf(previewKeywords.error) ?? messageOf(apply.error);
    const ready = source === "pages" ? selected.size : lines.length;
    const edges = outcome?.edges?.length ?? 0;

    const footer =
        stage === "pick" ? (
            <div className="flex items-center justify-end gap-2">
                {previewing ? (
                    <span aria-live="polite" className="mr-auto text-xs text-ink-soft">
                        {copy.graph.ai.previewing} <span className="font-mono text-2xs text-ink-faint">{copy.graph.ai.elapsed(seconds)}</span>
                    </span>
                ) : null}
                <Button variant="secondary" onClick={previewing ? stop : () => close(false)}>
                    {previewing ? copy.graph.ai.stop : copy.graph.ai.cancel}
                </Button>
                <Button variant="primary" icon={Stars2Icon} busy={previewing} disabled={ready === 0 || previewing} onClick={preview}>
                    {copy.graph.ai.preview(ready)}
                </Button>
            </div>
        ) : stage === "review" ? (
            <div className="flex items-center justify-end gap-2">
                <Button
                    variant="ghost"
                    disabled={apply.isPending}
                    onClick={() => {
                        setStage("pick");
                    }}
                >
                    {copy.graph.ai.back}
                </Button>
                <Button variant="primary" busy={apply.isPending} disabled={kept.size === 0 || apply.isPending} onClick={create}>
                    {copy.graph.ai.create(kept.size)}
                </Button>
            </div>
        ) : (
            <div className="flex items-center justify-end gap-2">
                <Button variant="secondary" onClick={() => close(false)}>
                    {copy.graph.ai.close}
                </Button>
                {edges > 0 ? (
                    <Button
                        variant="primary"
                        onClick={() => {
                            onOpenChange(false);
                            onReview();
                        }}
                    >
                        {copy.graph.ai.reviewEdges}
                    </Button>
                ) : null}
            </div>
        );

    return (
        <Drawer
            open={open}
            onOpenChange={close}
            title={copy.graph.ai.entitiesTitle}
            description={copy.graph.ai.entitiesBody}
            closeLabel={copy.graph.ai.close}
            width={drawerWidth}
            footer={footer}
        >
            <div className="flex flex-col gap-3 p-3">
                {stage === "pick" ? (
                    <>
                        <Segmented
                            label={copy.graph.ai.entitiesTitle}
                            value={source}
                            options={[
                                { value: "pages", label: copy.graph.ai.sourcePages },
                                { value: "keywords", label: copy.graph.ai.sourceKeywords },
                            ]}
                            onValueChange={setSource}
                        />
                        {source === "pages" ? (
                            <PageTree
                                siteId={siteId}
                                selected={selected}
                                onChange={setSelected}
                                pickable={unmapped}
                                keep={keep}
                                label={copy.graph.ai.pickPages}
                                picked={copy.graph.ai.pickedPages(selected.size)}
                                refusalOf={(page) => (page.path === rootPath ? copy.graph.ai.root : unmapped(page) ? null : copy.graph.ai.mapped)}
                                filter={
                                    <Segmented
                                        label={copy.graph.ai.pickScope}
                                        size="sm"
                                        value={scope}
                                        options={scopeOptions}
                                        onValueChange={setScope}
                                    />
                                }
                            />
                        ) : (
                            <>
                                <Field label={copy.graph.ai.keywords} hint={copy.graph.ai.keywordsHint}>
                                    {(control) => (
                                        <Textarea
                                            id={control.id}
                                            aria-describedby={control["aria-describedby"]}
                                            rows={10}
                                            value={keywords}
                                            onChange={(event) => {
                                                setKeywords(event.target.value);
                                            }}
                                        />
                                    )}
                                </Field>
                                <div className="flex items-center justify-between gap-2">
                                    <span className="font-mono text-2xs text-ink-faint">{copy.graph.ai.keywordsCount(lines.length)}</span>
                                    <div className="w-64">
                                        <Field label={copy.graph.ai.parentLabel}>
                                            {(control) => (
                                                <Select
                                                    id={control.id}
                                                    aria-describedby={control["aria-describedby"]}
                                                    value={parentId}
                                                    options={parents}
                                                    onValueChange={setParentId}
                                                />
                                            )}
                                        </Field>
                                    </div>
                                </div>
                            </>
                        )}
                    </>
                ) : stage === "review" ? (
                    <>
                        <div className="flex items-center justify-between gap-2">
                            <p className="text-xs text-ink-soft">
                                <span className="font-medium text-ink">{copy.graph.ai.reviewTitle(proposals.length)}</span> {copy.graph.ai.reviewBody}
                            </p>
                            <div className="flex shrink-0 gap-1">
                                <Button
                                    size="sm"
                                    variant="ghost"
                                    onClick={() => {
                                        setKept(new Set(proposals.map((_proposal, at) => at)));
                                    }}
                                >
                                    {copy.graph.ai.keepAll}
                                </Button>
                                <Button
                                    size="sm"
                                    variant="ghost"
                                    onClick={() => {
                                        setKept(new Set());
                                    }}
                                >
                                    {copy.graph.ai.keepNone}
                                </Button>
                            </div>
                        </div>
                        <DenseTable columns="3rem minmax(0,1.6fr) 6rem minmax(0,1.4fr) minmax(0,1.4fr)" label={copy.graph.ai.reviewTitle(proposals.length)}>
                            <TableHead>
                                <span>{copy.graph.ai.columnKeep}</span>
                                <span>{copy.graph.ai.columnName}</span>
                                <span>{copy.graph.ai.columnKind}</span>
                                <span>{copy.graph.ai.columnKeyword}</span>
                                <span>{copy.graph.ai.columnPlace}</span>
                            </TableHead>
                            {proposals.map((proposal, at) => (
                                <TableRow key={`${proposal.name}-${at}`} data-proposal={proposal.name}>
                                    <TableCell>
                                        <Checkbox
                                            checked={kept.has(at)}
                                            aria-label={proposal.name}
                                            onChange={() => {
                                                setKept((held) => {
                                                    const next = new Set(held);
                                                    if (next.has(at)) {
                                                        next.delete(at);
                                                    } else {
                                                        next.add(at);
                                                    }
                                                    return next;
                                                });
                                            }}
                                        />
                                    </TableCell>
                                    <TableCell title={proposal.name}>
                                        {proposal.name}
                                        {proposal.existingEntityId !== undefined && proposal.existingEntityId !== "" ? (
                                            <span className="ml-1 text-2xs text-ink-faint">{copy.graph.ai.existing}</span>
                                        ) : null}
                                    </TableCell>
                                    <TableCell muted={true}>{kindLabel(proposal.kind ?? "")}</TableCell>
                                    <TableCell muted={true} title={proposal.primaryKeyword ?? ""}>
                                        {proposal.primaryKeyword ?? ""}
                                    </TableCell>
                                    <TableCell mono={true} muted={true} title={placeOf(proposal)}>
                                        {placeOf(proposal)}
                                    </TableCell>
                                </TableRow>
                            ))}
                        </DenseTable>
                    </>
                ) : (
                    <p className="text-xs text-ink-soft">
                        {copy.graph.ai.applied(outcome?.entities?.length ?? 0, outcome?.mapped ?? 0, edges, outcome?.skipped ?? 0)}
                    </p>
                )}
                {stopped && !previewing ? <p className="text-xs text-ink-dim">{copy.graph.ai.cancelled}</p> : null}
                {error === null ? null : <p className="text-xs text-danger">{error}</p>}
            </div>
        </Drawer>
    );
}
