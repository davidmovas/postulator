import type { ReactElement } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useSites } from "../../data/hooks/sites.js";
import { Button, PublicIcon } from "../../ui/index.js";
import { ReadinessChecklist } from "./checklist.js";

export function OnboardingScreen(): ReactElement {
    const sites = useSites();
    const first = flatten(sites.data?.pages)[0] ?? null;

    return (
        <div className="flex flex-col gap-4 p-6">
            <header className="flex items-start justify-between gap-4">
                <div className="flex min-w-0 flex-col gap-1">
                    <h1 className="text-xl font-semibold tracking-tight text-ink">
                        {copy.onboarding.title}
                    </h1>
                    <p className="max-w-2xl text-sm text-ink-soft">{copy.onboarding.subtitle}</p>
                </div>
                {first === null ? (
                    <Link to="/sites">
                        <Button variant="primary" icon={PublicIcon}>
                            {copy.sites.add}
                        </Button>
                    </Link>
                ) : (
                    <Link to={`/s/${first.id}/overview`}>
                        <Button variant="secondary" icon={PublicIcon}>
                            {first.name}
                        </Button>
                    </Link>
                )}
            </header>
            <ReadinessChecklist siteId={null} className="max-w-4xl" />
        </div>
    );
}
