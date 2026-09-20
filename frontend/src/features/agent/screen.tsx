import type { KeyboardEvent, ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { react } from "../../data/errors.js";
import {
    useConversations,
    useCreateConversation,
    useDeleteConversation,
    usePendingActions,
    useStartConversation,
} from "../../data/hooks/agent.js";
import { useSites } from "../../data/hooks/sites.js";
import type { Conversation } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import type { MenuEntry, SelectOption } from "../../ui/index.js";
import {
    AddCommentIcon,
    Banner,
    BuildIcon,
    Button,
    CountBadge,
    cx,
    DeleteIcon,
    Dialog,
    EditIcon,
    Field,
    IconButton,
    Input,
    Menu,
    MoreHorizIcon,
    Screen,
    SearchIcon,
    SectionLabel,
    Select,
    SkeletonRows,
} from "../../ui/index.js";
import { ConversationView } from "./conversation/view.js";
import { agentTabs } from "./tabs.js";
import { forgetConversation, globalDockKey, takePrefill, useDock } from "./dock/state.js";
import { groupBySite, matches } from "./model/conversations.js";
import { RenameDialog } from "./rename-dialog.js";
import { ToolsDrawer } from "./tools-drawer.js";

const noSite = "no-site";

interface RowProps {
    conversation: Conversation;
    active: boolean;
    pending: number;
    onOpen: () => void;
    onDelete: () => void;
}

function ConversationRow({ conversation, active, pending, onOpen, onDelete }: RowProps): ReactElement {
    const keyed = (event: KeyboardEvent<HTMLDivElement>): void => {
        if (event.key === "Enter") {
            event.preventDefault();
            onOpen();
        }
    };
    return (
        <div
            role="button"
            tabIndex={0}
            onClick={onOpen}
            onKeyDown={keyed}
            className={cx(
                "group flex items-center gap-2 rounded-md px-2 py-1.5 text-left",
                active ? "bg-accent-soft" : "hover:bg-inset",
            )}
        >
            <span
                aria-hidden={true}
                className={cx(
                    "h-1.5 w-1.5 shrink-0 rounded-full",
                    conversation.mode === "autonomous" ? "bg-danger" : "bg-ink-faint",
                )}
            />
            <div className="flex min-w-0 flex-1 flex-col">
                <span
                    className={cx(
                        "truncate text-xs",
                        conversation.title === ""
                            ? "text-ink-dim"
                            : active
                              ? "font-semibold text-ink"
                              : "text-ink-soft",
                    )}
                >
                    {conversation.title === "" ? copy.agent.untitled : conversation.title}
                </span>
                <span className="font-mono text-2xs text-ink-faint" title={absoluteTime(conversation.createdAt)}>
                    {relativeTime(conversation.createdAt)}
                </span>
            </div>
            {pending > 0 ? <CountBadge tone="warn" count={pending} /> : null}
            <IconButton
                icon={DeleteIcon}
                label={copy.agent.screen.delete}
                variant="ghost"
                size="sm"
                className="opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
                onClick={(event) => {
                    event.stopPropagation();
                    onDelete();
                }}
            />
        </div>
    );
}

interface NewDialogProps {
    open: boolean;
    sites: readonly SelectOption<string>[];
    onOpenChange: (open: boolean) => void;
    onCreated: (conversation: Conversation) => void;
}

function NewConversationDialog({ open, sites, onOpenChange, onCreated }: NewDialogProps): ReactElement {
    const create = useCreateConversation();
    const [siteId, setSiteId] = useState(noSite);
    const thrown = create.error;
    const reaction = thrown === null ? null : react(thrown);

    useEffect(() => {
        if (open) {
            setSiteId(sites[0]?.value ?? noSite);
            create.reset();
        }
    }, [open, sites]);

    return (
        <Dialog
            open={open}
            onOpenChange={onOpenChange}
            title={copy.agent.screen.newTitle}
            description={copy.agent.subtitle}
            confirmLabel={copy.agent.screen.newStart}
            cancelLabel={copy.agent.screen.cancel}
            icon={AddCommentIcon}
            busy={create.isPending}
            onConfirm={() => {
                create.mutate(
                    { siteId: siteId === noSite ? undefined : siteId, mode: "confirm" },
                    {
                        onSuccess: (answered) => {
                            onOpenChange(false);
                            onCreated(answered.conversation);
                        },
                    },
                );
            }}
        >
            <Field
                label={copy.agent.screen.newSite}
                hint={siteId === noSite ? copy.agent.screen.newEverywhere : undefined}
            >
                {(control) => (
                    <Select
                        id={control.id}
                        value={siteId}
                        options={[...sites, { value: noSite, label: copy.agent.header.everywhere }]}
                        onValueChange={setSiteId}
                    />
                )}
            </Field>
            {reaction === null || reaction.kind === "silent" || reaction.kind === "unlock" ? null : (
                <p className="text-xs text-danger">{reaction.message}</p>
            )}
        </Dialog>
    );
}

export function AgentScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const dock = useDock();
    const conversationId = params.conversationId ?? null;
    const listed = useConversations({}, 100);
    const sites = useSites({}, null, 100);
    const pending = usePendingActions({ status: "pending" }, 100);
    const remove = useDeleteConversation();
    const start = useStartConversation();
    const [query, setQuery] = useState("");
    const [creating, setCreating] = useState(false);
    const [renaming, setRenaming] = useState<Conversation | null>(null);
    const [doomed, setDoomed] = useState<Conversation | null>(null);

    const conversations = useMemo(() => flatten(listed.data?.pages), [listed.data]);
    const siteRows = useMemo(() => flatten(sites.data?.pages), [sites.data]);
    const siteNames = useMemo(() => new Map(siteRows.map((site) => [site.id, site.name])), [siteRows]);
    const siteOptions = useMemo(() => siteRows.map((site) => ({ value: site.id, label: site.name })), [siteRows]);
    const pendingActions = useMemo(() => flatten(pending.data?.pages), [pending.data]);
    const pendingByConversation = useMemo(() => {
        const counts = new Map<string, number>();
        for (const action of pendingActions) {
            counts.set(action.conversationId, (counts.get(action.conversationId) ?? 0) + 1);
        }
        return counts;
    }, [pendingActions]);
    const groups = useMemo(
        () => groupBySite(conversations.filter((held) => matches(held, query)), siteNames, null),
        [conversations, query, siteNames],
    );

    const current =
        conversationId === null
            ? (conversations[0] ?? null)
            : (conversations.find((held) => held.id === conversationId) ?? null);
    const missing = conversationId !== null && current === null && !listed.isPending;
    const toolsOpen = searchParams.get("tools") === "1";

    const open = (id: string): void => {
        void navigate(`/agent/${id}`);
    };

    const closeTools = (): void => {
        const next = new URLSearchParams(searchParams);
        next.delete("tools");
        setSearchParams(next, { replace: true });
    };

    const overflow: MenuEntry[] = [
        {
            key: "tools",
            label: copy.agent.tools.title,
            icon: BuildIcon,
            onSelect: () => {
                const next = new URLSearchParams(searchParams);
                next.set("tools", "1");
                setSearchParams(next, { replace: true });
            },
        },
    ];
    if (current !== null) {
        overflow.push({
            key: "rename",
            label: copy.agent.header.renameAction,
            icon: EditIcon,
            onSelect: () => {
                setRenaming(current);
            },
        });
        overflow.push({
            key: "delete",
            label: copy.agent.screen.delete,
            icon: DeleteIcon,
            danger: true,
            onSelect: () => {
                setDoomed(current);
            },
        });
    }

    const rail = (
        <div className="flex h-full min-h-0 flex-col">
            <div className="relative shrink-0 p-2">
                <SearchIcon
                    size={14}
                    className="pointer-events-none absolute top-1/2 left-4 -translate-y-1/2 text-ink-faint"
                />
                <Input
                    aria-label={copy.agent.screen.search}
                    placeholder={copy.agent.screen.search}
                    value={query}
                    className="pl-7"
                    onChange={(event) => {
                        setQuery(event.target.value);
                    }}
                />
            </div>
            <div className="min-h-0 flex-1 overflow-auto px-2 pb-2">
                {listed.isPending ? (
                    <SkeletonRows rows={8} label={copy.app.loading} />
                ) : conversations.length === 0 ? (
                    <p className="px-2 py-3 text-xs text-ink-dim">{copy.empty.conversations}</p>
                ) : groups.length === 0 ? (
                    <p className="px-2 py-3 text-xs text-ink-dim">{copy.agent.screen.noMatch}</p>
                ) : (
                    groups.map((group) => (
                        <section key={group.key} className="flex flex-col gap-0.5 pt-2">
                            <SectionLabel className="px-2 pb-1">
                                {group.key === globalDockKey
                                    ? copy.agent.screen.groupEverywhere
                                    : (group.name ?? group.key)}
                            </SectionLabel>
                            {group.conversations.map((conversation) => (
                                <ConversationRow
                                    key={conversation.id}
                                    conversation={conversation}
                                    active={current?.id === conversation.id}
                                    pending={pendingByConversation.get(conversation.id) ?? 0}
                                    onOpen={() => {
                                        open(conversation.id);
                                    }}
                                    onDelete={() => {
                                        setDoomed(conversation);
                                    }}
                                />
                            ))}
                        </section>
                    ))
                )}
            </div>
        </div>
    );

    return (
        <Screen
            title={copy.agent.title}
            tabs={agentTabs("chat", pendingActions.length, navigate)}
            actions={
                <>
                    <Button
                        icon={AddCommentIcon}
                        onClick={() => {
                            setCreating(true);
                        }}
                    >
                        {copy.agent.header.newConversation}
                    </Button>
                    <Menu
                        label={copy.app.more}
                        align="end"
                        items={overflow}
                        trigger={
                            <IconButton icon={MoreHorizIcon} label={copy.app.more} variant="ghost" size="sm" />
                        }
                    />
                </>
            }
            variant="split"
            left={rail}
        >
            {listed.isPending ? (
                <div className="p-4">
                    <SkeletonRows rows={8} label={copy.agent.states.loading} />
                </div>
            ) : missing ? (
                <div className="p-4">
                    <Banner
                        tone="warn"
                        title={copy.errors.NOT_FOUND}
                        actions={
                            <Button
                                onClick={() => {
                                    void navigate("/agent");
                                }}
                            >
                                {copy.agent.screen.conversations}
                            </Button>
                        }
                    />
                </div>
            ) : (
                <ConversationView
                    conversation={current}
                    siteId={null}
                    starting={start.isPending}
                    startError={start.error}
                    prefillSeq={dock.focusSeq}
                    takePrefill={takePrefill}
                    onStart={(text) => {
                        start.mutate(
                            { siteId: null, mode: "confirm", text },
                            {
                                onSuccess: (conversation) => {
                                    open(conversation.id);
                                },
                            },
                        );
                    }}
                    onOpenTools={() => {
                        const next = new URLSearchParams(searchParams);
                        next.set("tools", "1");
                        setSearchParams(next, { replace: true });
                    }}
                />
            )}
            <NewConversationDialog
                open={creating}
                sites={siteOptions}
                onOpenChange={setCreating}
                onCreated={(conversation) => {
                    open(conversation.id);
                }}
            />
            <RenameDialog
                conversation={renaming}
                onClose={() => {
                    setRenaming(null);
                }}
            />
            <Dialog
                open={doomed !== null}
                onOpenChange={(next) => {
                    if (!next) {
                        setDoomed(null);
                    }
                }}
                title={copy.agent.screen.deleteTitle}
                description={copy.agent.screen.deleteBody}
                confirmLabel={copy.agent.screen.deleteConfirm}
                cancelLabel={copy.agent.screen.cancel}
                destructive={true}
                icon={DeleteIcon}
                busy={remove.isPending}
                onConfirm={() => {
                    if (doomed === null) {
                        return;
                    }
                    const target = doomed;
                    remove.mutate(
                        { conversationId: target.id },
                        {
                            onSuccess: () => {
                                forgetConversation(target.id);
                                setDoomed(null);
                                if (current?.id === target.id) {
                                    void navigate("/agent");
                                }
                            },
                        },
                    );
                }}
            />
            {toolsOpen ? <ToolsDrawer onClose={closeTools} /> : null}
        </Screen>
    );
}
