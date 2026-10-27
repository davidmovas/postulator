import { describe, expect, it } from "vitest";

import { agentHome, conversationPath, inboxPath, onAgentScreen, switchedSitePath } from "./paths.js";

describe("the agent's paths", () => {
    it.each([
        { siteId: null, home: "/agent", conversation: "/agent/c1", inbox: "/agent/inbox" },
        { siteId: "s1", home: "/s/s1/agent", conversation: "/s/s1/agent/c1", inbox: "/s/s1/agent/inbox" },
    ])("puts the chat of $siteId under its site", ({ siteId, home, conversation, inbox }) => {
        expect(agentHome(siteId)).toBe(home);
        expect(conversationPath(siteId, "c1")).toBe(conversation);
        expect(inboxPath(siteId)).toBe(inbox);
    });

    it.each([
        { pathname: "/agent", on: true },
        { pathname: "/agent/inbox", on: true },
        { pathname: "/agent/c1", on: true },
        { pathname: "/s/s1/agent", on: true },
        { pathname: "/s/s1/agent/c1", on: true },
        { pathname: "/s/s1/agents", on: false },
        { pathname: "/s/s1/pages", on: false },
        { pathname: "/settings/agent", on: false },
        { pathname: "/sites", on: false },
    ])("knows $pathname is the agent's screen: $on", ({ pathname, on }) => {
        expect(onAgentScreen(pathname)).toBe(on);
    });

    it.each([
        { pathname: "/s/s1/agent/c1", to: "/s/s2/agent" },
        { pathname: "/s/s1/agent", to: "/s/s2/agent" },
        { pathname: "/agent", to: "/s/s2/agent" },
        { pathname: "/s/s1/pages/p1", to: "/s/s2/overview" },
        { pathname: "/sites", to: "/s/s2/overview" },
    ])("switching the site from $pathname lands on $to", ({ pathname, to }) => {
        expect(switchedSitePath(pathname, "s2")).toBe(to);
    });
});
