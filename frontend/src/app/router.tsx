import { createHashRouter, Navigate } from "react-router";

import { copy } from "../copy/index.js";
import { AgentScreen, InboxScreen } from "../features/agent/index.js";
import { GraphScreen } from "../features/graph/index.js";
import { LinksScreen } from "../features/links/index.js";
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
import {
    MonitoringIcon,
    ScheduleIcon,
    SpaceDashboardIcon,
    UploadFileIcon,
} from "../ui/index.js";
import { Landing } from "./landing.js";
import { Placeholder } from "./placeholder.js";
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
                    {
                        path: "overview",
                        element: <Placeholder title={copy.nav.overview} icon={SpaceDashboardIcon} />,
                    },
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
                    {
                        path: "schedules",
                        element: <Placeholder title={copy.nav.schedules} icon={ScheduleIcon} />,
                    },
                    {
                        path: "import",
                        element: <Placeholder title={copy.nav.importExport} icon={UploadFileIcon} />,
                    },
                    {
                        path: "reports",
                        element: <Placeholder title={copy.nav.reports} icon={MonitoringIcon} />,
                    },
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
            { path: "*", element: <Navigate to="/" replace /> },
        ],
    },
]);
