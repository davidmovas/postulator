import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { Navigate, useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import {
    useConversations,
    useDeleteConversation,
    usePendingActions,
    useStartConversation,
} from "../../data/hooks/agent.js";
import { useSite, useSites } from "../../data/hooks/sites.js";
import type { Conversation, Site } from "../../data/types.js";
import type { MenuEntry } from "../../ui/index.js";
import {
    AddCommentIcon,
    AddIcon,
    Banner,
    BuildIcon,
    Button,
    DeleteIcon,
    Dialog,
    EditIcon,
    IconButton,
    Input,
    Menu,
    MoreHorizIcon,
    PublicIcon,
    Screen,
    SearchIcon,
    SkeletonRows,
} from "../../ui/index.js";
import { ConversationRow } from "./conversation-row.js";
import { ConversationView } from "./conversation/view.js";
import { chooseConversation, dockKey, forgetConversation, rememberConversation, takePrefill, useDock } from "./dock/state.js";
import { inScope } from "./model/conversations.js";
import { agentHome, conversationPath } from "./model/paths.js";
import { RenameDialog } from "./rename-dialog.js";
import { agentTabs } from "./tabs.js";
import { ToolsDrawer } from "./tools-drawer.js";

interface SiteChatsProps {
    sites: readonly Site[];
    onPick: (siteId: string) => void;
    onAdd: () => void;
}

function SiteChats({ sites, onPick, onAdd }: SiteChatsProps): ReactElement {
    return (
        <Banner
            tone="info"
            className="m-3 mb-0 shrink-0"
            title={copy.agent.screen.noSiteTitle}
            body={sites.length === 0 ? copy.agent.screen.noSites : copy.agent.screen.noSiteBody}
            actions={
                sites.length === 0 ? (
                    <Button size="sm" icon={AddIcon} onClick={onAdd}>
                        {copy.agent.screen.addSite}
                    </Button>
                ) : (
                    sites.map((site) => (
                        <Button
                            key={site.id}
                            size="sm"
                            icon={PublicIcon}
                            aria-label={copy.agent.screen.siteChat(site.name)}
                            title={copy.agent.screen.siteChat(site.name)}
                            className="max-w-60"
                            onClick={() => {
                                onPick(site.id);
                            }}
                        >
                            <span className="truncate">{site.name}</span>
                        </Button>
                    ))
                )
            }
        />
    );
}

export function AgentScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const dock = useDock();
    const siteId = params.siteId ?? null;
    const conversationId = params.conversationId ?? null;
    const fresh = searchParams.get("new") === "1";
    const listed = useConversations(siteId === null ? {} : { siteId }, 100);
    const site = useSite(siteId);
    const sites = useSites({}, null, 100);
    const pending = usePendingActions({ status: "pending" }, 100);
    const remove = useDeleteConversation();
    const start = useStartConversation();
    const [query, setQuery] = useState("");
    const [started, setStarted] = useState<Conversation | null>(null);
    const [renaming, setRenaming] = useState<Conversation | null>(null);
    const [doomed, setDoomed] = useState<Conversation | null>(null);

    const everyone = useMemo(() => flatten(listed.data?.pages), [listed.data]);
    const conversations = useMemo(() => inScope(everyone, siteId, ""), [everyone, siteId]);
    const shown = useMemo(() => inScope(everyone, siteId, query), [everyone, siteId, query]);
    const siteRows = useMemo(() => flatten(sites.data?.pages), [sites.data]);
    const pendingActions = useMemo(() => flatten(pending.data?.pages), [pending.data]);
    const pendingByConversation = useMemo(() => {
        const counts = new Map<string, number>();
        for (const action of pendingActions) {
            counts.set(action.conversationId, (counts.get(action.conversationId) ?? 0) + 1);
        }
        return counts;
    }, [pendingActions]);

    const wanted = conversationId ?? chooseConversation(dock.choices[dockKey(siteId)], conversations);
    const current = fresh
        ? null
        : (conversations.find((held) => held.id === wanted) ?? (started !== null && started.id === wanted ? started : null));
    const elsewhere =
        siteId === null && conversationId !== null
            ? (everyone.find((held) => held.id === conversationId && held.siteId !== null) ?? null)
            : null;
    const missing = conversationId !== null && current === null && !fresh && !listed.isPending;
    const toolsOpen = searchParams.get("tools") === "1";
    const title =
        siteId === null
            ? copy.agent.title
            : copy.agent.screen.onSite(site.data?.site.name ?? siteRows.find((row) => row.id === siteId)?.name ?? "");

    useEffect(() => {
        setStarted(null);
    }, [siteId]);

    useEffect(() => {
        if (current !== null) {
            rememberConversation(siteId, current.id);
        }
    }, [siteId, current]);

    if (elsewhere !== null && elsewhere.siteId !== null) {
        return <Navigate to={conversationPath(elsewhere.siteId, elsewhere.id)} replace={true} />;
    }

    const open = (id: string): void => {
        void navigate(conversationPath(siteId, id));
    };

    const showTools = (): void => {
        const next = new URLSearchParams(searchParams);
        next.set("tools", "1");
        setSearchParams(next, { replace: true });
    };

    const closeTools = (): void => {
        const next = new URLSearchParams(searchParams);
        next.delete("tools");
        setSearchParams(next, { replace: true });
    };

    const overflow: MenuEntry[] = [{ key: "tools", label: copy.agent.tools.title, icon: BuildIcon, onSelect: showTools }];
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
            <div className="flex min-h-0 flex-1 flex-col gap-0.5 overflow-auto px-2 pb-2">
                {listed.isPending ? (
                    <SkeletonRows rows={8} label={copy.app.loading} />
                ) : conversations.length === 0 ? (
                    <p className="px-2 py-3 text-xs text-ink-dim">{copy.empty.conversations}</p>
                ) : shown.length === 0 ? (
                    <p className="px-2 py-3 text-xs text-ink-dim">{copy.agent.screen.noMatch}</p>
                ) : (
                    shown.map((conversation) => (
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
                    ))
                )}
            </div>
        </div>
    );

    return (
        <Screen
            title={title}
            tabs={agentTabs("chat", pendingActions.length, navigate, siteId)}
            actions={
                <>
                    <Button
                        icon={AddCommentIcon}
                        onClick={() => {
                            void navigate(`${agentHome(siteId)}?new=1`);
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
            <div className="flex h-full min-h-0 flex-col">
                {siteId === null ? (
                    <SiteChats
                        sites={siteRows}
                        onPick={(id) => {
                            void navigate(agentHome(id));
                        }}
                        onAdd={() => {
                            void navigate("/sites?action=new");
                        }}
                    />
                ) : null}
                <div className="min-h-0 flex-1">
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
                                            void navigate(agentHome(siteId));
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
                            siteId={siteId}
                            starting={start.isPending}
                            startError={start.error}
                            prefillSeq={dock.focusSeq}
                            takePrefill={takePrefill}
                            onStart={(text) => {
                                start.mutate(
                                    { siteId, mode: "confirm", text },
                                    {
                                        onSuccess: (conversation) => {
                                            setStarted(conversation);
                                            rememberConversation(siteId, conversation.id);
                                            open(conversation.id);
                                        },
                                    },
                                );
                            }}
                            onOpenTools={showTools}
                        />
                    )}
                </div>
            </div>
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
                                    void navigate(agentHome(siteId));
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
