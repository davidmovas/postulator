import type { ReactElement } from "react";
import { useEffect, useMemo, useRef, useState } from "react";

import { copy } from "../../../copy/index.js";
import { errorMessageOf } from "../../../data/errors.js";
import { useApplyProposals, usePreviewFromPages, useProposeFromKeywords } from "../../../data/hooks/graph.js";
import { pushToast } from "../../../data/toasts.js";
import type { ApplyProposalsResponse, PreviewFromPagesRequest, ProposedEntity } from "../../../data/types.js";
import { Button, Drawer, Stars2Icon, cx, useElapsed } from "../../../ui/index.js";
import type { GraphIndex } from "../model/index.js";
import { chosenOf, everyTickable, keywordLines, writesOf } from "../model/proposals.js";
import type { Pick, ProposalInput } from "./propose-pick.js";
import { ProposalPick, freshPick, noParent } from "./propose-pick.js";
import { ProposalConfirm, ProposalReview } from "./propose-review.js";

export type { ProposalInput } from "./propose-pick.js";

const said = copy.graph.ai;

const drawerWidth = 688;

const siteSource = "site" as PreviewFromPagesRequest["source"];

export type ProposalStart =
    | { kind: "pick"; source: ProposalInput; index: GraphIndex; onReview: () => void }
    | { kind: "seed"; seed: PreviewFromPagesRequest };

type Stage = "pick" | "review" | "confirm" | "done";

export interface ProposeEntitiesDrawerProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: string;
    start: ProposalStart;
}

export function ProposeEntitiesDrawer({ open, onOpenChange, siteId, start }: ProposeEntitiesDrawerProps): ReactElement | null {
    const input = start.kind === "pick" ? start.source : null;
    const seed = start.kind === "seed" ? start.seed : null;
    const [stage, setStage] = useState<Stage>(seed === null ? "pick" : "review");
    const [pick, setPick] = useState<Pick>(() => freshPick(input ?? "pages"));
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
    const lines = useMemo(() => keywordLines(pick.keywords), [pick.keywords]);

    const received = (answered: readonly ProposedEntity[]): void => {
        setProposals(answered);
        setKept(everyTickable(answered));
        setStage("review");
    };

    useEffect(() => {
        if (!open) {
            return;
        }
        setStage(seed === null ? "pick" : "review");
        setPick(freshPick(input ?? "pages"));
        setProposals([]);
        setKept(new Set());
        setOutcome(null);
        setStopped(false);
        previewPages.reset();
        previewKeywords.reset();
        apply.reset();
        if (seed === null) {
            return;
        }
        const held = new AbortController();
        controller.current = held;
        previewPages.mutate({ request: { ...seed, source: siteSource }, signal: held.signal }, { onSuccess: (answered) => received(answered.entities ?? []) });
        return () => {
            held.abort();
        };
    }, [open, siteId, input, seed]);

    if (!open) {
        return null;
    }

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
                { onSuccess: (answered) => received(answered.entities ?? []) },
            );
            return;
        }
        const request: Parameters<typeof previewKeywords.mutate>[0]["request"] = { siteId, keywords: lines };
        if (pick.parentId !== noParent) {
            request.parentEntityId = pick.parentId;
        }
        previewKeywords.mutate({ request, signal: held.signal }, { onSuccess: (answered) => received(answered.entities ?? []) });
    };

    const chosen = chosenOf(proposals, kept);

    const create = (): void => {
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

    const write = (): void => {
        apply.mutate(
            { request: { siteId, entities: chosen, source: siteSource } },
            {
                onSuccess: (answered) => {
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

    const error = errorMessageOf(previewPages.error) ?? errorMessageOf(previewKeywords.error) ?? errorMessageOf(apply.error);
    const ready = pick.input === "pages" ? pick.selected.size : lines.length;
    const edges = outcome?.edges?.length ?? 0;
    const writes = writesOf(chosen);
    const working = (
        <span aria-live="polite" className="mr-auto text-xs text-ink-soft">
            {seed === null ? said.previewing : said.reading} <span className="font-mono text-2xs text-ink-faint">{said.elapsed(seconds)}</span>
        </span>
    );

    const footer =
        stage === "pick" ? (
            <div className="flex items-center justify-end gap-2">
                {previewing ? working : null}
                <Button variant="secondary" onClick={previewing ? stop : () => close(false)}>
                    {previewing ? said.stop : said.cancel}
                </Button>
                <Button variant="primary" icon={Stars2Icon} busy={previewing} disabled={ready === 0 || previewing} onClick={preview}>
                    {said.preview(ready)}
                </Button>
            </div>
        ) : stage === "review" ? (
            <div className="flex items-center justify-end gap-2">
                {previewing ? working : null}
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
            </div>
        ) : stage === "confirm" ? (
            <div className="flex items-center justify-end gap-2">
                <Button variant="ghost" disabled={apply.isPending} onClick={() => setStage("review")}>
                    {said.back}
                </Button>
                <Button variant="primary" busy={apply.isPending} disabled={apply.isPending} onClick={write}>
                    {said.confirmApply}
                </Button>
            </div>
        ) : (
            <div className="flex items-center justify-end gap-2">
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
            </div>
        );

    return (
        <Drawer
            open={open}
            onOpenChange={close}
            title={said.entitiesTitle}
            description={seed === null ? said.entitiesBody : said.siteBody}
            closeLabel={said.close}
            width={drawerWidth}
            footer={footer}
        >
            <div className={cx("flex flex-col gap-3 p-3", stage === "review" && "h-full")}>
                {stage === "pick" && start.kind === "pick" ? (
                    <ProposalPick siteId={siteId} index={start.index} pick={pick} onPick={setPick} />
                ) : stage === "review" ? (
                    previewing ? null : <ProposalReview proposals={proposals} kept={kept} onKept={setKept} fromSite={seed !== null} />
                ) : stage === "confirm" ? (
                    <ProposalConfirm written={writes.written} joined={writes.joined} />
                ) : (
                    <p className="text-xs text-ink-soft">
                        {said.applied(outcome?.entities?.length ?? 0, outcome?.mapped ?? 0, edges, outcome?.skipped ?? 0)}
                    </p>
                )}
                {stopped && !previewing ? <p className="shrink-0 text-xs text-ink-dim">{said.cancelled}</p> : null}
                {error === null ? null : <p className="shrink-0 text-xs text-danger">{error}</p>}
            </div>
        </Drawer>
    );
}
