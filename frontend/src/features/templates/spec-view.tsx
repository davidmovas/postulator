import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { JsonObject } from "../../domain/merge-patch.js";
import { SectionLabel } from "../../ui/index.js";
import { decimal, flagLabel, imageSourceLabel, percent, stepLabel } from "./labels.js";
import type { SpecPath } from "./patch.js";
import { layerAt, paths, profilePath } from "./patch.js";
import { LayerBadge } from "./provenance.js";
import type { SpecDraft } from "./spec.js";

interface Row {
    key: string;
    label: string;
    value: string;
    path: SpecPath;
}

function rowsOf(draft: SpecDraft): Row[] {
    const rows: Row[] = [
        { key: "tone", label: copy.templates.content.tone, value: draft.tone === "" ? copy.templates.layer.none : draft.tone, path: paths.tone },
        { key: "lengthMin", label: copy.templates.content.lengthMin, value: String(draft.lengthMin), path: paths.lengthMin },
        { key: "lengthMax", label: copy.templates.content.lengthMax, value: String(draft.lengthMax), path: paths.lengthMax },
        { key: "primaryInTitle", label: copy.templates.content.primaryInTitle, value: flagLabel(draft.primaryInTitle), path: paths.primaryInTitle },
        { key: "primaryInH1", label: copy.templates.content.primaryInH1, value: flagLabel(draft.primaryInH1), path: paths.primaryInH1 },
        { key: "primaryInFirstParagraph", label: copy.templates.content.primaryInFirstParagraph, value: flagLabel(draft.primaryInFirstParagraph), path: paths.primaryInFirstParagraph },
        { key: "maxDensity", label: copy.templates.content.maxDensity, value: percent(draft.maxDensity), path: paths.maxDensity },
        { key: "upDepth", label: copy.templates.links.upDepth, value: String(draft.upDepth), path: paths.upDepth },
        { key: "downLinks", label: copy.templates.links.downLinks, value: flagLabel(draft.downLinks), path: paths.downLinks },
        { key: "siblingMinWeight", label: copy.templates.links.siblingMinWeight, value: decimal(draft.siblingMinWeight), path: paths.siblingMinWeight },
        { key: "maxLinks", label: copy.templates.links.maxLinks, value: String(draft.maxLinks), path: paths.maxLinks },
        { key: "maxPerTarget", label: copy.templates.links.maxPerTarget, value: String(draft.maxPerTarget), path: paths.maxPerTarget },
        { key: "parentLinkWithinParagraphs", label: copy.templates.links.parentLinkWithinParagraphs, value: String(draft.parentLinkWithinParagraphs), path: paths.parentLinkWithinParagraphs },
        { key: "childrenSection", label: copy.templates.links.childrenSection, value: flagLabel(draft.childrenSection), path: paths.childrenSection },
        { key: "titlePattern", label: copy.templates.meta.titlePattern, value: draft.titlePattern === "" ? copy.templates.layer.none : draft.titlePattern, path: paths.titlePattern },
        { key: "descriptionMax", label: copy.templates.meta.descriptionMax, value: String(draft.descriptionMax), path: paths.descriptionMax },
        { key: "featured", label: copy.templates.meta.featured, value: flagLabel(draft.featuredImage), path: paths.featuredImage },
        { key: "inline", label: copy.templates.meta.inline, value: String(draft.inlineImages), path: paths.inlineImages },
        { key: "source", label: copy.templates.meta.source, value: imageSourceLabel(draft.imageSource), path: paths.imageSource },
    ];
    for (const profile of draft.profiles) {
        rows.push({
            key: `role-${profile.role}`,
            label: profile.role,
            value: `${profile.provider} ${profile.model}`,
            path: profilePath(profile.role),
        });
    }
    return rows;
}

export interface SpecViewProps {
    draft: SpecDraft;
    site: JsonObject | null;
    page: JsonObject | null;
}

export function SpecView({ draft, site, page }: SpecViewProps): ReactElement {
    const enabled = draft.recipe.filter((step) => step.enabled);
    return (
        <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
                <div className="flex items-center justify-between gap-2">
                    <SectionLabel>{copy.templates.sections.title}</SectionLabel>
                    <LayerBadge layer={layerAt(paths.sections, site, page)} />
                </div>
                <ol className="flex flex-col gap-0.5">
                    {draft.sections.map((section, index) => (
                        <li key={index} className="flex gap-2 text-xs text-ink-soft">
                            <span className="w-4 shrink-0 text-right font-mono text-2xs text-ink-faint">
                                {index + 1}
                            </span>
                            <span className="min-w-0 truncate">
                                {section.heading === "" ? copy.templates.sections.untitled : section.heading}
                            </span>
                            <span className="ml-auto shrink-0 font-mono text-2xs text-ink-faint">
                                {section.targetWords}
                            </span>
                        </li>
                    ))}
                </ol>
            </div>
            <dl className="grid grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-x-3 gap-y-1">
                {rowsOf(draft).map((row) => (
                    <div key={row.key} className="contents">
                        <dt className="min-w-0 truncate text-xs text-ink-dim">{row.label}</dt>
                        <dd className="truncate font-mono text-xs text-ink">{row.value}</dd>
                        <dd>
                            <LayerBadge layer={layerAt(row.path, site, page)} />
                        </dd>
                    </div>
                ))}
            </dl>
            <div className="flex flex-col gap-1.5">
                <div className="flex items-center justify-between gap-2">
                    <SectionLabel>{copy.templates.recipe.title}</SectionLabel>
                    <LayerBadge layer={layerAt(paths.recipe, site, page)} />
                </div>
                <ol className="flex flex-col gap-0.5">
                    {enabled.map((step, index) => (
                        <li key={step.name} className="flex gap-2 text-xs text-ink-soft">
                            <span className="w-4 shrink-0 text-right font-mono text-2xs text-ink-faint">
                                {index + 1}
                            </span>
                            <span className="min-w-0 truncate">{stepLabel(step.name)}</span>
                        </li>
                    ))}
                </ol>
            </div>
        </div>
    );
}
