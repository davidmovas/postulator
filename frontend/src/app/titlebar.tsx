import type { MouseEvent, ReactElement } from "react";

import { copy } from "../copy/index.js";
import { close, minimise, toggleMaximise, useMaximised } from "../data/window.js";
import { openPalette } from "../features/palette/index.js";
import {
    IconButton,
    Kbd,
    RightPanelOpenIcon,
    SearchIcon,
    WindowControls,
    dragRegion,
    noDrag,
} from "../ui/index.js";
import { SitePill } from "./site-pill.js";

export interface TitleBarProps {
    siteId: string | null;
    dockOpen: boolean;
    onToggleDock: () => void;
}

function onBareBar(event: MouseEvent<HTMLElement>): boolean {
    const target = event.target;
    if (!(target instanceof Element)) {
        return false;
    }
    return target.closest("button, a, input, [role='button']") === null;
}

export function TitleBar({ siteId, dockOpen, onToggleDock }: TitleBarProps): ReactElement {
    const maximised = useMaximised();

    return (
        <header
            style={dragRegion}
            onDoubleClick={(event) => {
                if (onBareBar(event)) {
                    toggleMaximise();
                }
            }}
            className="flex h-10 shrink-0 items-center gap-3 border-b border-hairline bg-panel pl-3"
        >
            <SitePill siteId={siteId} />
            <div className="flex min-w-0 flex-1 justify-center">
                <button
                    type="button"
                    style={noDrag}
                    onClick={openPalette}
                    aria-keyshortcuts="F"
                    className="flex h-7 w-80 max-w-full items-center gap-2 rounded-md border border-hairline bg-inset px-2.5 text-left transition-colors duration-100 ease-out hover:border-edge"
                >
                    <SearchIcon size={15} className="shrink-0 text-ink-faint" />
                    <span className="min-w-0 flex-1 truncate text-sm text-ink-faint">{copy.palette.field}</span>
                    <Kbd keys={["F"]} />
                </button>
            </div>
            <div style={noDrag} className="flex shrink-0 items-center">
                {dockOpen ? null : (
                    <IconButton
                        icon={RightPanelOpenIcon}
                        label={copy.shell.expandDock}
                        variant="ghost"
                        size="sm"
                        onClick={onToggleDock}
                        className="mr-2"
                    />
                )}
                <WindowControls
                    maximised={maximised}
                    minimiseLabel={copy.shell.minimise}
                    maximiseLabel={copy.shell.maximise}
                    restoreLabel={copy.shell.restore}
                    closeLabel={copy.shell.close}
                    onMinimise={minimise}
                    onToggleMaximise={toggleMaximise}
                    onClose={close}
                />
            </div>
        </header>
    );
}
