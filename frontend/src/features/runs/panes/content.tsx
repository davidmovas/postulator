import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { HtmlPreview, SectionLabel } from "../../../ui/index.js";
import { draftView, imagesView, metaView } from "../artifacts.js";
import { Rows, Unreadable } from "./shared.js";

export interface PayloadPaneProps {
    payload: unknown;
}

export function DraftPane({ payload }: PayloadPaneProps): ReactElement {
    const view = draftView(payload);
    if (view === null) {
        return <Unreadable />;
    }
    return (
        <div className="flex flex-col gap-2 pb-3">
            <Rows
                entries={[
                    [copy.pages.detail.title, view.title],
                    [copy.pages.detail.h1, view.h1],
                    [copy.runs.review.draft.summary, view.summary],
                ]}
            />
            <SectionLabel className="px-3">{copy.runs.review.draft.sections}</SectionLabel>
            <ul className="flex flex-col">
                {view.sections.map((section, position) => (
                    <li
                        key={`${section.heading}:${String(position)}`}
                        className="border-b border-inset px-3 py-1.5 text-xs text-ink-soft last:border-b-0"
                    >
                        {section.heading}
                    </li>
                ))}
            </ul>
        </div>
    );
}

export function MetaPane({ payload }: PayloadPaneProps): ReactElement {
    const view = metaView(payload);
    if (view === null) {
        return <Unreadable />;
    }
    return (
        <Rows
            entries={[
                [copy.pages.detail.metaTitle, view.title],
                [copy.pages.detail.metaDescription, view.description],
                [copy.pages.detail.canonicalUrl, view.canonical],
                ["og:title", view.ogTitle],
                ["og:description", view.ogDescription],
            ]}
        />
    );
}

export function ImagesPane({ payload }: PayloadPaneProps): ReactElement {
    const view = imagesView(payload);
    if (view === null) {
        return <Unreadable />;
    }
    if (view.images.length === 0) {
        return <p className="px-3 py-2 text-xs text-ink-dim">{copy.runs.review.images.none}</p>;
    }
    return (
        <div className="flex flex-col gap-2 pb-3">
            <ul className="flex flex-col">
                {view.images.map((image) => (
                    <li
                        key={image.url}
                        className="flex items-center gap-2 border-b border-inset px-3 py-1.5 text-xs last:border-b-0"
                    >
                        <span className="w-20 shrink-0 text-2xs text-ink-faint">{image.role}</span>
                        <span className="min-w-0 flex-1 truncate font-mono text-ink-soft">{image.url}</span>
                        <span className="w-40 shrink-0 truncate text-2xs text-ink-faint">{image.alt}</span>
                    </li>
                ))}
            </ul>
            <Rows
                entries={[
                    [copy.runs.review.images.featured, view.featuredId === null ? "" : String(view.featuredId)],
                    [copy.runs.review.images.skipped, view.skipped.join(", ")],
                ]}
            />
        </div>
    );
}

export interface BodyPaneProps {
    html: string;
}

export function BodyPane({ html }: BodyPaneProps): ReactElement {
    if (html === "") {
        return <p className="px-3 py-2 text-xs text-ink-dim">{copy.runs.review.body.empty}</p>;
    }
    return (
        <div className="min-h-0 flex-1">
            <HtmlPreview bodyHtml={html} title={copy.runs.review.body.preview} />
        </div>
    );
}
