import { agent } from "./agent.js";
import { app, empty, lock, nav, shell, toasts } from "./app.js";
import { errorMessages } from "./errors.js";
import { graph } from "./graph.js";
import { links } from "./links.js";
import { onboarding, readiness } from "./onboarding.js";
import { pages } from "./pages.js";
import { palette } from "./palette.js";
import { policies, templates } from "./templates.js";
import { runs } from "./runs.js";
import { settings } from "./settings.js";
import { sites } from "./sites.js";

export { errorMessages } from "./errors.js";

export const copy = {
    app,
    lock,
    nav,
    shell,
    palette,
    empty,
    readiness,
    onboarding,
    sites,
    settings,
    runs,
    agent,
    pages,
    graph,
    links,
    templates,
    policies,
    toasts,
    errors: errorMessages,
};
