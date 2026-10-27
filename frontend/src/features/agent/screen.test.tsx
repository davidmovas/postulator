import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { Conversation, ConversationFilter } from "../../data/types.js";
import { forgetConversation, rememberConversation } from "./dock/state.js";

interface StartRequest {
    siteId: string | null;
    mode: string;
    text: string;
}

interface ViewProps {
    conversation: Conversation | null;
    siteId: string | null;
    onStart: (text: string) => void;
}

function conversation(id: string, title: string, siteId: string | null, createdAt: string): Conversation {
    return { id, siteId, title, mode: "confirm", createdAt, updatedAt: createdAt };
}

const rows = [
    conversation("c1", "Plan the hub", "s1", "2026-09-19T10:00:00Z"),
    conversation("c2", "Add a site", null, "2026-09-19T11:00:00Z"),
    conversation("c3", "Fix the links", "s2", "2026-09-19T12:00:00Z"),
    conversation("c4", "Audit mugs", "s1", "2026-09-19T13:00:00Z"),
];

const sites = [
    { id: "s1", name: "Clay and Kiln" },
    { id: "s2", name: "Mugs Direct" },
];

const held: { filters: ConversationFilter[]; started: StartRequest[] } = { filters: [], started: [] };

function listOf<T>(items: readonly T[]) {
    return {
        data: { pages: [{ items, hasMore: false }] },
        isPending: false,
        error: null,
        hasNextPage: false,
        fetchNextPage: () => undefined,
    };
}

const idle = { mutate: () => undefined, reset: () => undefined, isPending: false, error: null };

vi.mock("../../data/hooks/agent.js", () => ({
    useConversations: (filter: ConversationFilter) => {
        held.filters.push(filter);
        return listOf(filter.siteId === undefined ? rows : rows.filter((row) => row.siteId === filter.siteId));
    },
    usePendingActions: () => listOf([]),
    useDeleteConversation: () => idle,
    useRenameConversation: () => idle,
    useStartConversation: () => ({
        ...idle,
        mutate: (request: StartRequest, options?: { onSuccess?: (answered: Conversation) => void }) => {
            held.started.push(request);
            options?.onSuccess?.(conversation("fresh", "", request.siteId, "2026-09-20T10:00:00Z"));
        },
    }),
}));

vi.mock("../../data/hooks/sites.js", () => ({
    useSites: () => listOf(sites),
    useSite: (id: string | null) => ({
        data: id === null ? undefined : { site: sites.find((site) => site.id === id) },
        isPending: false,
        error: null,
    }),
}));

vi.mock("./conversation/view.js", () => ({
    ConversationView: ({ conversation: shown, siteId, onStart }: ViewProps) => (
        <div data-testid="conversation" data-conversation={shown?.id ?? ""} data-site={siteId ?? ""}>
            <button
                type="button"
                onClick={() => {
                    onStart("hello");
                }}
            >
                send
            </button>
        </div>
    ),
}));

const { AgentScreen } = await import("./screen.js");
const { useDock } = await import("./dock/state.js");

function Where(): ReactElement {
    const location = useLocation();
    return <output data-testid="where">{`${location.pathname}${location.search}`}</output>;
}

function Choices(): ReactElement {
    const dock = useDock();
    return <output data-testid="choices">{JSON.stringify(dock.choices)}</output>;
}

function at(path: string): void {
    render(
        <MemoryRouter initialEntries={[path]}>
            <Routes>
                <Route path="/agent" element={<AgentScreen />} />
                <Route path="/agent/:conversationId" element={<AgentScreen />} />
                <Route path="/s/:siteId/agent" element={<AgentScreen />} />
                <Route path="/s/:siteId/agent/:conversationId" element={<AgentScreen />} />
            </Routes>
            <Where />
            <Choices />
        </MemoryRouter>,
    );
}

function listed(): HTMLElement[] {
    return screen.getAllByRole("button").filter((node) => node.tagName === "DIV");
}

function shown(): { conversation: string | undefined; site: string | undefined } {
    const view = screen.getByTestId("conversation");
    return { conversation: view.dataset.conversation, site: view.dataset.site };
}

beforeEach(() => {
    held.filters.length = 0;
    held.started.length = 0;
    for (const row of [...rows, conversation("fresh", "", null, "")]) {
        forgetConversation(row.id);
    }
});

describe("the agent inside a site", () => {
    it("lists the site's conversations alone and names the site in its header", () => {
        at("/s/s1/agent");

        expect(held.filters.every((filter) => filter.siteId === "s1")).toBe(true);
        expect(screen.getByRole("heading", { level: 1 }).textContent).toBe(copy.agent.screen.onSite("Clay and Kiln"));
        expect(listed()).toHaveLength(2);
        expect(screen.getByText("Audit mugs")).toBeTruthy();
        expect(screen.getByText("Plan the hub")).toBeTruthy();
        expect(screen.queryByText("Fix the links")).toBeNull();
        expect(screen.queryByText("Add a site")).toBeNull();
        expect(screen.queryByText(copy.agent.screen.noSiteTitle)).toBeNull();
    });

    it("opens the conversation the dock holds for the site", () => {
        rememberConversation("s1", "c1");
        at("/s/s1/agent");

        expect(shown()).toStrictEqual({ conversation: "c1", site: "s1" });
    });

    it("hands the dock the conversation it opens, so both show the same one", () => {
        rememberConversation("s1", "c1");
        at("/s/s1/agent");

        fireEvent.click(screen.getByText("Audit mugs"));

        expect(screen.getByTestId("where").textContent).toBe("/s/s1/agent/c4");
        expect(shown().conversation).toBe("c4");
        expect(JSON.parse(screen.getByTestId("choices").textContent ?? "{}")).toMatchObject({ s1: "c4" });
    });

    it("starts a new conversation on the site and opens it", () => {
        at("/s/s1/agent/c4");

        fireEvent.click(screen.getByRole("button", { name: copy.agent.header.newConversation }));
        expect(screen.getByTestId("where").textContent).toBe("/s/s1/agent?new=1");
        expect(shown()).toStrictEqual({ conversation: "", site: "s1" });

        fireEvent.click(screen.getByRole("button", { name: "send" }));

        expect(held.started).toStrictEqual([{ siteId: "s1", mode: "confirm", text: "hello" }]);
        expect(screen.getByTestId("where").textContent).toBe("/s/s1/agent/fresh");
        expect(JSON.parse(screen.getByTestId("choices").textContent ?? "{}")).toMatchObject({ s1: "fresh" });
    });
});

describe("the agent with no site", () => {
    it("lists the siteless conversations and says a site's tools need its chat", () => {
        at("/agent");

        expect(screen.getByText("Add a site")).toBeTruthy();
        expect(screen.queryByText("Plan the hub")).toBeNull();
        expect(screen.queryByText("Fix the links")).toBeNull();
        expect(screen.getByText(copy.agent.screen.noSiteTitle)).toBeTruthy();
        expect(shown()).toStrictEqual({ conversation: "c2", site: "" });

        fireEvent.click(screen.getByRole("button", { name: copy.agent.screen.siteChat("Mugs Direct") }));

        expect(screen.getByTestId("where").textContent).toBe("/s/s2/agent");
    });

    it("sends a site's conversation to the chat of its site", () => {
        at("/agent/c3");

        expect(screen.getByTestId("where").textContent).toBe("/s/s2/agent/c3");
    });

    it("starts a conversation with no site", () => {
        at("/agent?new=1");

        fireEvent.click(screen.getByRole("button", { name: "send" }));

        expect(held.started).toStrictEqual([{ siteId: null, mode: "confirm", text: "hello" }]);
        expect(screen.getByTestId("where").textContent).toBe("/agent/fresh");
    });
});
