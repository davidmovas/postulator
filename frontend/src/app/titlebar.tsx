import { useNavigate } from "react-router";

import { copy } from "../copy/index.js";
import { flatten } from "../data/call.js";
import { useSites } from "../data/hooks/sites.js";

export interface TitleBarProps {
    siteId: string | null;
}

export function TitleBar({ siteId }: TitleBarProps) {
    const navigate = useNavigate();
    const sites = useSites();
    const rows = flatten(sites.data?.pages);

    return (
        <header className="flex h-9 shrink-0 items-center gap-3 border-b border-base-700 bg-base-900 px-3">
            <span className="text-ink-100 text-sm font-semibold tracking-tight">{copy.app.name}</span>
            <label className="sr-only" htmlFor="site-switcher">
                {copy.shell.siteSwitcher}
            </label>
            <select
                id="site-switcher"
                className="h-6 rounded-panel border border-base-600 bg-base-800 px-2 text-xs text-ink-200"
                value={siteId ?? ""}
                onChange={(event) => {
                    const picked = event.target.value;
                    void navigate(picked === "" ? "/sites" : `/s/${picked}/overview`);
                }}
            >
                <option value="">{copy.shell.noSiteSelected}</option>
                {rows.map((site) => (
                    <option key={site.id} value={site.id}>
                        {site.name}
                    </option>
                ))}
            </select>
        </header>
    );
}
