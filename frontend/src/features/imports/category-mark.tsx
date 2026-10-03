import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { cx, toneClasses } from "../../ui/index.js";

export function CategoryMark(): ReactElement {
    return (
        <span
            data-site-category={true}
            title={copy.imports.preview.siteCategoryHint}
            className={cx(
                "inline-flex h-4 shrink-0 items-center rounded-sm border px-1 text-2xs font-medium whitespace-nowrap",
                toneClasses.info.soft,
                toneClasses.info.border,
                toneClasses.info.ink,
            )}
        >
            {copy.imports.preview.siteCategory}
        </span>
    );
}
