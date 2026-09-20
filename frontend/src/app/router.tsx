import { createHashRouter, Navigate } from "react-router";

import { copy } from "../copy/index.js";
import { AgentScreen, InboxScreen } from "../features/agent/index.js";
import { GraphScreen } from "../features/graph/index.js";
import { LinksScreen } from "../features/links/index.js";
import { OnboardingScreen } from "../features/onboarding/index.js";
import { PagesScreen } from "../features/pages/index.js";
import { RunDetailScreen, RunsScreen } from "../features/runs/index.js";
import {
    AboutScreen,
    GeneralSettingsScreen,
    ModelSettingsScreen,
    SecurityScreen,
    SettingsScreen,
} from "../features/settings/index.js";
import { SitesScreen } from "../features/sites/index.js";
import { TemplateEditorScreen, TemplatesScreen } from "../features/templates/index.js";
import { NotBuilt } from "./not-built.js";
import { Shell } from "./shell.js";

function panel(screen: string, wave: string) {
    return <NotBuilt screen={screen} wave={wave} />;
}

export const router = createHashRouter([
    {
        path: "/",
        element: <Shell />,
        children: [
            { index: true, element: <Navigate to="/sites" replace /> },
            { path: "onboarding", element: <OnboardingScreen /> },
            { path: "sites", element: <SitesScreen /> },
            {
                path: "s/:siteId",
                children: [
                    { index: true, element: <Navigate to="overview" replace /> },
                    { path: "overview", element: panel(copy.nav.overview, "wave 3, agent 7") },
                    { path: "graph", element: <GraphScreen /> },
                    { path: "graph/:entityId", element: <GraphScreen /> },
                    { path: "pages", element: <PagesScreen /> },
                    { path: "pages/:pageId", element: <PagesScreen /> },
                    { path: "links", element: <LinksScreen /> },
                    { path: "links/:pageId", element: <LinksScreen /> },
                    { path: "runs", element: <RunsScreen /> },
                    { path: "runs/:runId", element: <RunDetailScreen /> },
                    { path: "runs/:runId/items/:itemId", element: <RunDetailScreen /> },
                    { path: "templates", element: <TemplatesScreen /> },
                    { path: "templates/:templateId", element: <TemplateEditorScreen /> },
                    { path: "schedules", element: panel(copy.nav.schedules, "wave 3, agent 7") },
                    { path: "import", element: panel(copy.nav.importExport, "wave 3, agent 8") },
                    { path: "reports", element: panel(copy.nav.reports, "wave 3, agent 7") },
                ],
            },
            {
                path: "agent",
                children: [
                    { index: true, element: <AgentScreen /> },
                    { path: "inbox", element: <InboxScreen /> },
                    { path: ":conversationId", element: <AgentScreen /> },
                ],
            },
            {
                path: "settings",
                element: <SettingsScreen />,
                children: [
                    { index: true, element: <Navigate to="general" replace /> },
                    { path: "general", element: <GeneralSettingsScreen /> },
                    { path: "models", element: <ModelSettingsScreen /> },
                    { path: "security", element: <SecurityScreen /> },
                    { path: "about", element: <AboutScreen /> },
                ],
            },
            { path: "*", element: <Navigate to="/sites" replace /> },
        ],
    },
]);
