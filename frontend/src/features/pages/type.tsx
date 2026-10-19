import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { errorMessageOf } from "../../data/errors.js";
import { useSetPageType } from "../../data/hooks/pages.js";
import { usePageTypes } from "../../data/hooks/templates.js";
import { pushToast } from "../../data/toasts.js";
import type { Page } from "../../data/types.js";
import type { SelectOption, Tone } from "../../ui/index.js";
import { Button, Panel, PanelHeader, Select, Skeleton, StatusBadge, toneClasses } from "../../ui/index.js";
import { pageTypesOf } from "../settings/tabs.js";
import { presetIcon, presetLabel, presetTone, reasonLabel, sourceLabel } from "../settings/types/labels.js";
import { orderedTypes } from "../settings/types/model.js";

const said = copy.pages.detail;

function sourceTone(source: string): Tone {
    switch (source) {
        case "manual":
            return "accent";
        case "auto":
            return "info";
        default:
            return "muted";
    }
}

export interface TypePanelProps {
    page: Page;
    siteId: string;
}

export function TypePanel({ page, siteId }: TypePanelProps): ReactElement {
    const listed = usePageTypes(siteId);
    const set = useSetPageType();
    const [choice, setChoice] = useState<string | null>(page.pageTypeId);

    useEffect(() => {
        setChoice(page.pageTypeId);
        set.reset();
    }, [page.id, page.pageTypeId]);

    const types = useMemo(() => orderedTypes(listed.data?.types ?? []), [listed.data]);
    const current = types.find((held) => held.id === page.pageTypeId) ?? null;
    const options: readonly SelectOption<string>[] = types.map((held) => ({
        value: held.id,
        label: said.typeOption(held.name, presetLabel(held.preset)),
    }));
    const changed = choice !== null && choice !== page.pageTypeId;
    const problem = errorMessageOf(set.error);
    const Icon = presetIcon(current?.preset ?? "");

    return (
        <Panel>
            <PanelHeader title={said.type}>
                <StatusBadge tone={sourceTone(page.typeSource)} dot={false}>
                    {sourceLabel(page.typeSource)}
                </StatusBadge>
            </PanelHeader>
            <div className="flex flex-col gap-2.5 p-3" data-page-type={current?.preset ?? ""}>
                {listed.isPending ? (
                    <Skeleton height={18} width="60%" />
                ) : page.pageTypeId === null ? (
                    <p className="text-xs text-ink-dim">{said.typeNone}</p>
                ) : current === null ? (
                    <p className="text-xs text-warn">{said.typeUnknown}</p>
                ) : (
                    <div className="flex items-start gap-2">
                        <Icon size={18} className={`mt-px shrink-0 ${toneClasses[presetTone(current.preset)].ink}`} />
                        <div className="flex min-w-0 flex-col">
                            <span className="truncate text-sm font-semibold text-ink">
                                {said.typeOption(current.name, presetLabel(current.preset))}
                            </span>
                            <span className="text-2xs text-ink-faint">
                                {said.typeWhy(sourceLabel(page.typeSource), reasonLabel(page.typeReason))}
                            </span>
                        </div>
                    </div>
                )}
                <div className="flex items-end gap-2">
                    <div className="min-w-0 flex-1">
                        <Select
                            value={choice}
                            options={options}
                            placeholder={said.typePick}
                            aria-label={said.type}
                            disabled={types.length === 0}
                            onValueChange={setChoice}
                        />
                    </div>
                    <Button
                        variant="primary"
                        disabled={!changed}
                        busy={set.isPending}
                        data-page-type-save={true}
                        onClick={() => {
                            if (choice === null) {
                                return;
                            }
                            const picked = types.find((held) => held.id === choice);
                            set.mutate(
                                { pageId: page.id, pageTypeId: choice },
                                {
                                    onSuccess: () => {
                                        pushToast("info", said.typeSaved(picked?.name ?? ""));
                                    },
                                },
                            );
                        }}
                    >
                        {said.typeSave}
                    </Button>
                </div>
                <p className="text-2xs text-ink-faint">{said.typeHint}</p>
                {problem === null ? null : <p className="text-xs text-danger">{problem}</p>}
                {current === null ? null : (
                    <Link
                        to={`${pageTypesOf(siteId)}&type=${encodeURIComponent(current.id)}`}
                        className="w-fit text-xs text-accent hover:underline"
                    >
                        {said.typeOpen}
                    </Link>
                )}
            </div>
        </Panel>
    );
}
