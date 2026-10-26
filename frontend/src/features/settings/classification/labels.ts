import { copy } from "../../../copy/index.js";
import type { RuleVerdict } from "../../../generated/vocab.js";
import { isOneOf, pageWpTypes, ruleConditions, ruleVerdicts, schemaTypes, urlModes } from "../../../generated/vocab.js";
import type { Tone } from "../../../ui/index.js";

const said = copy.classification;

const verdictTones: Readonly<Record<RuleVerdict, Tone>> = {
    met: "ok",
    unmet: "muted",
    waiting: "info",
    blocked: "warn",
};

export function verdictLabel(verdict: string): string {
    return isOneOf(ruleVerdicts, verdict) ? said.verdicts[verdict] : verdict;
}

export function verdictTone(verdict: string): Tone {
    return isOneOf(ruleVerdicts, verdict) ? verdictTones[verdict] : "muted";
}

export function conditionLabel(condition: string): string {
    return isOneOf(ruleConditions, condition) ? said.conditions[condition] : condition;
}

export function urlModeLabel(mode: string): string {
    return isOneOf(urlModes, mode) ? said.urlModes[mode] : mode;
}

export function urlModeHint(mode: string): string {
    return isOneOf(urlModes, mode) ? said.urlHints[mode] : "";
}

export function schemaWord(schema: string): string {
    return isOneOf(schemaTypes, schema) ? said.schema[schema] : schema;
}

export function schemaOption(schema: string): string {
    return isOneOf(schemaTypes, schema) ? said.schemaOptions[schema] : schema;
}

export function postTypeWord(postType: string): string {
    return isOneOf(pageWpTypes, postType) ? said.postTypes[postType] : postType;
}

export function postTypeOption(postType: string): string {
    return isOneOf(pageWpTypes, postType) ? said.postTypeOptions[postType] : postType;
}
