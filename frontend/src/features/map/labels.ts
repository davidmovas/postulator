import { copy } from "../../copy/index.js";
import type { BodyState } from "../../generated/vocab.js";
import { bodyStates, isOneOf } from "../../generated/vocab.js";
import type { Tone } from "../../ui/index.js";

const said = copy.map;

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
