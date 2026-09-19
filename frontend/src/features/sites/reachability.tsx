import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { Reachability } from "../../data/types.js";
import {
    Banner,
    Button,
    CloudDoneIcon,
    CloudOffIcon,
    KeyOffIcon,
    ShieldIcon,
} from "../../ui/index.js";
import type { IconComponent, Tone } from "../../ui/index.js";

export const reachValues = ["ok", "upgradeRequired", "unauthorized", "unreachable"] as const;

export type Reach = (typeof reachValues)[number];

const tones: Readonly<Record<Reach, Tone>> = {
    ok: "ok",
    upgradeRequired: "warn",
    unauthorized: "danger",
    unreachable: "danger",
};

const icons: Readonly<Record<Reach, IconComponent>> = {
    ok: CloudDoneIcon,
    upgradeRequired: ShieldIcon,
    unauthorized: KeyOffIcon,
    unreachable: CloudOffIcon,
};

function reachOf(raw: string): Reach | null {
    return (reachValues as readonly string[]).includes(raw) ? (raw as Reach) : null;
}

export interface ReachabilityReportProps {
    result: Reachability;
    onUseSuggested?: (baseUrl: string) => void;
}

export function ReachabilityReport({ result, onUseSuggested }: ReachabilityReportProps): ReactElement {
    const reach = reachOf(result.reach);
    const tone: Tone = reach === null ? "warn" : tones[reach];
    const icon = reach === null ? undefined : icons[reach];
    const title = reach === null ? result.message : copy.sites.reach[reach];

    const notes: string[] = [];
    if (reach !== null && result.message !== "" && result.message !== title) {
        notes.push(result.message);
    }
    if (result.siteName !== "") {
        notes.push(copy.sites.detected(result.siteName));
    }
    if (result.hasWoo) {
        notes.push(copy.sites.woo);
    }
    if (result.hasPlugin) {
        notes.push(copy.sites.plugin.installed(""));
    }

    const canUpgrade =
        reach === "upgradeRequired" && result.suggestedBaseUrl !== "" && onUseSuggested !== undefined;

    return (
        <Banner
            tone={tone}
            icon={icon}
            title={title}
            body={notes.length === 0 ? undefined : notes.join(" · ")}
            actions={
                canUpgrade ? (
                    <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => {
                            onUseSuggested(result.suggestedBaseUrl);
                        }}
                    >
                        {copy.sites.useSuggested(result.suggestedBaseUrl)}
                    </Button>
                ) : undefined
            }
        />
    );
}
