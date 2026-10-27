import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import {
    AddLinkIcon,
    Button,
    ChevronRightIcon,
    ChipInput,
    CloseIcon,
    cx,
    Field,
    IconButton,
    Input,
    PreviewIcon,
    Textarea,
} from "../../ui/index.js";
import { PageSearch } from "../findings/page-search.js";
import type { DirectiveDraft, DirectiveProblem, Directives, MustLinkDraft } from "./start-directions.js";
import { directiveProblems, directiveSize, emptyDirective, limits } from "./start-directions.js";

const said = copy.runs.start.directives;

const shownAtFirst = 12;

function countTone(failed: boolean, size: number): string {
    if (failed) {
        return "text-danger";
    }
    return size === 0 ? "text-ink-faint" : "text-accent";
}

export interface DirectedPage {
    id: string;
    path: string;
}

interface ListFieldProps {
    label: string;
    hint: string;
    values: readonly string[];
    most: number;
    failed: boolean;
    data: string;
    onChange: (next: string[]) => void;
}

function ListField({ label, hint, values, most, failed, data, onChange }: ListFieldProps): ReactElement {
    return (
        <Field label={label} hint={hint} error={failed ? said.tooMany(most) : null}>
            {(control) => (
                <div data-run-directive={data}>
                    <ChipInput
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        values={values}
                        removeLabel={said.removeChip}
                        onChange={onChange}
                    />
                </div>
            )}
        </Field>
    );
}

interface MustLinksProps {
    siteId: string;
    page: DirectedPage;
    links: readonly MustLinkDraft[];
    failed: boolean;
    onChange: (next: MustLinkDraft[]) => void;
}

function MustLinks({ siteId, page, links, failed, onChange }: MustLinksProps): ReactElement {
    const [picking, setPicking] = useState(false);
    const full = links.length >= limits.mustLinks;

    return (
        <div className="flex flex-col gap-1.5" data-run-directive="mustLink">
            <span className="text-xs font-medium text-ink-soft">{said.mustLink}</span>
            {links.length === 0 ? null : (
                <ul className="flex flex-col gap-1">
                    {links.map((link) => (
                        <li key={link.pageId} className="flex min-w-0 items-center gap-2">
                            <span className="w-2/5 min-w-0 truncate font-mono text-xs text-ink">{link.path}</span>
                            <Input
                                size="sm"
                                aria-label={said.anchor}
                                placeholder={said.anchorPlaceholder}
                                value={link.anchor}
                                invalid={[...link.anchor].length > limits.item}
                                onChange={(event) => {
                                    const anchor = event.target.value;
                                    onChange(
                                        links.map((held) => (held.pageId === link.pageId ? { ...held, anchor } : held)),
                                    );
                                }}
                            />
                            <IconButton
                                size="sm"
                                variant="ghost"
                                icon={CloseIcon}
                                label={said.removeLink(link.path)}
                                onClick={() => {
                                    onChange(links.filter((held) => held.pageId !== link.pageId));
                                }}
                            />
                        </li>
                    ))}
                </ul>
            )}
            {picking ? (
                <div className="flex flex-col gap-1 rounded-md border border-hairline bg-panel p-2">
                    <div className="flex items-center justify-between gap-2">
                        <span className="text-2xs font-medium text-ink-soft">{said.pickTitle}</span>
                        <IconButton
                            size="sm"
                            variant="ghost"
                            icon={CloseIcon}
                            label={copy.runs.start.cancel}
                            onClick={() => {
                                setPicking(false);
                            }}
                        />
                    </div>
                    <PageSearch
                        siteId={siteId}
                        picked={null}
                        onPick={(target) => {
                            setPicking(false);
                            if (target.pageId === page.id || links.some((held) => held.pageId === target.pageId)) {
                                return;
                            }
                            onChange([...links, { pageId: target.pageId, path: target.path, anchor: "" }]);
                        }}
                    />
                </div>
            ) : (
                <Button
                    size="sm"
                    variant="ghost"
                    icon={AddLinkIcon}
                    className="self-start"
                    disabled={full}
                    onClick={() => {
                        setPicking(true);
                    }}
                >
                    {said.addMustLink}
                </Button>
            )}
            <p className={cx("text-xs", failed ? "text-danger" : "text-ink-dim")}>
                {failed ? said.tooMany(limits.mustLinks) : said.mustLinkHint}
            </p>
        </div>
    );
}

interface DirectiveFormProps {
    siteId: string;
    page: DirectedPage;
    draft: DirectiveDraft;
    problems: ReadonlySet<DirectiveProblem>;
    onChange: (next: DirectiveDraft) => void;
    onPreview: () => void;
}

function DirectiveForm({ siteId, page, draft, problems, onChange, onPreview }: DirectiveFormProps): ReactElement {
    const words = (key: "minWords" | "maxWords", label: string): ReactElement => (
        <Field
            label={label}
            hint={said.wordsHint}
            error={
                problems.has(key)
                    ? said.wordsHint
                    : key === "maxWords" && problems.has("wordsOrder")
                      ? said.wordsOrder
                      : null
            }
        >
            {(control) => (
                <Input
                    id={control.id}
                    aria-describedby={control["aria-describedby"]}
                    invalid={control.invalid}
                    size="sm"
                    mono={true}
                    inputMode="numeric"
                    data-run-directive={key}
                    value={draft[key]}
                    onChange={(event) => {
                        onChange({ ...draft, [key]: event.target.value });
                    }}
                />
            )}
        </Field>
    );

    return (
        <div className="flex flex-col gap-2.5 border-t border-hairline px-3 py-2.5">
            <Field
                label={said.notes}
                hint={said.notesHint}
                error={problems.has("notes") ? copy.runs.start.brief.tooLong(limits.notes) : null}
            >
                {(control) => (
                    <Textarea
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        rows={3}
                        data-run-directive="notes"
                        value={draft.notes}
                        onChange={(event) => {
                            onChange({ ...draft, notes: event.target.value });
                        }}
                    />
                )}
            </Field>
            <ListField
                label={said.addSections}
                hint={said.addSectionsHint}
                values={draft.addSections}
                most={limits.sections}
                failed={problems.has("addSections")}
                data="addSections"
                onChange={(next) => {
                    onChange({ ...draft, addSections: next });
                }}
            />
            <div className="grid grid-cols-2 gap-2">
                <ListField
                    label={said.keywords}
                    hint={said.keywordsHint}
                    values={draft.keywords}
                    most={limits.listed}
                    failed={problems.has("keywords")}
                    data="keywords"
                    onChange={(next) => {
                        onChange({ ...draft, keywords: next });
                    }}
                />
                <ListField
                    label={said.anchors}
                    hint={said.anchorsHint}
                    values={draft.anchors}
                    most={limits.listed}
                    failed={problems.has("anchors")}
                    data="anchors"
                    onChange={(next) => {
                        onChange({ ...draft, anchors: next });
                    }}
                />
            </div>
            <MustLinks
                siteId={siteId}
                page={page}
                links={draft.mustLink}
                failed={problems.has("mustLink")}
                onChange={(next) => {
                    onChange({ ...draft, mustLink: next });
                }}
            />
            <div className="grid grid-cols-2 gap-2">
                {words("minWords", said.minWords)}
                {words("maxWords", said.maxWords)}
            </div>
            <Button size="sm" variant="secondary" icon={PreviewIcon} className="self-start" onClick={onPreview}>
                {said.preview}
            </Button>
        </div>
    );
}

export interface DirectiveListProps {
    siteId: string;
    pages: readonly DirectedPage[];
    drafts: Directives;
    onChange: (pageId: string, next: DirectiveDraft) => void;
    onPreview: (pageId: string) => void;
}

export function DirectiveList({ siteId, pages, drafts, onChange, onPreview }: DirectiveListProps): ReactElement {
    const [open, setOpen] = useState<string | null>(null);
    const [whole, setWhole] = useState(false);
    const shown = whole ? pages : pages.slice(0, shownAtFirst);
    const hidden = pages.length - shown.length;

    return (
        <section aria-label={said.title} data-run-directives={true} className="flex flex-col gap-1.5">
            <div className="flex flex-col">
                <span className="text-xs font-semibold text-ink">{said.title}</span>
                <span className="text-2xs text-ink-faint">{said.hint}</span>
            </div>
            <ul className="flex flex-col overflow-hidden rounded-md border border-hairline bg-panel">
                {shown.map((page) => {
                    const draft = drafts.get(page.id);
                    const size = directiveSize(draft);
                    const problems = draft === undefined ? new Set<DirectiveProblem>() : directiveProblems(draft);
                    const expanded = open === page.id;
                    return (
                        <li
                            key={page.id}
                            data-run-directive-page={page.id}
                            className="flex flex-col border-b border-hairline last:border-b-0"
                        >
                            <button
                                type="button"
                                aria-expanded={expanded}
                                title={expanded ? said.close(page.path) : said.open(page.path)}
                                onClick={() => {
                                    setOpen(expanded ? null : page.id);
                                }}
                                className="flex h-7 min-w-0 items-center gap-2 px-3 text-left hover:bg-inset"
                            >
                                <ChevronRightIcon
                                    size={14}
                                    className={cx(
                                        "shrink-0 text-ink-dim transition-transform duration-100",
                                        expanded && "rotate-90",
                                    )}
                                />
                                <span className="min-w-0 flex-1 truncate font-mono text-xs text-ink">{page.path}</span>
                                <span className={cx("shrink-0 text-2xs", countTone(problems.size > 0, size))}>
                                    {size === 0 ? said.none : said.some(size)}
                                </span>
                            </button>
                            {expanded ? (
                                <DirectiveForm
                                    siteId={siteId}
                                    page={page}
                                    draft={draft ?? emptyDirective}
                                    problems={problems}
                                    onChange={(next) => {
                                        onChange(page.id, next);
                                    }}
                                    onPreview={() => {
                                        onPreview(page.id);
                                    }}
                                />
                            ) : null}
                        </li>
                    );
                })}
            </ul>
            {hidden <= 0 ? null : (
                <Button
                    size="sm"
                    variant="ghost"
                    className="self-start"
                    onClick={() => {
                        setWhole(true);
                    }}
                >
                    {said.more(hidden)}
                </Button>
            )}
        </section>
    );
}
