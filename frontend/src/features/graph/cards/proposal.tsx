import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { failure } from "../../../data/errors.js";
import { useAcceptCard, useDismissCard } from "../../../data/hooks/graph.js";
import { pushToast } from "../../../data/toasts.js";
import type { CardProposal } from "../../../data/types.js";
import { absoluteTime, relativeTime } from "../../../domain/format.js";
import type { Tone } from "../../../ui/index.js";
import { Banner, Button, CheckIcon, CloseIcon, cx, SectionLabel, StatusBadge } from "../../../ui/index.js";
import type { FactMark, MarkedFact } from "./model.js";
import { currentFacts, definitionMoves, proposedFacts, refusalOf } from "./model.js";

const said = copy.graph.cards;

const markTones: Readonly<Record<FactMark, Tone>> = {
    same: "muted",
    changed: "warn",
    added: "ok",
    removed: "danger",
};

interface CardColumnProps {
    title: string;
    definition: string;
    moved: boolean;
    facts: readonly MarkedFact[];
}

function CardColumn({ title, definition, moved, facts }: CardColumnProps): ReactElement {
    return (
        <section aria-label={title} className="flex min-w-0 flex-col gap-2 rounded-md border border-hairline bg-inset p-2.5">
            <SectionLabel>{title}</SectionLabel>
            <div className="flex flex-col gap-0.5">
                <div className="flex items-center gap-1.5">
                    <span className="text-2xs text-ink-faint">{said.definition}</span>
                    {moved ? (
                        <StatusBadge tone="warn" dot={false}>
                            {said.moved}
                        </StatusBadge>
                    ) : null}
                </div>
                <p className={cx("text-xs whitespace-pre-wrap", definition.trim() === "" ? "text-ink-faint" : "text-ink-soft")}>
                    {definition.trim() === "" ? said.noDefinition : definition}
                </p>
            </div>
            <div className="flex flex-col gap-1">
                <span className="text-2xs text-ink-faint">{said.facts}</span>
                {facts.length === 0 ? (
                    <p className="text-xs text-ink-faint">{said.noFacts}</p>
                ) : (
                    <ul className="flex flex-col gap-1">
                        {facts.map((fact, at) => (
                            <li key={`${fact.label}:${String(at)}`} className="flex min-w-0 items-start gap-1.5 text-xs" data-card-fact-mark={fact.mark}>
                                <span className="shrink-0 text-ink-faint">{fact.label}</span>
                                <span className={cx("min-w-0 flex-1 text-ink-soft", fact.mark === "removed" && "line-through")}>
                                    {fact.value}
                                </span>
                                {fact.mark === "same" ? null : (
                                    <StatusBadge tone={markTones[fact.mark]} dot={false}>
                                        {said.marks[fact.mark]}
                                    </StatusBadge>
                                )}
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </section>
    );
}

export interface CardProposalItemProps {
    siteId: string;
    proposal: CardProposal;
    focused: boolean;
    onReveal: (entityId: string) => void;
}

export function CardProposalItem({ siteId, proposal, focused, onReveal }: CardProposalItemProps): ReactElement {
    const accept = useAcceptCard(siteId);
    const dismiss = useDismissCard(siteId);
    const busy = accept.isPending || dismiss.isPending;
    const moved = definitionMoves(proposal);
    const refused = accept.error === null ? null : failure(accept.error);
    const refusal = refused === null ? null : refusalOf(refused.code, refused.details?.["reason"]);
    const dismissFailed = dismiss.error === null ? null : failure(dismiss.error);

    const onDismiss = (): void => {
        accept.reset();
        dismiss.mutate(
            { findingId: proposal.findingId },
            {
                onSuccess: () => {
                    pushToast("info", said.dismissed(proposal.name));
                },
            },
        );
    };

    return (
        <li
            data-card-proposal={proposal.entityId}
            aria-label={proposal.name}
            className={cx(
                "flex flex-col gap-2.5 rounded-lg border bg-panel p-3",
                focused ? "border-accent" : "border-hairline",
            )}
        >
            <header className="flex min-w-0 items-baseline gap-2">
                <button
                    type="button"
                    title={said.show(proposal.name)}
                    className="min-w-0 truncate text-left text-sm font-semibold text-ink hover:text-accent"
                    onClick={() => {
                        onReveal(proposal.entityId);
                    }}
                >
                    {proposal.name}
                </button>
                <span className="ml-auto shrink-0 text-2xs text-ink-faint" title={absoluteTime(proposal.createdAt)}>
                    {said.proposedAt(relativeTime(proposal.createdAt))}
                </span>
            </header>

            {proposal.changed ? <Banner tone="warn" title={said.editedTitle} body={said.editedBody} /> : null}

            <div className="grid grid-cols-2 gap-2">
                <CardColumn title={said.current} definition={proposal.current.definition} moved={moved} facts={currentFacts(proposal)} />
                <CardColumn title={said.proposed} definition={proposal.proposed.definition} moved={moved} facts={proposedFacts(proposal)} />
            </div>

            <div className="flex flex-col gap-0.5">
                <span className="text-2xs text-ink-faint">{said.reason}</span>
                <p className={cx("text-xs", proposal.reason.trim() === "" ? "text-ink-faint" : "text-ink-soft")}>
                    {proposal.reason.trim() === "" ? said.noReason : proposal.reason}
                </p>
            </div>

            {refusal === "changed" ? (
                <Banner tone="danger" title={said.refusedChanged} />
            ) : refusal === "settled" ? (
                <Banner tone="info" title={said.refusedSettled} />
            ) : refused !== null ? (
                <Banner tone="danger" title={refused.message} />
            ) : null}
            {dismissFailed === null ? null : <Banner tone="danger" title={dismissFailed.message} />}

            <footer className="flex justify-end gap-2">
                <Button
                    size="sm"
                    variant={proposal.changed || refusal === "changed" ? "secondary" : "ghost"}
                    icon={CloseIcon}
                    disabled={busy}
                    busy={dismiss.isPending}
                    data-card-dismiss={proposal.entityId}
                    onClick={onDismiss}
                >
                    {said.dismiss}
                </Button>
                <Button
                    size="sm"
                    variant="primary"
                    icon={CheckIcon}
                    disabled={busy}
                    busy={accept.isPending}
                    data-card-accept={proposal.entityId}
                    onClick={() => {
                        dismiss.reset();
                        accept.mutate(
                            { findingId: proposal.findingId },
                            {
                                onSuccess: () => {
                                    pushToast("info", said.accepted(proposal.name));
                                },
                            },
                        );
                    }}
                >
                    {said.accept}
                </Button>
            </footer>
        </li>
    );
}
