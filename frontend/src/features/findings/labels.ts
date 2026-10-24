import { copy } from "../../copy/index.js";
import type { FindingOption, FindingStatus } from "../../generated/vocab.js";
import { findingOptions, findingStatuses, isOneOf } from "../../generated/vocab.js";

const said = copy.findings;

const optionWords: Readonly<Record<FindingOption, string>> = said.options;
const optionSentences: Readonly<Record<FindingOption, string>> = said.optionDoes;
const statusWords: Readonly<Record<FindingStatus, string>> = said.statuses;

export function optionLabel(option: string): string {
    return isOneOf(findingOptions, option) ? optionWords[option] : option;
}

export function optionDoes(option: string): string {
    return isOneOf(findingOptions, option) ? optionSentences[option] : "";
}

export function statusLabel(status: string): string {
    return isOneOf(findingStatuses, status) ? statusWords[status] : status;
}

export function resolvedByLabel(by: string): string {
    return said.resolvedBy[by] ?? said.someone;
}

export function resolutionLabel(resolution: string): string {
    return said.resolutions[resolution] ?? said.unknownResolution;
}
