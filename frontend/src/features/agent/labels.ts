import { copy } from "../../copy/index.js";
import type { IconComponent, Tone } from "../../ui/index.js";
import {
    AccountTreeIcon,
    BuildIcon,
    DashboardCustomizeIcon,
    DescriptionIcon,
    HubIcon,
    LinkIcon,
    MonitoringIcon,
    PlayCircleIcon,
    PublicIcon,
    ScheduleIcon,
    Stars2Icon,
    SyncIcon,
    UploadFileIcon,
} from "../../ui/index.js";
import type { Family } from "./conversation/model/tools.js";
import type { ToolRowStatus } from "./conversation/model/transcript.js";

const familyIcons: Readonly<Record<Family, IconComponent>> = {
    sites: PublicIcon,
    graph: HubIcon,
    pages: AccountTreeIcon,
    templates: DashboardCustomizeIcon,
    policies: LinkIcon,
    runs: PlayCircleIcon,
    sync: SyncIcon,
    reports: MonitoringIcon,
    imports: UploadFileIcon,
    models: Stars2Icon,
    content: DescriptionIcon,
    schedules: ScheduleIcon,
    other: BuildIcon,
};

export function familyIcon(family: Family): IconComponent {
    return familyIcons[family];
}

export function familyLabel(family: Family): string {
    return copy.agent.families[family] ?? family;
}

export function toolStatusTone(status: ToolRowStatus): Tone {
    switch (status) {
        case "running":
            return "info";
        case "ok":
            return "ok";
        default:
            return "danger";
    }
}

export function toolStatusLabel(status: ToolRowStatus): string {
    switch (status) {
        case "running":
            return copy.agent.transcript.tool.running;
        case "ok":
            return copy.agent.transcript.tool.ok;
        default:
            return copy.agent.transcript.tool.error;
    }
}

export function riskTone(risk: string): Tone {
    switch (risk) {
        case "dangerous":
            return "danger";
        case "write":
            return "warn";
        default:
            return "muted";
    }
}

export function riskLabel(risk: string): string {
    switch (risk) {
        case "dangerous":
            return copy.agent.tools.dangerous;
        case "write":
            return copy.agent.tools.write;
        default:
            return copy.agent.tools.read;
    }
}

export function actionStatusTone(status: string): Tone {
    switch (status) {
        case "executed":
            return "ok";
        case "approved":
            return "info";
        case "failed":
            return "danger";
        case "rejected":
            return "muted";
        default:
            return "warn";
    }
}
