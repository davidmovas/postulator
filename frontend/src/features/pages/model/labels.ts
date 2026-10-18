import { copy } from "../../../copy/index.js";
import type {
    ContentRootState,
    FindingCode,
    FindingSeverity,
    LinkType,
    PageWpType,
    ParentMethod,
    ParentReason,
    RegionKind,
    LinkSource,
    TechnicalReason,
    TrailSource,
    UrlForm,
} from "../../../generated/vocab.js";
import {
    contentRootStates,
    findingCodes,
    findingSeverities,
    isOneOf,
    linkSources,
    linkTypes,
    pageWpTypes,
    parentMethods,
    parentReasons,
    regionKinds,
    technicalReasons,
    trailSources,
    urlForms,
} from "../../../generated/vocab.js";
import type { Tone } from "../../../ui/index.js";

const said = copy.pages.model;

const linkTypeWords: Readonly<Record<LinkType, string>> = said.linkTypes;
const linkSourceWords: Readonly<Record<LinkSource, string>> = said.linkSources;
const regionWords: Readonly<Record<RegionKind, string>> = said.regionKinds;
const reasonWords: Readonly<Record<ParentReason, string>> = said.parentReasons;
const sourceWords: Readonly<Record<ParentReason, string>> = said.parentSources;
const methodWords: Readonly<Record<ParentMethod, string>> = said.parentMethods;
const methodNames: Readonly<Record<ParentMethod, string>> = said.parentMethodNames;
const trailWords: Readonly<Record<TrailSource, string>> = said.trailSources;
const technicalWords: Readonly<Record<TechnicalReason, string>> = said.technicalReasons;
const rootWords: Readonly<Record<ContentRootState, string>> = said.rootStates;
const codeWords: Readonly<Record<FindingCode, string>> = said.findingCodes;
const severityWords: Readonly<Record<FindingSeverity, string>> = said.severities;
const urlFormWords: Readonly<Record<UrlForm, string>> = said.urlForms;
const wpTypeWords: Readonly<Record<PageWpType, string>> = said.wpTypes;

const severityTones: Readonly<Record<FindingSeverity, Tone>> = {
    info: "info",
    warning: "warn",
    error: "danger",
};

const rootTones: Readonly<Record<ContentRootState, Tone>> = {
    found: "ok",
    empty_raw: "info",
    dynamic_only: "info",
    archive: "muted",
    not_found: "warn",
};

const builderPrefix = "builder:";
const termReason = "term";

export function linkTypeLabel(type: string): string {
    return isOneOf(linkTypes, type) ? linkTypeWords[type] : type;
}

export function linkSourceLabel(source: string): string {
    return isOneOf(linkSources, source) ? linkSourceWords[source] : source;
}

export function regionKindLabel(kind: string): string {
    return isOneOf(regionKinds, kind) ? regionWords[kind] : kind;
}

export function parentReasonLabel(reason: string): string {
    return isOneOf(parentReasons, reason) ? reasonWords[reason] : reason;
}

export function parentSourceLabel(reason: string): string {
    return isOneOf(parentReasons, reason) ? sourceWords[reason] : reason;
}

export function parentMethodLabel(method: string): string {
    return isOneOf(parentMethods, method) ? methodWords[method] : method;
}

export function parentMethodName(method: string): string {
    return isOneOf(parentMethods, method) ? methodNames[method] : method;
}

export function parentReasonSentence(reason: string, method: string): string {
    const known = method !== "" && isOneOf(parentMethods, method);
    return said.hierarchy.reason(parentSourceLabel(reason), known ? methodWords[method] : "");
}

export function trailSourceLabel(source: string): string {
    return isOneOf(trailSources, source) ? trailWords[source] : source;
}

export function technicalReasonLabel(reason: string): string {
    return isOneOf(technicalReasons, reason) ? technicalWords[reason] : reason;
}

export function readOnlyReasonLabel(reason: string): string {
    if (reason === termReason) {
        return said.reasons.term;
    }
    if (reason.startsWith(builderPrefix)) {
        return said.reasons.builder(reason.slice(builderPrefix.length));
    }
    return reason;
}

export function rootStateLabel(state: string): string {
    return isOneOf(contentRootStates, state) ? rootWords[state] : state;
}

export function rootStateTone(state: string): Tone {
    return isOneOf(contentRootStates, state) ? rootTones[state] : "muted";
}

export function findingCodeLabel(code: string): string {
    return isOneOf(findingCodes, code) ? codeWords[code] : code;
}

export function severityLabel(severity: string): string {
    return isOneOf(findingSeverities, severity) ? severityWords[severity] : severity;
}

export function severityTone(severity: string): Tone {
    return isOneOf(findingSeverities, severity) ? severityTones[severity] : "muted";
}

export function urlFormLabel(form: string): string {
    if (form === "") {
        return said.render.noLinks;
    }
    return isOneOf(urlForms, form) ? urlFormWords[form] : form;
}

export function wpTypeLabel(type: string): string {
    return isOneOf(pageWpTypes, type) ? wpTypeWords[type] : type;
}

export function wpStatusLabel(status: string): string {
    return said.wpStatuses[status] ?? status;
}

export function taxonomyLabel(taxonomy: string): string {
    return said.taxonomies[taxonomy] ?? taxonomy;
}

export function blockKindLabel(kind: string): string {
    return said.blockKinds[kind] ?? kind;
}

export function bodyKindLabel(kind: string): string {
    return said.bodyKinds[kind] ?? kind;
}

export function fieldLabel(key: string): string {
    const base = key.split(":")[0] ?? key;
    return said.fieldNames[base] ?? said.unknownField;
}
