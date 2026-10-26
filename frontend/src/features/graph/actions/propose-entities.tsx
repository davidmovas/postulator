import type { ReactElement } from "react";
import { useEffect, useMemo, useRef, useState } from "react";

import { copy } from "../../../copy/index.js";
import { errorMessageOf } from "../../../data/errors.js";
import { useApplyProposals, useGraph, usePreviewFromPages, useProposeFromKeywords } from "../../../data/hooks/graph.js";
import { usePageTypes } from "../../../data/hooks/templates.js";
import { pushToast } from "../../../data/toasts.js";
import type { ApplyProposalsResponse, PreviewFromPagesRequest, ProposedEntity } from "../../../data/types.js";
import { Button, RestartAltIcon, Stars2Icon, Workspace, useElapsed } from "../../../ui/index.js";
import type { GraphIndex } from "../model/index.js";
import { buildGraphIndex } from "../model/index.js";
import type { ProposalEdits, ProposalSource } from "../model/proposal-tree.js";
import { clashesOf, everyTickable, sentProposals, sourceTypesOf } from "../model/proposal-tree.js";
import { keywordLines, writesOf } from "../model/proposals.js";
import type { ProposalMemory, ProposalOpener, ProposalStage } from "./proposal-memory.js";
import { blankMemory, forgetProposal, memoryOf, remember, seedKey, useProposalMemory } from "./proposal-memory.js";
import type { Pick, ProposalInput } from "./propose-pick.js";
import { ProposalPick, noParent } from "./propose-pick.js";
import type { ExistingEntity, NamedType, ReviewState } from "./propose-review.js";
import { ProposalConfirm, ProposalReview } from "./propose-review.js";

export type { ProposalInput } from "./propose-pick.js";

const said = copy.graph.ai;

const siteSource = "site" as PreviewFromPagesRequest["source"];

const noEdits: ProposalEdits = new Map();

type SourceTypes = Parameters<typeof sourceTypesOf>[0];

export type ProposalStart =
    | { kind: "pick"; source: ProposalInput; index: GraphIndex; onReview: () => void }
    | { kind: "seed"; seed: PreviewFromPagesRequest };

export interface ProposeEntitiesDrawerProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: string;
    start: ProposalStart;
}

function freshReview(answered: readonly ProposedEntity[], pageTypes: SourceTypes): Partial<ProposalMemory> {
    return {
        proposals: answered,
        kept: everyTickable(answered.length, clashesOf(answered, noEdits)),
        edits: noEdits,
        query: "",
        filter: "all",
        collapsed: new Set(),
        focus: null,
        anchor: null,
        pageTypes: sourceTypesOf(pageTypes),
        typeChoice: null,
    };
}

function existingOf(index: GraphIndex | null): readonly ExistingEntity[] {
    if (index === null) {
        return [];
    }
    return index.entities
        .map((entity) => ({ id: entity.id, label: index.labels.get(entity.id) ?? entity.name }))
        .sort((a, b) => a.label.localeCompare(b.label));
}

export function ProposeEntitiesDrawer({ open, onOpenChange, siteId, start }: ProposeEntitiesDrawerProps): ReactElement | null {
    const input = start.kind === "pick" ? start.source : null;
    const seed = start.kind === "seed" ? start.seed : null;
    const opener: ProposalOpener = input ?? "seed";
    const source: ProposalSource = seed === null ? "model" : "site";
    const memory = useProposalMemory(siteId, opener);
    const { stage, pick, proposals } = memory;
    const [outcome, setOutcome] = useState<ApplyProposalsResponse | null>(null);
    const [stopped, setStopped] = useState(false);
    const controller = useRef<AbortController | null>(null);

    const graph = useGraph(open && start.kind === "seed" ? siteId : null);
    const index = useMemo(
        () =>
            start.kind === "pick"
                ? start.index
                : graph.data === undefined
                  ? null
                  : buildGraphIndex(graph.data.entities ?? [], graph.data.edges ?? []),
        [start, graph.data],
    );
    const existing = useMemo(() => existingOf(index), [index]);
    const pageTypes = usePageTypes(open ? siteId : null);
    const types: readonly NamedType[] = useMemo(
        () => (pageTypes.data?.types ?? []).map((type) => ({ id: type.id, name: type.name })),
        [pageTypes.data],
    );
    const previewPages = usePreviewFromPages();
    const previewKeywords = useProposeFromKeywords();
    const apply = useApplyProposals();
    const previewing = previewPages.isPending || previewKeywords.isPending;
    const seconds = useElapsed(previewing || apply.isPending);
    const lines = useMemo(() => keywordLines(pick.keywords), [pick.keywords]);

    const setStage = (next: ProposalStage): void => {
        remember(siteId, opener, { stage: next });
    };
    const setPick = (next: Pick): void => {
        remember(siteId, opener, { pick: next });
    };
    const setReview = (patch: Partial<ReviewState>): void => {
        remember(siteId, opener, patch);
    };

    useEffect(() => {
        return () => {
            controller.current?.abort();
        };
    }, []);

    const asked = seed === null ? null : seedKey(seed);

    useEffect(() => {
        if (!open) {
            return;
        }
        setOutcome(null);
        setStopped(false);
        previewPages.reset();
        previewKeywords.reset();
        apply.reset();
        if (seed === null || asked === null) {
            return;
        }
        const held = memoryOf(siteId, "seed");
        if (held.seed === asked && held.proposals.length > 0) {
            return;
        }
        forgetProposal(siteId, "seed");
        remember(siteId, "seed", { seed: asked });
        const abort = new AbortController();
        controller.current = abort;
        previewPages.mutate(
            { request: { ...seed, source: siteSource }, signal: abort.signal },
            {
                onSuccess: (answered) => {
                    remember(siteId, "seed", freshReview(answered.entities ?? [], answered.pageTypes));
                },
            },
        );
        return () => {
            abort.abort();
        };
    }, [open, siteId, asked]);

    if (!open) {
        return null;
    }

    const received = (answered: readonly ProposedEntity[], sourceTypes: SourceTypes): void => {
        remember(siteId, opener, { ...freshReview(answered, sourceTypes), stage: "review" });
    };

    const stop = (): void => {
        controller.current?.abort();
        setStopped(true);
    };

    const preview = (): void => {
        const held = new AbortController();
        controller.current = held;
        setStopped(false);
        if (pick.input === "pages") {
            previewPages.mutate(
                { request: { siteId, pageIds: [...pick.selected] }, signal: held.signal },
                { onSuccess: (answered) => received(answered.entities ?? [], answered.pageTypes) },
            );
            return;
        }
        const request: Parameters<typeof previewKeywords.mutate>[0]["request"] = { siteId, keywords: lines };
        if (pick.parentId !== noParent) {
            request.parentEntityId = pick.parentId;
        }
        previewKeywords.mutate({ request, signal: held.signal }, { onSuccess: (answered) => received(answered.entities ?? [], null) });
    };

    const chosen = sentProposals(proposals, memory.edits, memory.kept, {
        source,
        entityName: (id) => index?.byId.get(id)?.name,
    });

    const create = (): void => {
        apply.mutate(
            { request: { siteId, entities: chosen } },
            {
                onSuccess: (answered) => {
                    forgetProposal(siteId, opener);
                    setOutcome(answered);
                },
            },
        );
    };

    const write = (): void => {
        apply.mutate(
            { request: { siteId, entities: chosen, source: siteSource } },
            {
                onSuccess: (answered) => {
                    forgetProposal(siteId, opener);
                    pushToast("info", said.siteApplied(answered.entities?.length ?? 0, answered.mapped, answered.skipped));
                    onOpenChange(false);
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

    const startOver = (): void => {
        forgetProposal(siteId, opener);
        setStopped(false);
        previewPages.reset();
        previewKeywords.reset();
        apply.reset();
    };

    const error = errorMessageOf(previewPages.error) ?? errorMessageOf(previewKeywords.error) ?? errorMessageOf(apply.error);
    const ready = pick.input === "pages" ? pick.selected.size : lines.length;
    const edges = outcome?.edges?.length ?? 0;
    const writes = writesOf(chosen);
    const shown: ProposalStage | "done" = outcome === null ? stage : "done";
    const working = (
        <span aria-live="polite" className="mr-auto text-xs text-ink-soft">
            {seed === null ? said.previewing : said.reading} <span className="font-mono text-2xs text-ink-faint">{said.elapsed(seconds)}</span>
        </span>
    );
    const restart =
        start.kind === "pick" && memory !== blankMemory(opener) && !previewing ? (
            <Button
                variant="ghost"
                icon={RestartAltIcon}
                title={said.resetHint}
                className="mr-auto"
                disabled={apply.isPending}
                onClick={startOver}
            >
                {said.reset}
            </Button>
        ) : (
            <span className="mr-auto" />
        );
    const notes = (
        <>
            {stopped && !previewing ? <span className="text-xs text-ink-dim">{said.cancelled}</span> : null}
            {error === null ? null : <span className="min-w-0 truncate text-xs text-danger">{error}</span>}
        </>
    );

    const footer =
        shown === "pick" ? (
            <>
                {previewing ? working : restart}
                {notes}
                <Button variant="secondary" onClick={previewing ? stop : () => close(false)}>
                    {previewing ? said.stop : said.cancel}
                </Button>
                <Button variant="primary" icon={Stars2Icon} busy={previewing} disabled={ready === 0 || previewing} onClick={preview}>
                    {said.preview(ready)}
                </Button>
            </>
        ) : shown === "review" ? (
            <>
                {previewing ? working : restart}
                {notes}
                {seed === null ? (
                    <Button variant="ghost" disabled={apply.isPending} onClick={() => setStage("pick")}>
                        {said.back}
                    </Button>
                ) : (
                    <Button variant="secondary" onClick={() => close(false)}>
                        {said.cancel}
                    </Button>
                )}
                <Button
                    variant="primary"
                    busy={apply.isPending}
                    disabled={chosen.length === 0 || apply.isPending || previewing}
                    onClick={seed === null ? create : () => setStage("confirm")}
                >
                    {said.create(chosen.length)}
                </Button>
            </>
        ) : shown === "confirm" ? (
            <>
                {restart}
                {notes}
                <Button variant="ghost" disabled={apply.isPending} onClick={() => setStage("review")}>
                    {said.back}
                </Button>
                <Button variant="primary" busy={apply.isPending} disabled={apply.isPending} onClick={write}>
                    {said.confirmApply}
                </Button>
            </>
        ) : (
            <>
                <span className="mr-auto" />
                <Button variant="secondary" onClick={() => close(false)}>
                    {said.close}
                </Button>
                {edges > 0 && start.kind === "pick" ? (
                    <Button
                        variant="primary"
                        onClick={() => {
                            onOpenChange(false);
                            start.onReview();
                        }}
                    >
                        {said.reviewEdges}
                    </Button>
                ) : null}
            </>
        );

    return (
        <Workspace
            open={open}
            onOpenChange={close}
            title={said.entitiesTitle}
            description={seed === null ? said.entitiesBody : said.siteBody}
            closeLabel={said.close}
            footer={footer}
        >
            {shown === "pick" && start.kind === "pick" ? (
                <div className="flex min-h-0 flex-1 flex-col gap-3 p-4">
                    <ProposalPick siteId={siteId} index={start.index} pick={pick} onPick={setPick} />
                </div>
            ) : shown === "review" ? (
                previewing ? null : (
                    <ProposalReview
                        proposals={proposals}
                        state={memory}
                        source={source}
                        existing={existing}
                        types={types}
                        onChange={setReview}
                    />
                )
            ) : shown === "confirm" ? (
                <ProposalConfirm written={writes.written} joined={writes.joined} />
            ) : (
                <p className="p-4 text-xs text-ink-soft">
                    {said.applied(outcome?.entities?.length ?? 0, outcome?.mapped ?? 0, edges, outcome?.skipped ?? 0)}
                </p>
            )}
        </Workspace>
    );
}
