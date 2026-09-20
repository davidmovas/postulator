import type { ReactElement } from "react";
import { useEffect, useState } from "react";
import { useBlocker, useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { failure } from "../../data/errors.js";
import { usePage } from "../../data/hooks/pages.js";
import { useSite, useUpdateSite } from "../../data/hooks/sites.js";
import {
    useDeleteOverride,
    useDeleteTemplate,
    useSetOverride,
    useUpdateTemplate,
} from "../../data/hooks/templates.js";
import { Banner, Button, Screen, SkeletonRows } from "../../ui/index.js";
import { askAgent } from "../agent/index.js";
import { formErrorOf } from "./controls.js";
import { ConflictBar, EditorActions, EditorBadges, LayerChooser } from "./editor-actions.js";
import { EditorDialogs } from "./editor-dialogs.js";
import { useDraft, useLayerState } from "./editor-state.js";
import { GroupView } from "./groups/view.js";
import type { GroupKey } from "./outline.js";
import { groupTitles, isGroup } from "./outline.js";
import { OutlineRail } from "./outline-rail.js";
import type { Layer } from "./patch.js";
import { patchBetween } from "./patch.js";
import { SkeletonPanel } from "./preview/panel.js";
import { specOf } from "./spec.js";

export function TemplateEditorScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const templateId = params.templateId ?? "";
    const pageId = searchParams.get("page");
    const asked = searchParams.get("group") ?? "";

    const [layer, setLayer] = useState<Layer>(pageId === null ? "global" : "page");
    const [renaming, setRenaming] = useState(false);
    const [confirming, setConfirming] = useState(false);
    const [leaving, setLeaving] = useState<{ act: () => void } | null>(null);

    const state = useLayerState(templateId, siteId, pageId, layer);
    const { draft, dirty, marks, conflict, edit, clear } = useDraft(
        `${templateId}|${layer}|${pageId ?? ""}`,
        state,
        layer,
    );
    const page = usePage(pageId);
    const site = useSite(siteId === "" ? null : siteId);
    const updateSite = useUpdateSite();
    const update = useUpdateTemplate();
    const override = useSetOverride();
    const dropOverride = useDeleteOverride();
    const remove = useDeleteTemplate();

    useEffect(() => {
        setLayer(pageId === null ? "global" : "page");
    }, [pageId]);

    const group: GroupKey = isGroup(asked) ? asked : "sections";
    const { template, base, siteResolved, start, below, siteOverride, pageOverride } = state;

    const blocker = useBlocker(
        ({ currentLocation, nextLocation }) => dirty && currentLocation.pathname !== nextLocation.pathname,
    );

    const pending = update.isPending || override.isPending || dropOverride.isPending;
    const thrown = layer === "global" ? update.error : (override.error ?? dropOverride.error);
    const isDefault = site.data?.site.defaults.templateId === templateId;
    const pagePath = page.data?.page.path ?? "";

    const drop = (): void => {
        clear();
        update.reset();
        override.reset();
        dropOverride.reset();
    };

    const guard = (act: () => void): void => {
        if (dirty) {
            setLeaving({ act });
            return;
        }
        act();
    };

    const save = (): void => {
        if (template === undefined || draft === null || base === null || siteResolved === null) {
            return;
        }
        const done = { onSuccess: drop };
        if (layer === "global") {
            update.mutate({ id: template.id, spec: specOf(draft) }, done);
            return;
        }
        const against = layer === "site" ? base : siteResolved;
        const current = layer === "site" ? siteOverride : pageOverride;
        const patch = patchBetween(against, draft);
        if (patch === undefined) {
            if (current !== null) {
                dropOverride.mutate({ id: current.id }, done);
            } else {
                drop();
            }
            return;
        }
        override.mutate(
            {
                templateId: template.id,
                scope: layer,
                targetId: layer === "site" ? siteId : (pageId ?? ""),
                patch,
            },
            done,
        );
    };

    if (state.pending) {
        return (
            <div className="p-4">
                <SkeletonRows rows={12} label={copy.templates.loading} />
            </div>
        );
    }

    if (template === undefined || draft === null || base === null || start === null || below === null) {
        return (
            <div className="p-4">
                <Banner
                    tone="danger"
                    title={
                        failure(state.error).code === "NOT_FOUND"
                            ? copy.templates.editor.notFound
                            : failure(state.error).message
                    }
                    actions={
                        <Button
                            onClick={() => {
                                void navigate(`/s/${siteId}/templates`);
                            }}
                        >
                            {copy.templates.editor.back}
                        </Button>
                    }
                />
            </div>
        );
    }

    const formError = formErrorOf(thrown);

    return (
        <Screen
            title={template.name}
            badge={<EditorBadges template={template} isDefault={isDefault} dirty={dirty} />}
            variant="split"
            tabs={
                <LayerChooser
                    layer={layer}
                    pageId={pageId}
                    pagePath={pagePath}
                    onChoose={(next) => {
                        guard(() => {
                            clear();
                            setLayer(next);
                        });
                    }}
                />
            }
            actions={
                <EditorActions
                    layer={layer}
                    pageId={pageId}
                    pagePath={pagePath}
                    dirty={dirty}
                    saving={pending}
                    isDefault={isDefault}
                    canSetDefault={site.data !== undefined}
                    onOpenPage={() => {
                        void navigate(`/s/${siteId}/pages/${pageId ?? ""}?tab=preview`);
                    }}
                    onAskAgent={() => {
                        askAgent(copy.agent.ask.template(template.name, template.id));
                    }}
                    onSave={save}
                    onRevert={drop}
                    onRename={() => {
                        setRenaming(true);
                    }}
                    onSetDefault={() => {
                        const current = site.data?.site;
                        if (current === undefined) {
                            return;
                        }
                        updateSite.mutate({
                            id: current.id,
                            defaults: { ...current.defaults, templateId: template.id },
                        });
                    }}
                    onDelete={() => {
                        setConfirming(true);
                    }}
                />
            }
            toolbar={
                conflict === "none" ? undefined : (
                    <ConflictBar kind={conflict} saving={pending} onReload={drop} onOverwrite={save} />
                )
            }
            left={
                <OutlineRail
                    active={group}
                    changed={marks}
                    onSelect={(next) => {
                        const query = new URLSearchParams(searchParams);
                        query.set("group", next);
                        setSearchParams(query, { replace: true });
                    }}
                />
            }
            right={<SkeletonPanel draft={draft} />}
        >
            <div className="min-h-0 flex-1 overflow-auto p-4">
                <div className="flex flex-col gap-3">
                    <h2 className="text-sm font-semibold text-ink">{groupTitles[group]}</h2>
                    {formError === null ? null : <Banner tone="danger" title={formError} />}
                    <GroupView
                        group={group}
                        layer={layer}
                        siteId={siteId}
                        draft={draft}
                        below={below}
                        siteOverride={siteOverride}
                        pageOverrides={state.pageOverrides}
                        error={thrown}
                        onChange={edit}
                    />
                </div>
            </div>
            <EditorDialogs
                name={template.name}
                pageKind={template.pageKind}
                renaming={renaming}
                renameBusy={update.isPending}
                renameError={update.error}
                onRenameOpenChange={setRenaming}
                onRename={(identity) => {
                    update.mutate(
                        { id: template.id, name: identity.name, pageKind: identity.pageKind },
                        {
                            onSuccess: () => {
                                setRenaming(false);
                            },
                        },
                    );
                }}
                leaving={blocker.state === "blocked" || leaving !== null}
                onLeaveOpenChange={(next) => {
                    if (!next) {
                        blocker.reset?.();
                        setLeaving(null);
                    }
                }}
                onLeave={() => {
                    const act = leaving?.act ?? null;
                    clear();
                    setLeaving(null);
                    if (act !== null) {
                        act();
                        return;
                    }
                    blocker.proceed?.();
                }}
                deleting={confirming}
                deleteBusy={remove.isPending}
                onDeleteOpenChange={setConfirming}
                onDelete={() => {
                    remove.mutate(
                        { id: template.id },
                        {
                            onSuccess: () => {
                                setConfirming(false);
                                clear();
                                void navigate(`/s/${siteId}/templates`);
                            },
                        },
                    );
                }}
            />
        </Screen>
    );
}
