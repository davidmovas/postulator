import { createHashRouter, Navigate } from "react-router";

import { AgentScreen, InboxScreen } from "../features/agent/index.js";
import { GraphScreen } from "../features/graph/index.js";
import { ImportScreen } from "../features/imports/index.js";
import { LinksScreen } from "../features/links/index.js";
import { OverviewScreen } from "../features/overview/index.js";
import { PagesScreen } from "../features/pages/index.js";
import { ReportsScreen } from "../features/reports/index.js";
import { RunDetailScreen, RunsScreen } from "../features/runs/index.js";
import { SchedulesScreen } from "../features/schedules/index.js";
import {
    AboutScreen,
    AgentSettingsScreen,
    BrowserSettingsScreen,
    ModelSettingsScreen,
    RunSettingsScreen,
    SecurityScreen,
    SettingsScreen,
} from "../features/settings/index.js";
import { SitesScreen } from "../features/sites/index.js";
import { PoliciesScreen, TemplateEditorScreen, TemplatesScreen } from "../features/templates/index.js";
import { Landing } from "./landing.js";
import { Shell } from "./shell.js";

export const router = createHashRouter([
    {
        path: "/",
        element: <Shell />,
        children: [
            { index: true, element: <Landing /> },
            { path: "sites", element: <SitesScreen /> },
            {
                path: "s/:siteId",
                children: [
                    { index: true, element: <Navigate to="overview" replace /> },
                    { path: "overview", element: <OverviewScreen /> },
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
                    { path: "templates/policies", element: <PoliciesScreen /> },
                    { path: "templates/:templateId", element: <TemplateEditorScreen /> },
                    { path: "schedules", element: <SchedulesScreen /> },
                    { path: "import", element: <ImportScreen /> },
                    { path: "reports", element: <ReportsScreen /> },
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
                    { index: true, element: <Navigate to="models" replace /> },
                    { path: "models", element: <ModelSettingsScreen /> },
                    { path: "runs", element: <RunSettingsScreen /> },
                    { path: "agent", element: <AgentSettingsScreen /> },
                    { path: "browser", element: <BrowserSettingsScreen /> },
                    { path: "security", element: <SecurityScreen /> },
                    { path: "about", element: <AboutScreen /> },
                    { path: "*", element: <Navigate to="models" replace /> },
                ],
            },
            { path: "*", element: <Navigate to="/" replace /> },
        ],
    },
]);
