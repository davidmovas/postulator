import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { failure } from "../../data/errors.js";
import { useSite, useUpdateSite } from "../../data/hooks/sites.js";
import { useCreateTemplate, useDeleteTemplate } from "../../data/hooks/templates.js";
import { AddIcon, Banner, Button, CountBadge, Screen } from "../../ui/index.js";
import { askAgent } from "../agent/index.js";
import { CreateTemplateDrawer } from "./create.js";
import { TemplateDetail } from "./detail.js";
import { tabPath, TemplateTabs } from "./header.js";
import { copyName } from "./naming.js";
import type { TemplatesQuery } from "./params.js";
import { defaultQuery, narrowed, readQuery, wantsNew, writeQuery } from "./params.js";
import { namesIn, useTemplateRows } from "./rows.js";
import { TemplateTable } from "./table.js";
import { TemplateToolbar } from "./toolbar.js";

export function TemplatesScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const query = useMemo(() => readQuery(searchParams), [searchParams]);
    const rows = useTemplateRows(siteId, query);
    const site = useSite(siteId === "" ? null : siteId);
    const updateSite = useUpdateSite();
    const create = useCreateTemplate();
    const remove = useDeleteTemplate();
    const [creating, setCreating] = useState(false);
    const [selectedId, setSelectedId] = useState<string | null>(null);
    const asked = wantsNew(searchParams);

    useEffect(() => {
        if (asked) {
            setCreating(true);
            setSearchParams(writeQuery(readQuery(searchParams)), { replace: true });
        }
    }, [asked, searchParams, setSearchParams]);

    const selected = rows.shown.find((held) => held.id === selectedId) ?? rows.shown[0] ?? null;
    const defaultId = site.data?.site.defaults.templateId ?? null;
    const failed = rows.error !== null && rows.error !== undefined;

    const change = (next: TemplatesQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    const open = (id: string): void => {
        void navigate(`/s/${siteId}/templates/${id}`);
    };

    const duplicate = (): void => {
        if (selected === null) {
            return;
        }
        create.mutate(
            {
                scope: "site",
                siteId,
                name: copyName(selected.name, namesIn(rows.loaded, "site")),
                pageKind: selected.pageKind,
                spec: selected.spec,
            },
            {
                onSuccess: (answered) => {
                    setSelectedId(answered.template.id);
                },
            },
        );
    };

    const setDefault = (): void => {
        const held = site.data?.site;
        if (held === undefined || selected === null) {
            return;
        }
        updateSite.mutate({ id: held.id, defaults: { ...held.defaults, templateId: selected.id } });
    };

    return (
        <Screen
            title={copy.templates.title}
            badge={rows.pending ? undefined : <CountBadge tone="muted" count={rows.shown.length} />}
            variant="split"
            tabs={<TemplateTabs siteId={siteId} tab="templates" />}
            actions={
                <>
                    <Button
                        onClick={() => {
                            void navigate(`${tabPath(siteId, "policies")}?action=new`);
                        }}
                    >
                        {copy.policies.newPolicy}
                    </Button>
                    <Button
                        variant="primary"
                        icon={AddIcon}
                        data-template-new={true}
                        onClick={() => {
                            setCreating(true);
                        }}
                    >
                        {copy.templates.newTemplate}
                    </Button>
                </>
            }
            toolbar={<TemplateToolbar query={query} kinds={rows.kinds} onChange={change} />}
            right={
                selected === null || rows.pending ? null : (
                    <TemplateDetail
                        key={selected.id}
                        template={selected}
                        isDefault={selected.id === defaultId}
                        duplicating={create.isPending}
                        settingDefault={updateSite.isPending}
                        deleting={remove.isPending}
                        onOpen={() => {
                            open(selected.id);
                        }}
                        onDuplicate={duplicate}
                        onSetDefault={setDefault}
                        onDelete={() => {
                            remove.mutate(
                                { id: selected.id },
                                {
                                    onSuccess: () => {
                                        setSelectedId(null);
                                    },
                                },
                            );
                        }}
                        onAskAgent={() => {
                            askAgent(copy.agent.ask.template(selected.name, selected.id));
                        }}
                    />
                )
            }
        >
            {failed ? (
                <div className="p-4">
                    <Banner
                        tone="danger"
                        title={failure(rows.error).message}
                        actions={<Button onClick={rows.retry}>{copy.app.retry}</Button>}
                    />
                </div>
            ) : (
                <TemplateTable
                    rows={rows.shown}
                    pending={rows.pending}
                    narrowed={narrowed(query)}
                    selectedId={selected?.id ?? null}
                    defaultId={defaultId}
                    hasMore={rows.hasMore}
                    loadingMore={rows.loadingMore}
                    onSelect={setSelectedId}
                    onOpen={open}
                    onLoadMore={rows.loadMore}
                    onCreate={() => {
                        setCreating(true);
                    }}
                    onAskAgent={() => {
                        askAgent(copy.agent.ask.newTemplate);
                    }}
                    onClearFilters={() => {
                        change(defaultQuery);
                    }}
                />
            )}
            <CreateTemplateDrawer
                open={creating}
                siteId={siteId}
                templates={rows.loaded}
                onOpenChange={setCreating}
                onCreated={open}
            />
        </Screen>
    );
}
