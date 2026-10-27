import { copy } from "../../copy/index.js";
import type { ParentWhy } from "../../data/types.js";
import type { BodyState, ParentWhyReason } from "../../generated/vocab.js";
import { bodyStates, isOneOf, parentWhyReasons } from "../../generated/vocab.js";
import type { Tone } from "../../ui/index.js";

const said = copy.map;

const whyWords: Readonly<Record<ParentWhyReason, string>> = said.why.labels;
const whySentences: Readonly<Record<ParentWhyReason, (why: ParentWhy) => string>> = said.why.sentences;

export function whyLabel(why: ParentWhy | null | undefined): string {
    if (why === null || why === undefined) {
        return said.why.none;
    }
    if (!isOneOf(parentWhyReasons, why.reason)) {
        return said.why.none;
    }
    return why.reason === "root" ? whyWords.root : `${said.why.none} · ${whyWords[why.reason]}`;
}

export function whySentence(why: ParentWhy | null | undefined): string | null {
    if (why === null || why === undefined || !isOneOf(parentWhyReasons, why.reason)) {
        return null;
    }
    return whySentences[why.reason](why);
}

const bodyWords: Readonly<Record<BodyState, string>> = said.bodyStates;
const bodyTooltips: Readonly<Record<BodyState, string>> = said.bodyTooltips;

const bodyTones: Readonly<Record<BodyState, Tone>> = {
    empty: "muted",
    client: "info",
    ours: "ok",
    mixed: "warn",
    dynamic: "muted",
};

export function bodyStateLabel(state: string): string {
    return isOneOf(bodyStates, state) ? bodyWords[state] : state;
}

export function bodyStateTooltip(state: string): string {
    return isOneOf(bodyStates, state) ? bodyTooltips[state] : "";
}

export function bodyStateTone(state: string): Tone {
    return isOneOf(bodyStates, state) ? bodyTones[state] : "muted";
}
