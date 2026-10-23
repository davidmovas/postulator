import { copy } from "../../../copy/index.js";
import { fieldLabel, technicalReasonLabel } from "./labels.js";

const said = copy.pages.model.findings;

export type Subject = Readonly<Record<string, unknown>> | null | undefined;

function text(subject: Subject, key: string): string {
    const held = subject?.[key];
    return typeof held === "string" ? held : "";
}

function count(subject: Subject, key: string): number {
    const held = subject?.[key];
    return typeof held === "number" && Number.isFinite(held) ? held : 0;
}

function texts(subject: Subject, key: string): string[] {
    const held = subject?.[key];
    if (!Array.isArray(held)) {
        return [];
    }
    return held.filter((item): item is string => typeof item === "string" && item !== "");
}

export function findingSentence(code: string, subject: Subject): string {
    switch (code) {
        case "parent_unresolved":
            return said.unresolved(text(subject, "name"), text(subject, "path"));
        case "parent_ambiguous":
            return said.ambiguous(text(subject, "name"), texts(subject, "candidates"));
        case "parent_cycle":
            return said.cycle(texts(subject, "paths"));
        case "parent_name_twin":
            return said.twin(text(subject, "name"), text(subject, "page"), texts(subject, "terms"));
        case "parent_differs":
            return said.differs(texts(subject, "sheet"), texts(subject, "site"));
        case "row_ambiguous":
            return said.rowAmbiguous(text(subject, "sheet"), texts(subject, "candidates"));
        case "row_without_page":
            return said.rowWithoutPage(text(subject, "sheet"), text(subject, "plannedPath"), subject?.["product"] === true);
        case "plugin_outdated":
            return said.outdated(text(subject, "version"), text(subject, "required"));
        case "field_changed_on_site":
            return said.field(fieldLabel(text(subject, "key")));
        case "body_changed_on_site":
            return said.body;
        case "render_unreachable":
            return said.unreachable(count(subject, "status"));
        case "render_redirected":
            return said.redirected(text(subject, "target"));
        case "builder_detected":
            return text(subject, "builder") === "" ? said.generic : said.builder(text(subject, "builder"));
        case "technical_detected":
            return text(subject, "reason") === ""
                ? said.generic
                : said.technical(technicalReasonLabel(text(subject, "reason")));
        case "page_title_differs":
        case "page_h1_differs":
            return text(subject, "planned") === "" && text(subject, "site") === ""
                ? said.generic
                : said.titleDiffers(text(subject, "planned"), text(subject, "site"));
        case "url_form_mixed":
            return said.urlForms(count(subject, "absolute"), count(subject, "relative"));
        case "entity_card_proposed":
            return said.cardProposed(text(subject, "name"));
        default:
            return said.generic;
    }
}
