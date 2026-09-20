import type { PointerEvent, ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";

import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { useConversations, useStartConversation } from "../../../data/hooks/agent.js";
import type { Conversation } from "../../../data/types.js";
import type { MenuEntry, SegmentedOption } from "../../../ui/index.js";
import {
    AddCommentIcon,
    BoltIcon,
    ChatIcon,
    cx,
    EditIcon,
    ForumIcon,
    IconButton,
    KeyboardArrowDownIcon,
    Menu,
    OpenInFullIcon,
    RightPanelCloseIcon,
    Segmented,
    SmartToyIcon,
} from "../../../ui/index.js";
import { ConversationView } from "../conversation/view.js";
import { RenameDialog } from "../rename-dialog.js";
import type { DockMode } from "./state.js";
import {
    chooseConversation,
    closeDock,
    dockKey,
    rememberConversation,
    setDockMode,
    setDockWidth,
    takePrefill,
    useDock,
    widthOf,
} from "./state.js";

const recentInSwitcher = 8;

const widthOptions: readonly SegmentedOption<DockMode>[] = [
    { value: "narrow", label: copy.agent.dock.narrow },
    { value: "wide", label: copy.agent.dock.wide },
];

function useWindowWidth(): number {
    const [width, setWidth] = useState(() => window.innerWidth);

    useEffect(() => {
        const measure = (): void => {
            setWidth(window.innerWidth);
        };
        window.addEventListener("resize", measure);
        return () => {
            window.removeEventListener("resize", measure);
        };
    }, []);

    return width;
}

interface SwitcherProps {
    current: Conversation | null;
    recent: readonly Conversation[];
    onPick: (id: string) => void;
    onNew: () => void;
    onRename: () => void;
    onAll: () => void;
}

function Switcher({ current, recent, onPick, onNew, onRename, onAll }: SwitcherProps): ReactElement {
    const label =
        current === null
            ? copy.agent.header.newConversation
            : current.title === ""
              ? copy.agent.untitled
              : current.title;
    const items: MenuEntry[] = [
        { key: "new", label: copy.agent.header.newConversation, icon: AddCommentIcon, onSelect: onNew },
    ];
    if (current !== null) {
        items.push({
            key: "rename",
            label: copy.agent.header.renameAction,
            icon: EditIcon,
            onSelect: onRename,
        });
    }
    if (recent.length > 0) {
        items.push({ kind: "separator", key: "recent-separator" });
        items.push({ kind: "label", key: "recent-label", label: copy.agent.header.recent });
        for (const held of recent) {
            items.push({
                key: held.id,
                label: held.title === "" ? copy.agent.untitled : held.title,
                icon: held.mode === "autonomous" ? BoltIcon : ChatIcon,
                active: held.id === current?.id,
                onSelect: () => {
                    onPick(held.id);
                },
            });
        }
    }
    items.push({ kind: "separator", key: "all-separator" });
    items.push({ key: "all", label: copy.agent.header.more, icon: ForumIcon, onSelect: onAll });

    return (
        <Menu
            label={copy.agent.header.switchConversation}
            align="start"
            width={288}
            items={items}
            trigger={
                <button
                    type="button"
                    title={copy.agent.header.switchConversation}
                    className="flex h-6 min-w-0 flex-1 items-center gap-1 rounded-md px-1 text-left hover:bg-inset"
                >
                    <span
                        className={cx(
                            "truncate text-xs font-semibold",
                            current === null || current.title === "" ? "text-ink-dim" : "text-ink",
                        )}
                    >
                        {label}
                    </span>
                    <KeyboardArrowDownIcon size={14} className="shrink-0 text-ink-faint" />
                </button>
            }
        />
    );
}

export interface AgentDockProps {
    siteId: string | null;
}

export function AgentDock({ siteId }: AgentDockProps): ReactElement {
    const navigate = useNavigate();
    const dock = useDock();
    const windowWidth = useWindowWidth();
    const listed = useConversations(siteId === null ? {} : { siteId }, 50);
    const start = useStartConversation();
    const conversations = useMemo(() => flatten(listed.data?.pages), [listed.data]);
    const [fresh, setFresh] = useState(false);
    const [started, setStarted] = useState<Conversation | null>(null);
    const [renaming, setRenaming] = useState<Conversation | null>(null);

    useEffect(() => {
        setFresh(false);
        setStarted(null);
    }, [siteId]);

    const remembered = dock.choices[dockKey(siteId)];
    const chosenId = fresh ? null : chooseConversation(remembered, conversations);
    const current =
        chosenId === null
            ? null
            : (conversations.find((held) => held.id === chosenId) ?? (started?.id === chosenId ? started : null));

    const width = widthOf(dock.mode, dock.custom, windowWidth);
    const [dragging, setDragging] = useState<{ startX: number; startWidth: number } | null>(null);

    const pointerDown = (event: PointerEvent<HTMLDivElement>): void => {
        setDragging({ startX: event.clientX, startWidth: width });
        event.currentTarget.setPointerCapture(event.pointerId);
    };
    const pointerMove = (event: PointerEvent<HTMLDivElement>): void => {
        if (dragging !== null) {
            setDockWidth(dragging.startWidth + (dragging.startX - event.clientX));
        }
    };
    const pointerUp = (event: PointerEvent<HTMLDivElement>): void => {
        setDragging(null);
        event.currentTarget.releasePointerCapture(event.pointerId);
    };

    return (
        <aside
            aria-label={copy.shell.agentDock}
            className="relative flex shrink-0 flex-col border-l border-hairline bg-panel"
            style={{ width: `${width}px` }}
        >
            <div
                role="separator"
                aria-orientation="vertical"
                aria-label={copy.agent.dock.resize}
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
                    onRename={() => {
                        setRenaming(current);
                    }}
                    onAll={() => {
                        void navigate("/agent");
                    }}
                />
                <Segmented
                    label={copy.agent.dock.label}
                    size="sm"
                    options={widthOptions}
                    value={dock.mode}
                    onValueChange={setDockMode}
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
                <IconButton
                    icon={RightPanelCloseIcon}
                    label={copy.shell.collapseDock}
                    variant="ghost"
                    size="sm"
                    onClick={closeDock}
                />
            </header>
            <div className="min-h-0 flex-1">
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
                                    setFresh(false);
                                    rememberConversation(siteId, conversation.id);
                                },
                            },
                        );
                    }}
                    onOpenTools={() => {
                        if (current !== null) {
                            void navigate(`/agent/${current.id}?tools=1`);
                        }
                    }}
                />
            </div>
            <RenameDialog
                conversation={renaming}
                onClose={() => {
                    setRenaming(null);
                }}
            />
        </aside>
    );
}
