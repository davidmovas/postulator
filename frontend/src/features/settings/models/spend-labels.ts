import { copy } from "../../../copy/index.js";
import type { Code } from "../../../data/errors.js";
import { codes } from "../../../data/errors.js";
import type { ServiceTier, SpendPurpose } from "../../../generated/vocab.js";
import { isOneOf, serviceTiers, spendPurposes } from "../../../generated/vocab.js";
import { stepLabel } from "../../runs/labels.js";

const said = copy.settings.models.spend;

const proposalSteps = ["propose_from_pages", "propose_from_keywords", "propose_related"] as const;

type ProposalStep = (typeof proposalSteps)[number];

const proposalWords: Readonly<Record<ProposalStep, string>> = said.recent.proposals;

const failureWords: Readonly<Record<Code, string>> = said.failures;

export const spendDays = [7, 30, 90] as const;

export type SpendDays = (typeof spendDays)[number];

export const defaultSpendDays: SpendDays = 30;

export function daysOf(picked: string): SpendDays {
    return spendDays.find((days) => String(days) === picked) ?? defaultSpendDays;
}

const purposeWords: Readonly<Record<SpendPurpose, string>> = said.purposes;

export function purposeLabel(purpose: string): string {
    return isOneOf(spendPurposes, purpose) ? purposeWords[purpose] : purposeWords.other;
}

const tierWords: Readonly<Record<ServiceTier, string>> = said.tiers;

export function tierLabel(tier: string): string {
    return isOneOf(serviceTiers, tier) ? tierWords[tier] : tier;
}

export function failureText(code: string): string {
    return isOneOf(codes, code) ? failureWords[code] : said.failedUnknown;
}

export function callStep(purpose: string, step: string): string {
    if (step === "") {
        return "";
    }
    switch (purpose) {
        case "run":
        case "other":
            return stepLabel(step);
        case "graph":
            return isOneOf(proposalSteps, step) ? proposalWords[step] : "";
        default:
            return "";
    }
}
