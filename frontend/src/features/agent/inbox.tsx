import type { KeyboardEvent, ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useConfirmAction, useConversations, usePendingActions } from "../../data/hooks/agent.js";
import { useSites } from "../../data/hooks/sites.js";
import type { PendingAction } from "../../data/types.js";
import type { SegmentedOption } from "../../ui/index.js";
import {
    Banner,
    Button,
    Dialog,
    EmptyState,
    KeyboardIcon,
    PendingActionsIcon,
    Screen,
    Segmented,
    SkeletonRows,
    TaskAltIcon,
    Toolbar,
} from "../../ui/index.js";
import { ActionCard } from "./cards/action-card.js";
import type { CardBusy } from "./cards/card.js";
import { outcomeOf } from "./cards/outcome.js";
import type { InboxStatus } from "./model/inbox.js";
import { groupActions, inboxStatuses } from "./model/inbox.js";
import { agentTabs } from "./tabs.js";

function emptyText(status: InboxStatus): string {
    switch (status) {
        case "pending":
            return copy.empty.pendingActions;
        case "executed":
            return copy.agent.inbox.nothingDone;
        case "rejected":
            return copy.agent.inbox.nothingRejected;
        default:
            return copy.agent.inbox.nothingFailed;
    }
}

const filters: readonly SegmentedOption<InboxStatus>[] = inboxStatuses.map((held) => ({
    value: held,
    label: copy.agent.inbox.tabs[held],
}));

export function InboxScreen(): ReactElement {
    const navigate = useNavigate();
    const [status, setStatus] = useState<InboxStatus>("pending");
    const listed = usePendingActions({ status }, 100);
    const pendingOnly = usePendingActions({ status: "pending" }, 100);
    const conversations = useConversations({}, 100);
    const sites = useSites({}, null, 100);
    const confirm = useConfirmAction();
    const [active, setActive] = useState(0);
    const [settling, setSettling] = useState<{ id: string; busy: CardBusy } | null>(null);
    const [rejectingAll, setRejectingAll] = useState(false);
    const [rejectedSoFar, setRejectedSoFar] = useState<number | null>(null);

    const actions = useMemo(() => flatten(listed.data?.pages), [listed.data]);
    const pendingCount = useMemo(() => flatten(pendingOnly.data?.pages).length, [pendingOnly.data]);
    const rows = useMemo(() => flatten(conversations.data?.pages), [conversations.data]);
    const siteNames = useMemo(
        () => new Map(flatten(sites.data?.pages).map((site) => [site.id, site.name])),
        [sites.data],
    );
    const groups = useMemo(() => groupActions(actions, rows), [actions, rows]);
    const ordered = useMemo(() => groups.flatMap((group) => group.actions), [groups]);

    useEffect(() => {
        setActive((held) => Math.min(held, Math.max(ordered.length - 1, 0)));
    }, [ordered.length]);

    const settle = (action: PendingAction, approve: boolean): void => {
        setSettling({ id: action.id, busy: approve ? "approve" : "reject" });
        confirm.mutate(
            { actionId: action.id, approve },
            {
                onSettled: () => {
                    setSettling(null);
                },
            },
        );
    };

    const keyed = (event: KeyboardEvent<HTMLDivElement>): void => {
        if (ordered.length === 0 || event.ctrlKey || event.metaKey || event.altKey) {
            return;
        }
        const target = event.target as HTMLElement;
        if (target.tagName === "INPUT" || target.tagName === "TEXTAREA") {
            return;
        }
        const held = ordered[active];
        switch (event.key) {
            case "j":
            case "ArrowDown":
                event.preventDefault();
                setActive((index) => Math.min(index + 1, ordered.length - 1));
                break;
            case "k":
            case "ArrowUp":
                event.preventDefault();
                setActive((index) => Math.max(index - 1, 0));
                break;
            case "a":
                if (held !== undefined && held.status === "pending" && settling === null) {
                    event.preventDefault();
                    settle(held, true);
                }
                break;
            case "r":
                if (held !== undefined && held.status === "pending" && settling === null) {
                    event.preventDefault();
                    settle(held, false);
                }
                break;
            default:
                break;
        }
    };

    const rejectAll = async (): Promise<void> => {
        const doomed = ordered.filter((held) => held.status === "pending");
        setRejectedSoFar(0);
        for (const [index, held] of doomed.entries()) {
            try {
                await confirm.mutateAsync({ actionId: held.id, approve: false });
            } catch {
                break;
            }
            setRejectedSoFar(index + 1);
        }
        setRejectedSoFar(null);
        setRejectingAll(false);
    };

    return (
        <Screen
            title={copy.agent.title}
            tabs={agentTabs("inbox", pendingCount, navigate)}
            actions={
                status === "pending" && pendingCount > 0 ? (
                    <Button
                        variant="danger"
                        onClick={() => {
                            setRejectingAll(true);
                        }}
                    >
                        {copy.agent.inbox.rejectAll}
                    </Button>
                ) : undefined
            }
            toolbar={
                <Toolbar label={copy.agent.inbox.filter}>
                    <Segmented
                        label={copy.agent.inbox.filter}
                        options={filters}
                        value={status}
                        onValueChange={setStatus}
                    />
                    {ordered.length > 0 ? (
                        <span className="ml-auto flex items-center gap-1.5 font-mono text-2xs text-ink-faint">
                            <KeyboardIcon size={13} />
                            {copy.agent.inbox.keys}
                        </span>
                    ) : null}
                </Toolbar>
            }
            variant="full"
        >
            <div className="min-h-0 flex-1 overflow-auto" onKeyDown={keyed}>
                <div className="mx-auto flex w-full max-w-3xl flex-col gap-4 p-4">
                    {status === "pending" && pendingCount > 0 ? (
                        <Banner
                            tone="warn"
                            icon={PendingActionsIcon}
                            title={copy.agent.inbox.waiting(pendingCount)}
                            body={copy.agent.inbox.waitingBody}
                        />
                    ) : null}
                    {listed.isPending ? (
                        <SkeletonRows rows={6} label={copy.app.loading} />
                    ) : groups.length === 0 ? (
                        <EmptyState icon={TaskAltIcon} title={emptyText(status)} />
                    ) : (
                        groups.map((group) => (
                            <section key={group.conversationId} className="flex flex-col gap-2">
                                <div className="flex items-baseline justify-between gap-2">
                                    <Link
                                        to={`/agent/${group.conversationId}`}
                                        className="min-w-0 truncate text-xs font-semibold text-ink hover:text-accent"
                                    >
                                        {group.title === null
                                            ? copy.agent.inbox.deletedConversation
                                            : group.title === ""
                                              ? copy.agent.untitled
                                              : group.title}
                                    </Link>
                                    <span className="shrink-0 text-2xs text-ink-faint">
                                        {group.siteId === null
                                            ? copy.agent.header.everywhere
                                            : (siteNames.get(group.siteId) ?? group.siteId)}
                                    </span>
                                </div>
                                {group.actions.map((action) => {
                                    const index = ordered.indexOf(action);
                                    return (
                                        <ActionCard
                                            key={action.id}
                                            tool={action.tool}
                                            args={action.args}
                                            risk={null}
                                            status={action.status}
                                            createdAt={action.createdAt}
                                            outcome={outcomeOf(action)}
                                            busy={
                                                settling !== null && settling.id === action.id
                                                    ? settling.busy
                                                    : null
                                            }
                                            focus={index === active}
                                            keys="list"
                                            className={index === active ? "ring-1 ring-accent-border" : undefined}
                                            onApprove={() => {
                                                settle(action, true);
                                            }}
                                            onReject={() => {
                                                settle(action, false);
                                            }}
                                        />
                                    );
                                })}
                            </section>
                        ))
                    )}
                    {listed.hasNextPage ? (
                        <Button
                            variant="ghost"
                            className="self-center"
                            busy={listed.isFetchingNextPage}
                            onClick={() => {
                                void listed.fetchNextPage();
                            }}
                        >
                            {copy.app.loadMore}
                        </Button>
                    ) : null}
                </div>
            </div>
            <Dialog
                open={rejectingAll}
                onOpenChange={setRejectingAll}
                title={copy.agent.inbox.rejectAllTitle}
                description={copy.agent.inbox.rejectAllBody(pendingCount)}
                confirmLabel={copy.agent.inbox.rejectAllConfirm}
                cancelLabel={copy.agent.inbox.cancel}
                destructive={true}
                icon={PendingActionsIcon}
                busy={rejectedSoFar !== null}
                onConfirm={() => {
                    void rejectAll();
                }}
            />
        </Screen>
    );
}
