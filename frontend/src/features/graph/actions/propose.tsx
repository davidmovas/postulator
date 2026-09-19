import type { ReactElement } from "react";
import { useEffect, useRef, useState } from "react";

import { copy } from "../../../copy/index.js";
import { react } from "../../../data/errors.js";
import { useProposeFromPages, useProposeRelated } from "../../../data/hooks/graph.js";
import { useSiteOverview } from "../../../data/hooks/reports.js";
import { Dialog, Spinner, Stars2Icon } from "../../../ui/index.js";
import type { GraphIndex } from "../model/index.js";

const pagesPerCall = 40;

interface Outcome {
    line: string;
    tokens: number;
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

interface RunningProps {
    seconds: number;
}

function Running({ seconds }: RunningProps): ReactElement {
    return (
        <p className="flex items-center gap-2 text-xs text-ink-soft">
            <Spinner size={12} />
            {copy.graph.ai.running}
            <span className="font-mono text-2xs text-ink-faint">{copy.graph.ai.elapsed(seconds)}</span>
        </p>
    );
}

function messageOf(thrown: unknown): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    return reaction.kind === "silent" || reaction.kind === "unlock" ? null : reaction.message;
}

export interface ProposeFromPagesDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: string;
    onReview: () => void;
}

export function ProposeFromPagesDialog({ open, onOpenChange, siteId, onReview }: ProposeFromPagesDialogProps): ReactElement {
    const overview = useSiteOverview(open ? siteId : null);
    const propose = useProposeFromPages();
    const controller = useRef<AbortController | null>(null);
    const [outcome, setOutcome] = useState<Outcome | null>(null);
    const [stopped, setStopped] = useState(false);
    const seconds = useElapsed(propose.isPending);
    const unmapped = overview.data?.pages.unmapped ?? 0;
    const calls = Math.ceil(unmapped / pagesPerCall);

    useEffect(() => {
        if (open) {
            setOutcome(null);
            setStopped(false);
            propose.reset();
        }
    }, [open]);

    const start = (): void => {
        const held = new AbortController();
        controller.current = held;
        setStopped(false);
        propose.mutate(
            { request: { siteId }, signal: held.signal },
            {
                onSuccess: (answered) => {
                    const entities = answered.entities?.length ?? 0;
                    const edges = answered.edges?.length ?? 0;
                    setOutcome({
                        line: entities + edges === 0 ? copy.graph.ai.nothingProposed : copy.graph.ai.fromPagesDone(entities, edges, answered.skipped),
                        tokens: answered.tokens,
                    });
                },
            },
        );
    };

    const close = (next: boolean): void => {
        if (!next && propose.isPending) {
            controller.current?.abort();
            setStopped(true);
        }
        onOpenChange(next);
    };

    const error = messageOf(propose.error);

    return (
        <Dialog
            open={open}
            onOpenChange={close}
            title={copy.graph.ai.fromPagesTitle}
            description={copy.graph.ai.fromPagesBody(unmapped, calls)}
            icon={Stars2Icon}
            confirmLabel={outcome === null ? copy.graph.ai.start : copy.graph.ai.review}
            cancelLabel={propose.isPending ? copy.graph.ai.stop : outcome === null ? copy.graph.ai.cancel : copy.graph.ai.close}
            busy={propose.isPending}
            onConfirm={() => {
                if (outcome !== null) {
                    onOpenChange(false);
                    onReview();
                } else if (unmapped > 0 && !propose.isPending) {
                    start();
                }
            }}
        >
            {propose.isPending ? <Running seconds={seconds} /> : null}
            {outcome === null ? null : (
                <p className="text-xs text-ink-soft">
                    {outcome.line} <span className="font-mono text-2xs text-ink-faint">· {copy.graph.ai.tokens(outcome.tokens)}</span>
                </p>
            )}
            {stopped && !propose.isPending ? <p className="text-xs text-ink-dim">{copy.graph.ai.cancelled}</p> : null}
            {error === null ? null : <p className="text-xs text-danger">{error}</p>}
        </Dialog>
    );
}

export interface ProposeRelatedDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: string;
    index: GraphIndex;
    selectedId: string | null;
    onReview: () => void;
}

export function ProposeRelatedDialog({ open, onOpenChange, siteId, index, selectedId, onReview }: ProposeRelatedDialogProps): ReactElement {
    const propose = useProposeRelated();
    const controller = useRef<AbortController | null>(null);
    const [outcome, setOutcome] = useState<Outcome | null>(null);
    const [stopped, setStopped] = useState(false);
    const [onlySelection, setOnlySelection] = useState(true);
    const seconds = useElapsed(propose.isPending);
    const selected = selectedId === null ? undefined : index.byId.get(selectedId);

    useEffect(() => {
        if (open) {
            setOutcome(null);
            setStopped(false);
            setOnlySelection(true);
            propose.reset();
        }
    }, [open]);

    const start = (): void => {
        const held = new AbortController();
        controller.current = held;
        setStopped(false);
        const request = selected !== undefined && onlySelection ? { siteId, entityId: selected.id } : { siteId };
        propose.mutate(
            { request, signal: held.signal },
            {
                onSuccess: (answered) => {
                    const edges = answered.edges?.length ?? 0;
                    setOutcome({
                        line: edges === 0 ? copy.graph.ai.nothingProposed : copy.graph.ai.relatedDone(edges, answered.skipped),
                        tokens: answered.tokens,
                    });
                },
            },
        );
    };

    const close = (next: boolean): void => {
        if (!next && propose.isPending) {
            controller.current?.abort();
            setStopped(true);
        }
        onOpenChange(next);
    };

    const error = messageOf(propose.error);

    return (
        <Dialog
            open={open}
            onOpenChange={close}
            title={copy.graph.ai.relatedTitle}
            description={selected !== undefined && onlySelection ? copy.graph.ai.relatedForBody(selected.name) : copy.graph.ai.relatedBody(index.counts.total)}
            icon={Stars2Icon}
            confirmLabel={outcome === null ? copy.graph.ai.start : copy.graph.ai.review}
            cancelLabel={propose.isPending ? copy.graph.ai.stop : outcome === null ? copy.graph.ai.cancel : copy.graph.ai.close}
            busy={propose.isPending}
            onConfirm={() => {
                if (outcome !== null) {
                    onOpenChange(false);
                    onReview();
                } else if (!propose.isPending) {
                    start();
                }
            }}
        >
            {selected !== undefined && outcome === null && !propose.isPending ? (
                <fieldset className="flex flex-col gap-1">
                    <legend className="text-2xs text-ink-faint">{copy.graph.ai.scope}</legend>
                    <label className="flex items-center gap-2 text-xs text-ink-soft">
                        <input type="radio" name="related-scope" className="accent-accent" checked={onlySelection} onChange={() => setOnlySelection(true)} />
                        {copy.graph.ai.onlySelection(selected.name)}
                    </label>
                    <label className="flex items-center gap-2 text-xs text-ink-soft">
                        <input type="radio" name="related-scope" className="accent-accent" checked={!onlySelection} onChange={() => setOnlySelection(false)} />
                        {copy.graph.ai.wholeSite}
                    </label>
                </fieldset>
            ) : null}
            {propose.isPending ? <Running seconds={seconds} /> : null}
            {outcome === null ? null : (
                <p className="text-xs text-ink-soft">
                    {outcome.line} <span className="font-mono text-2xs text-ink-faint">· {copy.graph.ai.tokens(outcome.tokens)}</span>
                </p>
            )}
            {stopped && !propose.isPending ? <p className="text-xs text-ink-dim">{copy.graph.ai.cancelled}</p> : null}
            {error === null ? null : <p className="text-xs text-danger">{error}</p>}
        </Dialog>
    );
}
