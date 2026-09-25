import { useMemo, useState } from "react";

import { useTemplate } from "../../data/hooks/templates.js";
import type { Template, TemplateOverride } from "../../data/types.js";
import type { ConflictKind, Stamp } from "./conflict.js";
import { beneathOf, conflictOf } from "./conflict.js";
import type { GroupKey } from "./outline.js";
import { changedGroups } from "./outline.js";
import type { Layer } from "./patch.js";
import { layered, patchBetween, patchObject } from "./patch.js";
import type { SpecDraft } from "./spec.js";
import { draftOf, readyForImages } from "./spec.js";

export interface LayerState {
    template: Template | undefined;
    pending: boolean;
    error: unknown;
    base: SpecDraft | null;
    siteResolved: SpecDraft | null;
    start: SpecDraft | null;
    below: SpecDraft | null;
    siteOverride: TemplateOverride | null;
    pageOverride: TemplateOverride | null;
    pageOverrides: readonly TemplateOverride[];
    stamp: Stamp;
}

export function useLayerState(
    templateId: string,
    siteId: string,
    pageId: string | null,
    layer: Layer,
): LayerState {
    const detail = useTemplate(templateId);
    const template = detail.data?.template;
    const overrides = useMemo(() => detail.data?.overrides ?? [], [detail.data]);
    const siteOverride = useMemo(
        () => overrides.find((held) => held.scope === "site" && held.targetId === siteId) ?? null,
        [overrides, siteId],
    );
    const pageOverride = useMemo(
        () =>
            pageId === null
                ? null
                : (overrides.find((held) => held.scope === "page" && held.targetId === pageId) ?? null),
        [overrides, pageId],
    );
    const pageOverrides = useMemo(() => overrides.filter((held) => held.scope === "page"), [overrides]);
    const sitePatch = useMemo(() => patchObject(siteOverride?.patch), [siteOverride]);
    const pagePatch = useMemo(() => patchObject(pageOverride?.patch), [pageOverride]);
    const base = useMemo(() => (template === undefined ? null : draftOf(template.spec)), [template]);
    const siteResolved = useMemo(() => (base === null ? null : layered(base, sitePatch)), [base, sitePatch]);
    const start = useMemo(() => {
        if (base === null || siteResolved === null) {
            return null;
        }
        if (layer === "global") {
            return base;
        }
        return layer === "site" ? siteResolved : layered(siteResolved, pagePatch);
    }, [base, siteResolved, layer, pagePatch]);

    return {
        template,
        pending: detail.isPending,
        error: detail.error,
        base,
        siteResolved,
        start,
        below: layer === "page" ? siteResolved : base,
        siteOverride,
        pageOverride,
        pageOverrides,
        stamp: {
            beneath: beneathOf(layer === "page" ? siteResolved : base),
            overrideUpdatedAt: (layer === "page" ? pageOverride?.updatedAt : siteOverride?.updatedAt) ?? null,
        },
    };
}

export interface DraftState {
    draft: SpecDraft | null;
    dirty: boolean;
    marks: ReadonlySet<GroupKey>;
    conflict: ConflictKind;
    edit: (patch: Partial<SpecDraft>) => void;
    clear: () => void;
}

export function useDraft(key: string, state: LayerState, layer: Layer): DraftState {
    const [edited, setEdited] = useState<{ key: string; opened: Stamp; draft: SpecDraft } | null>(null);
    const held = edited !== null && edited.key === key ? edited : null;
    const draft = held?.draft ?? state.start;
    const dirty = state.start !== null && draft !== null && patchBetween(state.start, draft) !== undefined;
    const marks = useMemo(
        () =>
            changedGroups(
                state.below === null || draft === null ? null : patchObject(patchBetween(state.below, draft)),
            ),
        [state.below, draft],
    );

    return {
        draft,
        dirty,
        marks,
        conflict: conflictOf(layer, held?.opened ?? null, state.stamp, dirty),
        edit: (patch) => {
            if (draft === null) {
                return;
            }
            setEdited({ key, opened: held?.opened ?? state.stamp, draft: readyForImages(draft, { ...draft, ...patch }) });
        },
        clear: () => {
            setEdited(null);
        },
    };
}
