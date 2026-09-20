import { useNavigate } from "react-router";

import { copy } from "../copy/index.js";
import { flatten } from "../data/call.js";
import { useSites } from "../data/hooks/sites.js";
import { Select } from "../ui/index.js";
import type { SelectOption } from "../ui/index.js";

export interface TitleBarProps {
    siteId: string | null;
}

export function TitleBar({ siteId }: TitleBarProps) {
    const navigate = useNavigate();
    const sites = useSites();
    const rows = flatten(sites.data?.pages);
    const current = rows.find((site) => site.id === siteId) ?? null;
    const options: SelectOption<string>[] = rows.map((site) => ({ value: site.id, label: site.name }));

    return (
        <header className="flex h-9 shrink-0 items-center gap-3 border-b border-hairline bg-panel px-3">
            <div className="flex shrink-0 items-center gap-2">
                <img src="/appmark.svg" alt="" width={20} height={20} className="h-5 w-5 rounded-md" />
                <span className="text-lg font-semibold tracking-tight text-ink">{copy.app.name}</span>
            </div>
            <span aria-hidden={true} className="h-4 w-px shrink-0 bg-hairline" />
            <div className="w-64 shrink-0">
                <Select
                    aria-label={copy.shell.siteSwitcher}
                    value={current === null ? null : current.id}
                    placeholder={copy.shell.selectSite}
                    options={options}
                    onValueChange={(picked) => {
                        void navigate(`/s/${picked}/overview`);
                    }}
                />
            </div>
            {current === null ? null : (
                <span className="min-w-0 truncate font-mono text-2xs text-ink-faint">{current.baseUrl}</span>
            )}
        </header>
    );
}
