import { copy } from "../../copy/index.js";
import { actors, isOneOf, pageStatuses, publishModes } from "../../generated/vocab.js";

export function pageStatusLabel(status: string): string {
    return isOneOf(pageStatuses, status) ? copy.schedules.statuses[status] : status;
}

export function publishLabel(mode: string): string {
    return isOneOf(publishModes, mode) ? copy.schedules.publishModes[mode] : mode;
}

export function actorLabel(actor: string): string {
    return isOneOf(actors, actor) ? copy.schedules.panel.actors[actor] : actor;
}

export function targetsSentence(status: string, limit: number, entityName: string): string {
    const capped = limit > 0 ? limit : 500;
    const named = isOneOf(pageStatuses, status) ? copy.schedules.statuses[status].toLowerCase() : "";
    if (named !== "" && entityName !== "") {
        return copy.schedules.targets.both(capped, named, entityName);
    }
    if (named !== "") {
        return copy.schedules.targets.status(capped, named);
    }
    if (entityName !== "") {
        return copy.schedules.targets.entity(capped, entityName);
    }
    return copy.schedules.targets.any(capped);
}
