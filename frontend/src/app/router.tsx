import { createHashRouter, Navigate } from "react-router";

import { copy } from "../copy/index.js";
import { OnboardingScreen } from "../features/onboarding/index.js";
import { PagesScreen } from "../features/pages/index.js";
import { SitesScreen } from "../features/sites/index.js";
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
                    { path: "graph", element: panel(copy.nav.graph, "wave 3, agent 3") },
                    { path: "pages", element: <PagesScreen /> },
                    { path: "pages/:pageId", element: <PagesScreen /> },
                    { path: "runs", element: panel(copy.nav.runs, "wave 3, agent 5") },
                    { path: "runs/:runId", element: panel("Run detail", "wave 3, agent 5") },
                    { path: "runs/:runId/items/:itemId", element: panel("Review drawer", "wave 3, agent 5") },
                    { path: "templates", element: panel(copy.nav.templates, "wave 3, agent 4") },
                    { path: "templates/:templateId", element: panel("Template editor", "wave 3, agent 4") },
                    { path: "schedules", element: panel(copy.nav.schedules, "wave 3, agent 7") },
                    { path: "import", element: panel(copy.nav.importExport, "wave 3, agent 8") },
                    { path: "reports", element: panel(copy.nav.reports, "wave 3, agent 7") },
                ],
            },
            {
                path: "agent",
                children: [
                    { index: true, element: panel(copy.nav.agent, "wave 3, agent 6") },
                    { path: "inbox", element: panel(copy.nav.inbox, "wave 3, agent 6") },
                    { path: ":conversationId", element: panel("Conversation", "wave 3, agent 6") },
                ],
            },
            {
                path: "settings",
                children: [
                    { index: true, element: <Navigate to="general" replace /> },
                    { path: "general", element: panel("General settings", "wave 3, agent 8") },
                    { path: "models", element: panel("Models and profiles", "wave 3, agent 8") },
                    { path: "security", element: panel("Security and backup", "wave 3, agent 8") },
                    { path: "about", element: panel("About", "wave 3, agent 8") },
                ],
            },
            { path: "*", element: <Navigate to="/sites" replace /> },
        ],
    },
]);
