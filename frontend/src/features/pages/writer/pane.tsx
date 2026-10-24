import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router";

import { copy } from "../../../copy/index.js";
import { failure } from "../../../data/errors.js";
import { useWriterPack } from "../../../data/hooks/pages.js";
import { usePageTypes } from "../../../data/hooks/templates.js";
import type { Page, WriterPack, WriterPackDrop, WriterPackSection } from "../../../data/types.js";
import { tokens } from "../../../domain/format.js";
import type { SelectOption } from "../../../ui/index.js";
import { BudgetGauge, Field, Panel, PanelHeader, Select, SkeletonRows, StatusBadge } from "../../../ui/index.js";
import { settingsHome } from "../../settings/tabs.js";
import { fieldLabel } from "../../settings/types/labels.js";
import { Note } from "../model/facts.js";
import type { WriterField } from "./fields.js";
import { askedField, budgetRefusal, byPriority, writerFields } from "./fields.js";

const said = copy.pages.writer;

export interface WriterTabProps {
    page: Page;
    siteId: string;
}

export function WriterTab({ page, siteId }: WriterTabProps): ReactElement {
    const listed = usePageTypes(siteId);
    const type = listed.data?.types?.find((held) => held.id === page.pageTypeId) ?? null;
    const fields = useMemo(() => writerFields(type), [type]);
    const main = fields[0]?.key ?? "";
    const [choice, setChoice] = useState(main);

    useEffect(() => {
        setChoice(main);
    }, [page.id, main]);

    const asked = askedField(fields, choice);
    const pack = useWriterPack(page.id, asked);
    const shown = fields.some((field) => field.key === choice) ? choice : main;
    const options: readonly SelectOption<string>[] = fields.map((field) => ({ value: field.key, label: field.label }));

    return (
        <div className="flex flex-col gap-3 p-3" data-writer-pack={asked === "" ? main : asked}>
            <Panel>
                <PanelHeader title={said.title} />
                <div className="p-3">
                    <Field label={said.field} hint={said.fieldHint}>
                        {(control) => (
                            <Select
                                id={control.id}
                                aria-describedby={control["aria-describedby"]}
                                value={shown}
                                options={options}
                                disabled={fields.length < 2}
                                onValueChange={setChoice}
                            />
                        )}
                    </Field>
                </div>
            </Panel>
            {pack.isPending ? (
                <SkeletonRows rows={8} label={said.loading} />
            ) : pack.data === undefined ? (
                <Refused thrown={pack.error} />
            ) : (
                <Fitted pack={pack.data} fields={fields} />
            )}
        </div>
    );
}

function Refused({ thrown }: { thrown: unknown }): ReactElement {
    const reported = failure(thrown);
    const small = budgetRefusal(reported.details);
    return (
        <Panel>
            <div className="flex flex-col gap-2 p-3">
                <p className="text-sm text-ink">
                    {small === null ? reported.message : said.tooSmall(tokens(small.budget), tokens(small.needed))}
                </p>
                {small === null ? null : <SettingsLink />}
            </div>
        </Panel>
    );
}

function SettingsLink(): ReactElement {
    return (
        <Link to={settingsHome} className="w-fit text-xs text-accent hover:underline">
            {said.openSettings}
        </Link>
    );
}

function labelOf(fields: readonly WriterField[], key: string): string {
    return fields.find((field) => field.key === key)?.label ?? fieldLabel(key);
}

function Fitted({ pack, fields }: { pack: WriterPack; fields: readonly WriterField[] }): ReactElement {
    const sections = byPriority(pack.sections ?? []);
    const dropped = pack.dropped ?? [];
    const kept = said.kept(tokens(pack.tokens), tokens(pack.built));

    return (
        <>
            <Panel>
                <PanelHeader title={said.summaryTitle}>
                    <StatusBadge tone="accent" dot={false}>
                        {labelOf(fields, pack.field)}
                    </StatusBadge>
                </PanelHeader>
                <div className="flex flex-col gap-2 p-3">
                    {pack.budget > 0 ? (
                        <BudgetGauge
                            value={pack.tokens}
                            max={pack.budget}
                            label={kept}
                            leading={kept}
                            trailing={tokens(pack.budget)}
                        />
                    ) : (
                        <p className="font-mono text-xs text-ink-dim">{kept}</p>
                    )}
                    <p className="text-xs text-ink-soft">
                        {pack.budget > 0 ? said.withinBudget(tokens(pack.budget)) : said.wholeWindow}
                    </p>
                    <p className="text-2xs text-ink-faint">{said.budgetHint}</p>
                    <SettingsLink />
                </div>
            </Panel>
            <Panel>
                <PanelHeader title={said.sectionsTitle} />
                {sections.length === 0 ? (
                    <Note>{said.noSections}</Note>
                ) : (
                    <>
                        <p className="border-b border-inset px-3 py-1.5 text-2xs text-ink-faint">{said.priorityHint}</p>
                        <ol className="flex flex-col">
                            {sections.map((section) => (
                                <PackSection key={section.name} section={section} />
                            ))}
                        </ol>
                    </>
                )}
            </Panel>
            <Panel>
                <PanelHeader title={said.dropped.title} />
                {dropped.length === 0 ? (
                    <Note>{said.dropped.none}</Note>
                ) : (
                    <ul className="flex flex-col">
                        {dropped.map((drop) => (
                            <Dropped key={drop.section} drop={drop} />
                        ))}
                    </ul>
                )}
            </Panel>
        </>
    );
}

function PackSection({ section }: { section: WriterPackSection }): ReactElement {
    const lines = section.lines ?? [];
    const items = section.items ?? [];
    return (
        <li
            data-pack-section={section.name}
            className="flex flex-col gap-1.5 border-b border-inset px-3 py-2 last:border-b-0"
        >
            <div className="flex min-w-0 items-center gap-2">
                <h3 className="truncate text-xs font-semibold text-ink">{section.heading}</h3>
                <StatusBadge tone={section.priority === 1 ? "accent" : "muted"} dot={false}>
                    {said.priority(section.priority)}
                </StatusBadge>
                <span className="ml-auto shrink-0 font-mono text-2xs text-ink-faint">
                    {said.tokens(tokens(section.tokens))}
                </span>
            </div>
            {lines.map((line, at) => (
                <p key={`line-${at}`} className="text-xs whitespace-pre-wrap text-ink-soft">
                    {line}
                </p>
            ))}
            {items.length === 0 ? null : (
                <ul className="flex list-disc flex-col gap-0.5 pl-4">
                    {items.map((item, at) => (
                        <li key={`item-${at}`} className="text-xs whitespace-pre-wrap text-ink-soft">
                            {item}
                        </li>
                    ))}
                </ul>
            )}
        </li>
    );
}

function Dropped({ drop }: { drop: WriterPackDrop }): ReactElement {
    return (
        <li
            data-pack-dropped={drop.section}
            className="flex min-w-0 items-center gap-2 border-b border-inset px-3 py-2 last:border-b-0"
        >
            <span className="truncate text-xs text-ink-soft">{drop.heading}</span>
            <span className="shrink-0 text-2xs text-ink-faint">
                {drop.whole ? said.dropped.whole : said.dropped.items(drop.items)}
            </span>
            <span className="ml-auto shrink-0 font-mono text-2xs text-ink-faint">{said.tokens(tokens(drop.tokens))}</span>
        </li>
    );
}
