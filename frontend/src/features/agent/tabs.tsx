import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { TabItem } from "../../ui/index.js";
import { Tabs } from "../../ui/index.js";

export type AgentTab = "chat" | "inbox";

export function agentTabs(
    active: AgentTab,
    waiting: number,
    navigate: (to: string) => void | Promise<void>,
): ReactElement {
    const items: readonly TabItem<AgentTab>[] = [
        { key: "chat", label: copy.agent.screen.conversations },
        {
            key: "inbox",
            label: copy.agent.inbox.title,
            count: waiting === 0 ? undefined : waiting,
            countTone: "warn",
        },
    ];
    return (
        <Tabs
            label={copy.agent.title}
            items={items}
            value={active}
            onValueChange={(key) => {
                void navigate(key === "inbox" ? "/agent/inbox" : "/agent");
            }}
        />
    );
}
