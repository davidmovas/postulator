import type { ReactElement } from "react";
import { useNavigate } from "react-router";

import { copy } from "../copy/index.js";
import { flatten } from "../data/call.js";
import { useSites } from "../data/hooks/sites.js";
import { openPalette } from "../features/palette/index.js";
import { IconButton, Kbd, RightPanelOpenIcon, SearchIcon, Select } from "../ui/index.js";
import type { SelectOption } from "../ui/index.js";

export interface TitleBarProps {
    siteId: string | null;
    dockOpen: boolean;
    onToggleDock: () => void;
}

export function TitleBar({ siteId, dockOpen, onToggleDock }: TitleBarProps): ReactElement {
    const navigate = useNavigate();
    const sites = useSites();
    const rows = flatten(sites.data?.pages);
    const current = rows.find((site) => site.id === siteId) ?? null;
    const options: SelectOption<string>[] = rows.map((site) => ({ value: site.id, label: site.name }));

    return (
        <header className="flex h-10 shrink-0 items-center gap-3 border-b border-hairline bg-panel px-3">
            <div className="flex shrink-0 items-center gap-2">
                <img src="/appmark.svg" alt="" width={20} height={20} className="h-5 w-5 rounded-md" />
                <span className="text-sm font-semibold tracking-tight text-ink">{copy.app.name}</span>
            </div>
            <span aria-hidden={true} className="h-4 w-px shrink-0 bg-hairline" />
            <div className="w-56 shrink-0">
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
                <span className="min-w-0 shrink truncate font-mono text-2xs text-ink-faint">
                    {current.baseUrl}
                </span>
            )}
            <div className="flex min-w-0 flex-1 justify-center">
                <button
                    type="button"
                    onClick={openPalette}
                    aria-keyshortcuts="Control+K"
                    className="flex h-7 w-80 max-w-full items-center gap-2 rounded-md border border-hairline bg-inset px-2.5 text-left transition-colors duration-100 ease-out hover:border-edge"
                >
                    <SearchIcon size={15} className="shrink-0 text-ink-faint" />
                    <span className="min-w-0 flex-1 truncate text-sm text-ink-faint">
                        {copy.palette.placeholder}
                    </span>
                    <Kbd keys={["Ctrl", "K"]} />
                </button>
            </div>
            {dockOpen ? null : (
                <IconButton
                    icon={RightPanelOpenIcon}
                    label={copy.shell.expandDock}
                    variant="ghost"
                    size="sm"
                    onClick={onToggleDock}
                />
            )}
        </header>
    );
}
