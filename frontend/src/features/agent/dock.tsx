import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import type { PointerEvent, ReactElement } from "react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useConversations } from "../../data/hooks/agent.js";
import type { Conversation } from "../../data/types.js";
import {
    AddCommentIcon,
    cx,
    ForumIcon,
    IconButton,
    KeyboardArrowDownIcon,
    OpenInFullIcon,
    RightPanelCloseIcon,
    SmartToyIcon,
} from "../../ui/index.js";
import { StartConversation } from "./conversation/start.js";
import { ConversationView } from "./conversation/view.js";
import { closeDock, rememberConversation, setDockWidth, takePrefill, useDock } from "./dock-state.js";
import { chooseConversation, dockKey } from "./model/dock.js";

const recentInSwitcher = 10;

interface SwitcherProps {
    current: Conversation | null;
    recent: readonly Conversation[];
    onPick: (id: string) => void;
    onNew: () => void;
    onAll: () => void;
}

function Switcher({ current, recent, onPick, onNew, onAll }: SwitcherProps): ReactElement {
    const label = current === null ? copy.agent.header.newConversation : current.title === "" ? copy.agent.untitled : current.title;
    return (
        <DropdownMenu.Root>
            <DropdownMenu.Trigger asChild={true}>
                <button
                    type="button"
                    title={copy.agent.header.switchConversation}
                    className="flex h-6 min-w-0 flex-1 items-center gap-1 rounded-md px-1 text-left hover:bg-inset"
                >
                    <span className={cx("truncate text-xs font-semibold", current === null || current.title === "" ? "text-ink-dim" : "text-ink")}>
                        {label}
                    </span>
                    <KeyboardArrowDownIcon size={14} className="shrink-0 text-ink-faint" />
                </button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
                <DropdownMenu.Content
                    align="start"
                    sideOffset={4}
                    aria-label={copy.agent.header.switchConversation}
                    className="z-30 w-72 rounded-md border border-edge bg-raised p-1 data-[state=open]:animate-fade-in"
                >
                    <DropdownMenu.Item
                        className="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1 text-xs text-ink outline-none data-[highlighted]:bg-inset"
                        onSelect={onNew}
                    >
                        <AddCommentIcon size={14} className="text-accent" />
                        {copy.agent.header.newConversation}
                    </DropdownMenu.Item>
                    {recent.length === 0 ? null : (
                        <>
                            <DropdownMenu.Separator className="my-1 h-px bg-hairline" />
                            <DropdownMenu.Label className="px-2 py-1 text-2xs font-semibold tracking-label text-ink-faint uppercase">
                                {copy.agent.header.recent}
                            </DropdownMenu.Label>
                            {recent.map((held) => (
                                <DropdownMenu.Item
                                    key={held.id}
                                    className={cx(
                                        "flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1 text-xs outline-none data-[highlighted]:bg-inset",
                                        held.id === current?.id ? "text-accent" : "text-ink-soft data-[highlighted]:text-ink",
                                    )}
                                    onSelect={() => {
                                        onPick(held.id);
                                    }}
                                >
                                    <span
                                        aria-hidden={true}
                                        className={cx("h-1.5 w-1.5 shrink-0 rounded-full", held.mode === "autonomous" ? "bg-danger" : "bg-ink-faint")}
                                    />
                                    <span className="truncate">{held.title === "" ? copy.agent.untitled : held.title}</span>
                                </DropdownMenu.Item>
                            ))}
                        </>
                    )}
                    <DropdownMenu.Separator className="my-1 h-px bg-hairline" />
                    <DropdownMenu.Item
                        className="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1 text-xs text-ink-soft outline-none data-[highlighted]:bg-inset data-[highlighted]:text-ink"
                        onSelect={onAll}
                    >
                        <ForumIcon size={14} />
                        {copy.agent.header.more}
                    </DropdownMenu.Item>
                </DropdownMenu.Content>
            </DropdownMenu.Portal>
        </DropdownMenu.Root>
    );
}

export interface AgentDockProps {
    siteId: string | null;
}

export function AgentDock({ siteId }: AgentDockProps): ReactElement {
    const navigate = useNavigate();
    const dock = useDock();
    const listed = useConversations(siteId === null ? {} : { siteId }, 50);
    const conversations = useMemo(() => flatten(listed.data?.pages), [listed.data]);
    const [fresh, setFresh] = useState(false);
    const [started, setStarted] = useState<Conversation | null>(null);
    const dragging = useRef<{ startX: number; startWidth: number } | null>(null);

    useEffect(() => {
        setFresh(false);
        setStarted(null);
    }, [siteId]);

    const remembered = dock.choices[dockKey(siteId)];
    const chosenId = fresh ? null : chooseConversation(remembered, conversations);
    const current = chosenId === null ? null : (conversations.find((held) => held.id === chosenId) ?? (started?.id === chosenId ? started : null));

    const pointerDown = (event: PointerEvent<HTMLDivElement>): void => {
        dragging.current = { startX: event.clientX, startWidth: dock.width };
        event.currentTarget.setPointerCapture(event.pointerId);
    };
    const pointerMove = (event: PointerEvent<HTMLDivElement>): void => {
        const held = dragging.current;
        if (held !== null) {
            setDockWidth(held.startWidth + (held.startX - event.clientX));
        }
    };
    const pointerUp = (event: PointerEvent<HTMLDivElement>): void => {
        dragging.current = null;
        event.currentTarget.releasePointerCapture(event.pointerId);
    };

    return (
        <aside
            aria-label={copy.shell.agentDock}
            className="relative flex shrink-0 flex-col border-l border-hairline bg-panel"
            style={{ width: `${dock.width}px` }}
        >
            <div
                role="separator"
                aria-orientation="vertical"
                aria-label={copy.shell.agentDock}
                onPointerDown={pointerDown}
                onPointerMove={pointerMove}
                onPointerUp={pointerUp}
                onPointerCancel={pointerUp}
                className="absolute inset-y-0 -left-0.5 z-10 w-1 cursor-col-resize hover:bg-accent-border"
            />
            <header className="flex h-8 shrink-0 items-center gap-1 border-b border-hairline pr-1 pl-2">
                <SmartToyIcon size={16} className="shrink-0 text-accent" />
                <Switcher
                    current={current}
                    recent={conversations.slice(0, recentInSwitcher)}
                    onPick={(id) => {
                        setFresh(false);
                        rememberConversation(siteId, id);
                    }}
                    onNew={() => {
                        setFresh(true);
                    }}
                    onAll={() => {
                        void navigate("/agent");
                    }}
                />
                {current === null ? null : (
                    <IconButton
                        icon={OpenInFullIcon}
                        label={copy.agent.header.openFull}
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                            void navigate(`/agent/${current.id}`);
                        }}
                    />
                )}
                <IconButton icon={RightPanelCloseIcon} label={copy.shell.collapseDock} variant="ghost" size="sm" onClick={closeDock} />
            </header>
            <div className="min-h-0 flex-1">
                {current === null ? (
                    <StartConversation
                        siteId={siteId}
                        prefillSeq={dock.focusSeq}
                        onStarted={(conversation) => {
                            setStarted(conversation);
                            setFresh(false);
                            rememberConversation(siteId, conversation.id);
                        }}
                    />
                ) : (
                    <ConversationView
                        key={current.id}
                        conversation={current}
                        prefillSeq={dock.focusSeq}
                        takePrefill={takePrefill}
                        onOpenTools={() => {
                            void navigate(`/agent/${current.id}?tools=1`);
                        }}
                    />
                )}
            </div>
        </aside>
    );
}
