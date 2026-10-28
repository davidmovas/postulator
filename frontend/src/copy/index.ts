import { agent } from "./agent.js";
import { app, empty, lock, nav, shell, toasts } from "./app.js";
import { classification } from "./classification.js";
import { errorMessages } from "./errors.js";
import { findings } from "./findings.js";
import { graph } from "./graph.js";
import { imports } from "./imports.js";
import { links } from "./links.js";
import { map } from "./map.js";
import { overview, readiness } from "./overview.js";
import { pages } from "./pages.js";
import { palette } from "./palette.js";
import { plans } from "./plans.js";
import { reports } from "./reports.js";
import { pageTypes } from "./pagetypes.js";
import { templates } from "./templates.js";
import { runs } from "./runs.js";
import { schedules } from "./schedules.js";
import { settings } from "./settings.js";
import { sites } from "./sites.js";
import { sync } from "./sync.js";

export { errorMessages } from "./errors.js";

export const copy = {
    app,
    lock,
    nav,
    shell,
    palette,
    empty,
    readiness,
    overview,
    sites,
    sync,
    settings,
    runs,
    reports,
    schedules,
    agent,
    plans,
    pages,
    graph,
    links,
    map,
    findings,
    imports,
    templates,
    pageTypes,
    classification,
    toasts,
    errors: errorMessages,
};
