import { copy } from "../../copy/index.js";
import type { FindingOption } from "../../generated/vocab.js";
import { findingOptions, isOneOf } from "../../generated/vocab.js";

const said = copy.findings;

const optionWords: Readonly<Record<FindingOption, string>> = said.options;
const optionSentences: Readonly<Record<FindingOption, string>> = said.optionDoes;

export function optionLabel(option: string): string {
    return isOneOf(findingOptions, option) ? optionWords[option] : option;
}

export function optionDoes(option: string): string {
    return isOneOf(findingOptions, option) ? optionSentences[option] : "";
}
