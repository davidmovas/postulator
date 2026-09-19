import { copy } from "../../copy/index.js";
import type { IconComponent, Tone } from "../../ui/index.js";
import { ArrowDownwardIcon, ArrowUpwardIcon, SyncAltIcon } from "../../ui/index.js";
import type { Severity } from "./model/audit.js";
import type { Show } from "./model/params.js";

export function relationIcon(relation: string): IconComponent {
    switch (relation) {
        case "up":
            return ArrowUpwardIcon;
        case "down":
            return ArrowDownwardIcon;
        default:
            return SyncAltIcon;
    }
}

export function relationLabel(relation: string): string {
    return copy.links.relations[relation] ?? relation;
}

export function classTone(kind: string): Tone {
    switch (kind) {
        case "graph":
            return "ok";
        case "self":
            return "muted";
        case "external":
            return "info";
        default:
            return "warn";
    }
}

export function classLabel(kind: string): string {
    return copy.links.classes[kind] ?? kind;
}

export function severityTone(severity: Severity): Tone {
    return severity;
}

export function showLabel(show: Show): string {
    return copy.links.show[show];
}
