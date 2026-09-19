import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import {
    AddIcon,
    cx,
    FitScreenIcon,
    IconButton,
    PolylineIcon,
    RemoveIcon,
    UnfoldLessIcon,
    UnfoldMoreIcon,
} from "../../../ui/index.js";
import { useZoomPercent } from "../state.js";

export interface ControlsProps {
    siteId: string;
    showRelated: boolean;
    onZoom: (factor: number) => void;
    onFit: () => void;
    onExpandAll: () => void;
    onCollapse: () => void;
    onToggleRelated: () => void;
}

export function Controls({ siteId, showRelated, onZoom, onFit, onExpandAll, onCollapse, onToggleRelated }: ControlsProps): ReactElement {
    const percent = useZoomPercent(siteId);
    return (
        <div className="absolute top-3 right-3 flex flex-col items-end gap-2">
            <div className="flex items-center rounded-md border border-hairline bg-panel/90 backdrop-blur">
                <IconButton icon={RemoveIcon} label={copy.graph.controls.zoomOut} variant="ghost" onClick={() => onZoom(1 / 1.25)} />
                <button
                    type="button"
                    className="h-7 min-w-11 px-1 font-mono text-2xs text-ink-soft hover:text-ink"
                    title={copy.graph.controls.fit}
                    onClick={onFit}
                >
                    {copy.graph.controls.zoom(percent)}
                </button>
                <IconButton icon={AddIcon} label={copy.graph.controls.zoomIn} variant="ghost" onClick={() => onZoom(1.25)} />
                <span className="h-4 w-px bg-hairline" aria-hidden={true} />
                <IconButton icon={FitScreenIcon} label={copy.graph.controls.fit} variant="ghost" onClick={onFit} />
            </div>
            <div className="flex items-center rounded-md border border-hairline bg-panel/90 backdrop-blur">
                <IconButton icon={UnfoldMoreIcon} label={copy.graph.controls.expandAll} variant="ghost" onClick={onExpandAll} />
                <IconButton icon={UnfoldLessIcon} label={copy.graph.controls.collapseToRoots} variant="ghost" onClick={onCollapse} />
                <span className="h-4 w-px bg-hairline" aria-hidden={true} />
                <IconButton
                    icon={PolylineIcon}
                    label={showRelated ? copy.graph.controls.hideRelated : copy.graph.controls.showRelated}
                    variant="ghost"
                    aria-pressed={showRelated}
                    className={cx(showRelated && "text-info")}
                    onClick={onToggleRelated}
                />
            </div>
        </div>
    );
}
