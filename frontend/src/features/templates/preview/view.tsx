import type { ReactElement } from "react";
import { useMemo } from "react";

import { copy } from "../../../copy/index.js";
import { AdsClickIcon, cx, DescriptionIcon, LinkIcon, PreviewIcon, StatusBadge } from "../../../ui/index.js";
import type { SpecDraft } from "../spec.js";
import type { Skeleton, SkeletonSection } from "./model.js";
import { skeletonOf } from "./model.js";

const minBlockPx = 20;
const blockRangePx = 120;

function KeywordMark(): ReactElement {
    return (
        <span className="inline-flex items-center gap-0.5 text-2xs text-accent" title={copy.templates.preview.keyword}>
            <AdsClickIcon size={11} />
        </span>
    );
}

function ImageSlot({ label }: { label: string }): ReactElement {
    return (
        <div className="flex h-12 items-center justify-center gap-1.5 rounded-sm border border-dashed border-edge text-2xs text-ink-faint">
            <PreviewIcon size={14} />
            {label}
        </div>
    );
}

function TextLines({ share, words }: { share: number; words: number }): ReactElement {
    const height = minBlockPx + Math.round(share * blockRangePx);
    const lines = Math.max(2, Math.floor(height / 8));
    return (
        <div className="relative flex flex-col gap-1" style={{ height: `${height}px` }} aria-label={copy.templates.preview.words(words)}>
            {Array.from({ length: lines }, (_unused, index) => (
                <span
                    key={index}
                    aria-hidden={true}
                    className="block h-1 rounded-full bg-raised-strong"
                    style={{ width: index === lines - 1 ? "58%" : `${88 + ((index * 7) % 12)}%` }}
                />
            ))}
            <span className="absolute right-0 -bottom-0.5 rounded-sm bg-panel px-1 font-mono text-2xs text-ink-faint">
                {copy.templates.preview.words(words)}
            </span>
        </div>
    );
}

function Section({ section, parentLink }: { section: SkeletonSection; parentLink: string | null }): ReactElement {
    return (
        <div className="flex flex-col gap-1.5">
            <div className="flex items-center gap-1.5">
                <span className={cx("h-2 flex-1 rounded-full", section.heading === "" ? "bg-warn-soft" : "bg-ink-dim")} style={{ maxWidth: "70%" }} />
                {section.keywordInHeading ? <KeywordMark /> : null}
                <span className="ml-auto truncate text-2xs text-ink-soft">{section.heading === "" ? copy.templates.preview.untitled : section.heading}</span>
                <span className={cx("shrink-0 text-2xs", section.required ? "text-ink-faint" : "text-ink-faint italic")}>
                    {section.required ? copy.templates.preview.required : copy.templates.preview.optional}
                </span>
            </div>
            {parentLink === null ? null : (
                <span className="inline-flex items-center gap-1 self-start rounded-sm bg-info-soft px-1.5 py-0.5 text-2xs text-info">
                    <LinkIcon size={11} />
                    {parentLink}
                </span>
            )}
            <TextLines share={section.share} words={section.words} />
            {section.imageAfter ? <ImageSlot label={copy.templates.preview.inline} /> : null}
        </div>
    );
}

function Footer({ skeleton }: { skeleton: Skeleton }): ReactElement {
    const said = copy.templates.preview;
    const tone = skeleton.lengthState === "within" || skeleton.lengthState === "unbounded" ? "muted" : "warn";
    return (
        <div className="flex flex-col gap-1 border-t border-hairline pt-2 text-2xs text-ink-faint">
            <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                <span className="font-mono text-ink-soft">{said.total(skeleton.totalWords)}</span>
                <span className="font-mono">
                    {skeleton.lengthState === "unbounded" ? said.unbounded : said.range(skeleton.lengthMin, skeleton.lengthMax)}
                </span>
                <span className="font-mono">{said.links(skeleton.maxLinks)}</span>
                <span className="font-mono">{said.images((skeleton.featured ? 1 : 0) + skeleton.inlineImages)}</span>
            </div>
            {tone === "warn" ? (
                <StatusBadge tone="warn" className="self-start">
                    {skeleton.lengthState === "under" ? said.under : said.over}
                </StatusBadge>
            ) : null}
        </div>
    );
}

export interface PagePreviewProps {
    draft: SpecDraft;
}

export function PagePreview({ draft }: PagePreviewProps): ReactElement {
    const said = copy.templates.preview;
    const skeleton = useMemo(() => skeletonOf(draft), [draft]);
    const parentLink = skeleton.upDepth === 0 ? null : skeleton.parentLinkWithin === 0 ? said.noParentLink : said.parentLink(skeleton.parentLinkWithin);

    return (
        <div className="flex flex-col gap-3">
            <div className="overflow-hidden rounded-lg border border-hairline bg-canvas">
                <div className="flex flex-col gap-1 border-b border-hairline bg-panel px-2.5 py-2">
                    <span className="text-2xs font-semibold tracking-label text-ink-faint uppercase">{said.tab}</span>
                    <div className="flex flex-wrap items-center gap-0.5 text-xs text-ink">
                        {skeleton.titleParts.length === 0 ? (
                            <span className="text-ink-faint">{copy.templates.layer.none}</span>
                        ) : (
                            skeleton.titleParts.map((part, index) =>
                                part.kind === "text" ? (
                                    <span key={index}>{part.text}</span>
                                ) : (
                                    <span key={index} className="rounded-sm bg-accent-soft px-1 font-mono text-2xs text-accent">
                                        {part.text}
                                    </span>
                                ),
                            )
                        )}
                        {skeleton.titleKeyword ? <KeywordMark /> : null}
                    </div>
                    <span className="text-2xs text-ink-faint">{said.description(skeleton.descriptionMax)}</span>
                </div>
                <div className="flex flex-col gap-3 p-3">
                    {skeleton.featured ? <ImageSlot label={said.featured} /> : null}
                    <div className="flex items-center gap-1.5">
                        <span className="h-3 w-3/4 rounded-full bg-ink" />
                        {skeleton.h1Keyword ? <KeywordMark /> : null}
                        <span className="ml-auto text-2xs text-ink-faint">{said.h1}</span>
                    </div>
                    {skeleton.sections.length === 0 ? (
                        <p className="flex items-center gap-1.5 text-xs text-ink-faint">
                            <DescriptionIcon size={14} />
                            {said.noSections}
                        </p>
                    ) : (
                        skeleton.sections.map((section, index) => (
                            <Section key={index} section={section} parentLink={index === 0 ? parentLink : null} />
                        ))
                    )}
                    {skeleton.childrenSection ? (
                        <div className="flex items-center gap-1.5 rounded-sm border border-hairline bg-inset px-2 py-1.5 text-2xs text-ink-soft">
                            <LinkIcon size={12} className="text-ink-faint" />
                            {said.children}
                        </div>
                    ) : null}
                </div>
            </div>
            <Footer skeleton={skeleton} />
        </div>
    );
}
