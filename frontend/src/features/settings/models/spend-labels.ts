import { copy } from "../../../copy/index.js";
import type { ServiceTier, SpendPurpose } from "../../../generated/vocab.js";
import { isOneOf, serviceTiers, spendPurposes } from "../../../generated/vocab.js";

const said = copy.settings.models.spend;

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
