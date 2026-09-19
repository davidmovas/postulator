import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { failure } from "../../data/errors.js";
import { useUpdatePage } from "../../data/hooks/pages.js";
import { useResolvedTemplate, useTemplate, useTemplates } from "../../data/hooks/templates.js";
import type { Page } from "../../data/types.js";
import type { SelectOption } from "../../ui/index.js";
import { Button, DashboardCustomizeIcon, Panel, PanelHeader, Select, Skeleton, StatusBadge } from "../../ui/index.js";

const siteDefault = "site-default";

export interface TemplatePanelProps {
    page: Page;
    siteId: string;
}

export function TemplatePanel({ page, siteId }: TemplatePanelProps): ReactElement {
    const navigate = useNavigate();
    const resolved = useResolvedTemplate(page.id);
    const named = useTemplate(resolved.data?.templateId ?? null);
    const globals = useTemplates({ scope: "global" }, null, 100);
    const locals = useTemplates({ siteId }, null, 100);
    const update = useUpdatePage();
    const [choice, setChoice] = useState(page.templateId ?? siteDefault);

    useEffect(() => {
        setChoice(page.templateId ?? siteDefault);
        update.reset();
    }, [page.id, page.templateId]);

    const options = useMemo<SelectOption<string>[]>(
        () => [
            { value: siteDefault, label: copy.pages.detail.templateDefaultOption },
            ...[...flatten(globals.data?.pages), ...flatten(locals.data?.pages)].map((template) => ({
                value: template.id,
                label: `${template.name} · ${template.pageKind}`,
            })),
        ],
        [globals.data, locals.data],
    );

    const changed = choice !== (page.templateId ?? siteDefault);
    const missing = resolved.isError && failure(resolved.error).code === "NOT_FOUND";
    const templateId = resolved.data?.templateId ?? null;

    return (
        <Panel>
            <PanelHeader title={copy.pages.detail.template}>
                {page.templateId === null ? (
                    <StatusBadge tone="muted" dot={false}>
                        {copy.pages.detail.templateDefault}
                    </StatusBadge>
                ) : (
                    <StatusBadge tone="accent" dot={false}>
                        {copy.pages.detail.templateOwn}
                    </StatusBadge>
                )}
            </PanelHeader>
            <div className="flex flex-col gap-2.5 p-3">
                {resolved.isPending ? (
                    <Skeleton height={18} width="60%" />
                ) : missing ? (
                    <p className="text-xs text-warn">{copy.pages.detail.templateNone}</p>
                ) : resolved.data === undefined ? (
                    <p className="text-xs text-ink-dim">{failure(resolved.error).message}</p>
                ) : (
                    <div className="flex items-center gap-2">
                        <DashboardCustomizeIcon size={18} className="shrink-0 text-accent" />
                        <span className="min-w-0 truncate text-sm font-semibold text-ink">
                            {named.data === undefined
                                ? copy.app.loading
                                : copy.pages.detail.templateVersion(named.data.template.name, resolved.data.version)}
                        </span>
                    </div>
                )}
                <div className="flex items-end gap-2">
                    <Select
                        className="min-w-0 flex-1"
                        value={choice}
                        options={options}
                        aria-label={copy.pages.detail.template}
                        onValueChange={setChoice}
                    />
                    <Button
                        variant="primary"
                        disabled={!changed}
                        busy={update.isPending}
                        onClick={() => {
                            update.mutate({ id: page.id, templateId: choice === siteDefault ? "" : choice });
                        }}
                    >
                        {copy.pages.detail.templateSave}
                    </Button>
                </div>
                {templateId === null ? null : (
                    <div className="flex flex-wrap gap-2">
                        <Button
                            size="sm"
                            onClick={() => {
                                void navigate(`/s/${siteId}/templates/${templateId}?page=${page.id}`);
                            }}
                        >
                            {copy.pages.detail.templateCustomize}
                        </Button>
                        <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => {
                                void navigate(`/s/${siteId}/templates/${templateId}`);
                            }}
                        >
                            {copy.pages.detail.templateOpen}
                        </Button>
                    </div>
                )}
            </div>
        </Panel>
    );
}
