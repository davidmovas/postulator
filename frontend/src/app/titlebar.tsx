import { useNavigate } from "react-router";

import { copy } from "../copy/index.js";
import { flatten } from "../data/call.js";
import { useSites } from "../data/hooks/sites.js";
import { Select } from "../ui/index.js";
import type { SelectOption } from "../ui/index.js";

const noSite = "no-site";

export interface TitleBarProps {
    siteId: string | null;
}

export function TitleBar({ siteId }: TitleBarProps) {
    const navigate = useNavigate();
    const sites = useSites();
    const rows = flatten(sites.data?.pages);

    const options: SelectOption<string>[] = [
        { value: noSite, label: copy.shell.noSiteSelected },
        ...rows.map((site) => ({ value: site.id, label: site.name })),
    ];

    return (
        <header className="flex h-9 shrink-0 items-center gap-3 border-b border-hairline bg-panel px-3">
            <div className="flex items-center gap-2">
                <span className="flex h-5 w-5 items-center justify-center rounded-md bg-accent text-2xs font-bold text-on-accent">
                    P
                </span>
                <span className="text-lg font-semibold tracking-tight text-ink">{copy.app.name}</span>
            </div>
            <Select
                aria-label={copy.shell.siteSwitcher}
                className="w-56"
                value={siteId ?? noSite}
                options={options}
                onValueChange={(picked) => {
                    void navigate(picked === noSite ? "/sites" : `/s/${picked}/overview`);
                }}
            />
        </header>
    );
}
