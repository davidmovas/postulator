import { copy } from "../../../copy/index.js";
import { fieldLabel, technicalReasonLabel } from "../../pages/model/labels.js";

const said = copy.findings.subjects;

export type Subject = Readonly<Record<string, unknown>> | null | undefined;

export function subjectText(subject: Subject, key: string): string {
    const held = subject?.[key];
    return typeof held === "string" ? held : "";
}

function count(subject: Subject, key: string): number {
    const held = subject?.[key];
    return typeof held === "number" && Number.isFinite(held) ? held : 0;
}

export function subjectTexts(subject: Subject, key: string): string[] {
    const held = subject?.[key];
    if (!Array.isArray(held)) {
        return [];
    }
    return held.filter((item): item is string => typeof item === "string" && item !== "");
}

export function findingSentence(code: string, subject: Subject): string {
    switch (code) {
        case "parent_unresolved":
            return said.unresolved(subjectText(subject, "name"), subjectText(subject, "path"));
        case "parent_ambiguous":
            return said.ambiguous(subjectText(subject, "name"), subjectTexts(subject, "candidates"));
        case "parent_cycle":
            return said.cycle(subjectTexts(subject, "paths"));
        case "parent_name_twin":
            return said.twin(subjectText(subject, "name"), subjectText(subject, "page"), subjectTexts(subject, "terms"));
        case "parent_differs":
            return said.differs(subjectTexts(subject, "sheet"), subjectTexts(subject, "site"));
        case "row_ambiguous":
            return said.rowAmbiguous(subjectText(subject, "sheet"), subjectTexts(subject, "candidates"));
        case "row_without_page":
            return said.rowWithoutPage(subjectText(subject, "sheet"), subjectText(subject, "plannedPath"), subject?.["product"] === true);
        case "plugin_outdated":
            return said.outdated(subjectText(subject, "version"), subjectText(subject, "required"));
        case "field_changed_on_site":
            return said.field(fieldLabel(subjectText(subject, "key")));
        case "body_changed_on_site":
            return said.body;
        case "render_unreachable":
            return said.unreachable(count(subject, "status"));
        case "render_redirected":
            return said.redirected(subjectText(subject, "target"));
        case "builder_detected":
            return subjectText(subject, "builder") === "" ? said.generic : said.builder(subjectText(subject, "builder"));
        case "technical_detected":
            return subjectText(subject, "reason") === ""
                ? said.generic
                : said.technical(technicalReasonLabel(subjectText(subject, "reason")));
        case "page_title_differs":
        case "page_h1_differs":
            return subjectText(subject, "planned") === "" && subjectText(subject, "site") === ""
                ? said.generic
                : said.titleDiffers(subjectText(subject, "planned"), subjectText(subject, "site"));
        case "url_form_mixed":
            return said.urlForms(count(subject, "absolute"), count(subject, "relative"));
        case "entity_card_proposed":
            return said.cardProposed(subjectText(subject, "name"));
        default:
            return said.generic;
    }
}

export interface RowCandidate {
    pageId: string;
    path: string;
}

function listed(subject: Subject, key: string): readonly unknown[] {
    const held = subject?.[key];
    return Array.isArray(held) ? held : [];
}

export function rowCandidates(subject: Subject): RowCandidate[] {
    const paths = listed(subject, "candidates");
    const out: RowCandidate[] = [];
    listed(subject, "pageIds").forEach((pageId, at) => {
        const path = paths[at];
        if (typeof pageId === "string" && pageId !== "") {
            out.push({ pageId, path: typeof path === "string" && path !== "" ? path : pageId });
        }
    });
    return out;
}
