import { copy } from "../../copy/index.js";
import type { SiteSettings } from "../../data/types.js";
import type { UrlForm } from "../../generated/vocab.js";
import { isOneOf, urlForms } from "../../generated/vocab.js";

const said = copy.sites.settings;

const maxConcurrency = 16;

export interface SettingsDraft {
    userAgent: string;
    renderConcurrency: string;
    fullWalkEvery: string;
    linkUrlForm: UrlForm;
}

export interface SettingsProblems {
    renderConcurrency: string | null;
    fullWalkEvery: string | null;
}

export function draftOf(settings: SiteSettings): SettingsDraft {
    const form: string = settings.linkUrlForm;
    return {
        userAgent: settings.userAgent,
        renderConcurrency: String(settings.observe.renderConcurrency),
        fullWalkEvery: String(settings.observe.fullWalkEvery),
        linkUrlForm: isOneOf(urlForms, form) ? form : "auto",
    };
}

function wholeOf(text: string): number | null {
    const trimmed = text.trim();
    return /^\d+$/.test(trimmed) ? Number(trimmed) : null;
}

export function problemsOf(draft: SettingsDraft): SettingsProblems {
    const concurrency = wholeOf(draft.renderConcurrency);
    const cadence = wholeOf(draft.fullWalkEvery);
    return {
        renderConcurrency:
            concurrency === null ? said.problem.whole : concurrency < 1 || concurrency > maxConcurrency ? said.problem.concurrency : null,
        fullWalkEvery: cadence === null ? said.problem.whole : cadence < 1 ? said.problem.cadence : null,
    };
}

export function sameDraft(left: SettingsDraft, right: SettingsDraft): boolean {
    return (
        left.userAgent.trim() === right.userAgent.trim() &&
        left.renderConcurrency.trim() === right.renderConcurrency.trim() &&
        left.fullWalkEvery.trim() === right.fullWalkEvery.trim() &&
        left.linkUrlForm === right.linkUrlForm
    );
}

function wireUrlForm(form: UrlForm): SiteSettings["linkUrlForm"] {
    return form as SiteSettings["linkUrlForm"];
}

export function settingsOf(stored: SiteSettings, draft: SettingsDraft): SiteSettings | null {
    const concurrency = wholeOf(draft.renderConcurrency);
    const cadence = wholeOf(draft.fullWalkEvery);
    const problems = problemsOf(draft);
    if (concurrency === null || cadence === null || problems.renderConcurrency !== null || problems.fullWalkEvery !== null) {
        return null;
    }
    return {
        ...stored,
        userAgent: draft.userAgent.trim(),
        linkUrlForm: wireUrlForm(draft.linkUrlForm),
        observe: { ...stored.observe, renderConcurrency: concurrency, fullWalkEvery: cadence },
    };
}
