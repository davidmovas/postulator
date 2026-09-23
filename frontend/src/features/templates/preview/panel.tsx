import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import type { SpecDraft } from "../spec.js";
import { PagePreview } from "./view.js";

export interface SkeletonPanelProps {
    draft: SpecDraft;
}

export function SkeletonPanel({ draft }: SkeletonPanelProps): ReactElement {
    return (
        <div className="flex flex-col">
            <header className="sticky top-0 z-10 flex h-8 shrink-0 items-center justify-between gap-2 border-b border-hairline bg-panel px-3">
                <h2 className="text-2xs font-semibold tracking-label text-ink-faint uppercase">
                    {copy.templates.editor.skeleton}
                </h2>
                <span className="font-mono text-2xs text-ink-faint">
                    {copy.templates.wordRange(draft.lengthMin, draft.lengthMax)}
                </span>
            </header>
            <div className="p-3">
                <PagePreview draft={draft} />
            </div>
        </div>
    );
}
