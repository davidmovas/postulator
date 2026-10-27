import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { failure, reasonOf } from "../../../data/errors.js";
import { useResolveSelection, useSelectionCount, useSetPageTemplates, useSetPageTypes } from "../../../data/hooks/pages.js";
import { useTemplates } from "../../../data/hooks/templates.js";
import { pushToast } from "../../../data/toasts.js";
import type { MenuEntry } from "../../../ui/index.js";
import {
    Button,
    CategoryIcon,
    CloseIcon,
    DescriptionIcon,
    Dialog,
    IconButton,
    Menu,
    PlayArrowIcon,
    SmartToyIcon,
    Spinner,
    useDebounced,
} from "../../../ui/index.js";
import type { Selection } from "../../../ui/selection.js";
import { count, dropWhole, exact, noSelection, toRequest } from "../../../ui/selection.js";
import { askAgent } from "../../agent/index.js";
import type { KnownRow, Naming } from "./model.js";
import { prefillOf, selectedPaths, selectionWords } from "./model.js";
import type { TypeNaming } from "./naming.js";

const said = copy.pages.bulk;

const templatePage = 100;

const shownPaths = 20;

const countDelayMs = 250;

const selectionMoved = "selection_moved";

const selectionLimit = "selection_limit";

type Asked = { kind: "type"; typeId: string } | { kind: "rules" } | { kind: "template"; templateId: string } | { kind: "follow" };

export interface SelectAllOffer {
    count: number;
    onSelect: () => void;
}

export interface BulkBarProps {
    siteId: string;
    selection: Selection;
    onSelection: (next: Selection) => void;
    naming: TypeNaming;
    known: () => ReadonlyMap<string, KnownRow>;
    entityName: (entityId: string) => string;
    selectAll: SelectAllOffer | null;
    onStartRun: (pageIds: readonly string[], paths: ReadonlyMap<string, string>) => void;
}

function pathsOf(held: Readonly<Record<string, string | undefined>> | null): ReadonlyMap<string, string> {
    return new Map(Object.entries(held ?? {}).filter((entry): entry is [string, string] => entry[1] !== undefined));
}

export function BulkBar({ siteId, selection, onSelection, naming, known, entityName, selectAll, onStartRun }: BulkBarProps): ReactElement {
    const [asked, setAsked] = useState<Asked | null>(null);
    const request = useMemo(() => toRequest(siteId, selection), [siteId, selection]);
    const certain = exact(selection);
    const settled = useDebounced(JSON.stringify(request), countDelayMs);
    const counted = useSelectionCount(certain ? null : (JSON.parse(settled) as typeof request));
    const setTypes = useSetPageTypes();
    const setTemplates = useSetPageTemplates();
    const resolve = useResolveSelection();
    const globals = useTemplates({ scope: "global" }, null, templatePage);
    const locals = useTemplates({ siteId }, null, templatePage);
    const templates = useMemo(
        () => [...flatten(locals.data?.pages), ...flatten(globals.data?.pages)].map((held) => ({ id: held.id, name: held.name })),
        [globals.data, locals.data],
    );
    const templateName = (templateId: string): string => templates.find((held) => held.id === templateId)?.name ?? templateId;

    const countReason = reasonOf(counted.error);
    const shown: number | null = certain ? count(selection) : counted.data === undefined ? null : (counted.data.pageIds ?? []).length;
    const label =
        countReason === selectionLimit
            ? said.tooMany
            : countReason === selectionMoved
              ? said.moved
              : shown === null
                ? said.counting
                : said.selected(shown);

    const refused = (thrown: unknown): void => {
        if (reasonOf(thrown) === selectionMoved) {
            pushToast("warning", said.moved);
            onSelection(dropWhole(selection));
            return;
        }
        pushToast("warning", failure(thrown).message);
    };

    const wording: Naming = {
        typeName: naming.typeName,
        ruleName: naming.ruleName,
        entityName,
        pagePath: (pageId) => known().get(pageId)?.path ?? pageId,
    };

    const apply = (): void => {
        if (asked === null) {
            return;
        }
        const done = (): void => {
            setAsked(null);
            onSelection(noSelection);
        };
        switch (asked.kind) {
            case "type":
            case "rules": {
                const pageTypeId = asked.kind === "type" ? asked.typeId : null;
                setTypes.mutate(
                    { selection: request, pageTypeId },
                    {
                        onSuccess: (answered) => {
                            pushToast(
                                "info",
                                pageTypeId === null
                                    ? said.ruled(answered.typed)
                                    : answered.typed === 0 && answered.refused === 0
                                      ? said.unchanged
                                      : said.typed(answered.typed, answered.refused, naming.typeName(pageTypeId)),
                            );
                            done();
                        },
                        onError: (thrown) => {
                            setAsked(null);
                            refused(thrown);
                        },
                    },
                );
                return;
            }
            default: {
                const templateId = asked.kind === "template" ? asked.templateId : null;
                setTemplates.mutate(
                    { selection: request, templateId },
                    {
                        onSuccess: (answered) => {
                            pushToast(
                                "info",
                                answered.changed === 0
                                    ? said.unchanged
                                    : templateId === null
                                      ? said.followed(answered.changed)
                                      : said.templated(answered.changed, templateName(templateId)),
                            );
                            done();
                        },
                        onError: (thrown) => {
                            setAsked(null);
                            refused(thrown);
                        },
                    },
                );
            }
        }
    };

    const typeItems: MenuEntry[] = [
        ...naming.types.map((type) => ({
            key: type.id,
            label: type.name,
            onSelect: () => {
                setAsked({ kind: "type", typeId: type.id });
            },
        })),
        { kind: "separator", key: "rules-separator" },
        {
            key: "rules",
            label: said.backToRules,
            onSelect: () => {
                setAsked({ kind: "rules" });
            },
        },
    ];

    const templateItems: MenuEntry[] = [
        ...templates.map((template) => ({
            key: template.id,
            label: template.name,
            onSelect: () => {
                setAsked({ kind: "template", templateId: template.id });
            },
        })),
        { kind: "separator", key: "follow-separator" },
        {
            key: "follow",
            label: said.followType,
            onSelect: () => {
                setAsked({ kind: "follow" });
            },
        },
    ];

    const dialog = ((): { title: string; body: string } => {
        switch (asked?.kind) {
            case "type":
                return { title: said.typeTitle(naming.typeName(asked.typeId)), body: said.typeBody(shown, naming.typeName(asked.typeId)) };
            case "rules":
                return { title: said.rulesTitle, body: said.rulesBody(shown) };
            case "template":
                return { title: said.templateTitle(templateName(asked.templateId)), body: said.templateBody(shown, templateName(asked.templateId)) };
            default:
                return { title: said.followTitle, body: said.followBody(shown) };
        }
    })();

    const busy = setTypes.isPending || setTemplates.isPending;

    return (
        <section
            aria-label={said.label}
            data-bulk-bar={true}
            className="flex shrink-0 flex-wrap items-center gap-x-3 gap-y-2 border-t border-accent-border bg-panel px-3 py-2"
        >
            <span data-bulk-count={shown ?? ""} className="flex items-center gap-2 text-sm font-medium text-ink">
                {shown === null && countReason === null ? <Spinner size={12} /> : null}
                {label}
            </span>
            {selectAll === null ? null : (
                <Button size="sm" variant="ghost" onClick={selectAll.onSelect}>
                    {copy.pages.select.allMatching(selectAll.count)}
                </Button>
            )}
            <span className="ml-auto flex flex-wrap items-center gap-1.5">
                <Menu
                    label={said.setType}
                    items={typeItems}
                    trigger={
                        <Button size="sm" icon={CategoryIcon} disabled={busy}>
                            {said.setType}
                        </Button>
                    }
                />
                <Menu
                    label={said.template}
                    items={templateItems}
                    trigger={
                        <Button size="sm" icon={DescriptionIcon} disabled={busy}>
                            {said.template}
                        </Button>
                    }
                />
                <Button
                    size="sm"
                    icon={PlayArrowIcon}
                    busy={resolve.isPending}
                    disabled={busy}
                    onClick={() => {
                        resolve.mutate(request, {
                            onSuccess: (answered) => {
                                onStartRun(answered.pageIds ?? [], pathsOf(answered.paths));
                            },
                            onError: refused,
                        });
                    }}
                >
                    {said.run}
                </Button>
                <Button
                    size="sm"
                    icon={SmartToyIcon}
                    onClick={() => {
                        askAgent(prefillOf(shown, selectionWords(selection, wording), selectedPaths(selection, known(), shownPaths)));
                    }}
                >
                    {said.ask}
                </Button>
                <IconButton
                    size="sm"
                    variant="ghost"
                    icon={CloseIcon}
                    label={said.clear}
                    onClick={() => {
                        onSelection(noSelection);
                    }}
                />
            </span>
            <Dialog
                open={asked !== null}
                onOpenChange={(next) => {
                    if (!next) {
                        setAsked(null);
                    }
                }}
                title={dialog.title}
                description={dialog.body}
                confirmLabel={said.confirm}
                cancelLabel={said.cancel}
                busy={busy}
                onConfirm={apply}
            />
        </section>
    );
}
