import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { failure } from "../../data/errors.js";
import { usePage } from "../../data/hooks/pages.js";
import { useSite, useUpdateSite } from "../../data/hooks/sites.js";
import {
    useDeleteOverride,
    useDeleteTemplate,
    useSetOverride,
    useTemplate,
    useUpdateTemplate,
} from "../../data/hooks/templates.js";
import { copy } from "../../copy/index.js";
import {
    Banner,
    Button,
    ChevronRightIcon,
    DeleteIcon,
    Dialog,
    Field,
    Input,
    Segmented,
    SkeletonRows,
    SmartToyIcon,
    StatusBadge,
    TabPanel,
    Tabs,
    VerifiedIcon,
} from "../../ui/index.js";
import type { SegmentedOption, TabItem } from "../../ui/index.js";
import { askAgent } from "../agent/index.js";
import { fieldErrorOf, formErrorOf } from "./controls.js";
import { ModelsForm } from "./models-form.js";
import { OverridesPanel } from "./overrides-panel.js";
import type { Layer } from "./patch.js";
import { layered, patchBetween, patchObject } from "./patch.js";
import { PagePreview } from "./preview/view.js";
import type { LayerView } from "./provenance.js";
import { RecipeForm } from "./recipe-form.js";
import { ContentForm, LinksForm, MetaForm } from "./rules-form.js";
import { scopeLabel, scopeTone } from "./labels.js";
import { SectionsForm } from "./sections-form.js";
import type { SpecDraft } from "./spec.js";
import { draftOf, specOf } from "./spec.js";

type EditorTab = "sections" | "content" | "links" | "meta" | "models" | "recipe";

type AsideTab = "preview" | "overrides";

const tabs: readonly TabItem<EditorTab>[] = [
    { key: "sections", label: copy.templates.editor.tabs.sections },
    { key: "content", label: copy.templates.editor.tabs.content },
    { key: "links", label: copy.templates.editor.tabs.links },
    { key: "meta", label: copy.templates.editor.tabs.meta },
    { key: "models", label: copy.templates.editor.tabs.models },
    { key: "recipe", label: copy.templates.editor.tabs.recipe },
];

const asideTabs: readonly TabItem<AsideTab>[] = [
    { key: "preview", label: copy.templates.editor.aside.preview },
    { key: "overrides", label: copy.templates.editor.aside.overrides },
];

const layerHints: Readonly<Record<Layer, string>> = {
    global: copy.templates.layer.globalHint,
    site: copy.templates.layer.siteHint,
    page: copy.templates.layer.pageHint,
};

function layerOptions(pageId: string | null): readonly SegmentedOption<Layer>[] {
    const listed: SegmentedOption<Layer>[] = [
        { value: "global", label: copy.templates.layer.global },
        { value: "site", label: copy.templates.layer.site },
    ];
    if (pageId !== null) {
        listed.push({ value: "page", label: copy.templates.layer.page });
    }
    return listed;
}

export function TemplateEditorScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const templateId = params.templateId ?? "";
    const pageId = searchParams.get("page");
    const detail = useTemplate(templateId);
    const page = usePage(pageId);
    const site = useSite(siteId === "" ? null : siteId);
    const updateSite = useUpdateSite();
    const update = useUpdateTemplate();
    const override = useSetOverride();
    const dropOverride = useDeleteOverride();
    const remove = useDeleteTemplate();

    const [layer, setLayer] = useState<Layer>(pageId === null ? "global" : "page");
    const [tab, setTab] = useState<EditorTab>("sections");
    const [asideTab, setAsideTab] = useState<AsideTab>("preview");
    const [confirming, setConfirming] = useState(false);
    const [identity, setIdentity] = useState<{ key: string; name: string; pageKind: string } | null>(null);
    const [edited, setEdited] = useState<{ key: string; draft: SpecDraft } | null>(null);

    useEffect(() => {
        setLayer(pageId === null ? "global" : "page");
        setEdited(null);
    }, [pageId]);

    const template = detail.data?.template;
    const overrides = useMemo(() => detail.data?.overrides ?? [], [detail.data]);
    const siteOverride = useMemo(
        () => overrides.find((held) => held.scope === "site" && held.targetId === siteId) ?? null,
        [overrides, siteId],
    );
    const pageOverride = useMemo(
        () => (pageId === null ? null : (overrides.find((held) => held.scope === "page" && held.targetId === pageId) ?? null)),
        [overrides, pageId],
    );
    const pageOverrides = useMemo(() => overrides.filter((held) => held.scope === "page"), [overrides]);
    const sitePatch = useMemo(() => patchObject(siteOverride?.patch), [siteOverride]);
    const pagePatch = useMemo(() => patchObject(pageOverride?.patch), [pageOverride]);
    const base = useMemo(() => (template === undefined ? null : draftOf(template.spec)), [template]);
    const siteResolved = useMemo(() => (base === null ? null : layered(base, sitePatch)), [base, sitePatch]);

    const key = `${template?.id ?? ""}|${template?.updatedAt ?? ""}|${layer}|${siteOverride?.updatedAt ?? ""}|${pageId ?? ""}|${pageOverride?.updatedAt ?? ""}`;
    const start = useMemo(() => {
        if (base === null || siteResolved === null) {
            return null;
        }
        switch (layer) {
            case "global":
                return base;
            case "site":
                return siteResolved;
            default:
                return layered(siteResolved, pagePatch);
        }
    }, [base, siteResolved, layer, pagePatch]);

    const draft = edited !== null && edited.key === key ? edited.draft : start;
    const identityDraft =
        identity !== null && identity.key === key
            ? identity
            : { key, name: template?.name ?? "", pageKind: template?.pageKind ?? "" };

    const specDirty = start !== null && draft !== null && patchBetween(start, draft) !== undefined;
    const identityDirty =
        template !== undefined &&
        (identityDraft.name !== template.name || identityDraft.pageKind !== template.pageKind);
    const dirty = specDirty || (layer === "global" && identityDirty);

    const pending = update.isPending || override.isPending || dropOverride.isPending;
    const thrown = layer === "global" ? update.error : (override.error ?? dropOverride.error);
    const isDefault = site.data?.site.defaults.templateId === templateId;

    const edit = (patch: Partial<SpecDraft>): void => {
        if (draft === null) {
            return;
        }
        setEdited({ key, draft: { ...draft, ...patch } });
    };

    const revert = (): void => {
        setEdited(null);
        setIdentity(null);
        update.reset();
        override.reset();
        dropOverride.reset();
    };

    const save = (): void => {
        if (template === undefined || draft === null || base === null || siteResolved === null) {
            return;
        }
        if (layer === "global") {
            update.mutate({
                id: template.id,
                name: identityDraft.name,
                pageKind: identityDraft.pageKind,
                spec: specOf(draft),
            });
            return;
        }
        const against = layer === "site" ? base : siteResolved;
        const held = layer === "site" ? siteOverride : pageOverride;
        const targetId = layer === "site" ? siteId : (pageId ?? "");
        const patch = patchBetween(against, draft);
        if (patch === undefined) {
            if (held !== null) {
                dropOverride.mutate({ id: held.id });
            }
            return;
        }
        override.mutate({ templateId: template.id, scope: layer, targetId, patch });
    };

    const makeDefault = (): void => {
        const held = site.data?.site;
        if (held === undefined || template === undefined) {
            return;
        }
        updateSite.mutate({ id: held.id, defaults: { ...held.defaults, templateId: template.id } });
    };

    if (detail.isPending) {
        return (
            <div className="p-4">
                <SkeletonRows rows={12} label={copy.templates.loading} />
            </div>
        );
    }

    if (template === undefined || draft === null || base === null || start === null || siteResolved === null) {
        return (
            <div className="p-4">
                <Banner
                    tone="warn"
                    title={
                        failure(detail.error).code === "NOT_FOUND"
                            ? copy.templates.editor.notFound
                            : failure(detail.error).message
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

    const layers: LayerView =
        layer === "page"
            ? { editing: "page", site: null, page: pagePatch, base: siteResolved }
            : { editing: layer, site: sitePatch, page: null, base };
    const formError = formErrorOf(thrown);

    return (
        <div className="flex h-full min-h-0 flex-col">
            <header className="flex h-8 shrink-0 items-center gap-2 border-b border-hairline px-3">
                <button
                    type="button"
                    onClick={() => {
                        void navigate(`/s/${siteId}/templates`);
                    }}
                    className="inline-flex shrink-0 items-center gap-0.5 text-xs text-ink-dim hover:text-ink"
                >
                    {copy.templates.editor.back}
                    <ChevronRightIcon size={13} />
                </button>
                <span className="min-w-0 truncate text-xs font-semibold text-ink">{template.name}</span>
                <span className="shrink-0 font-mono text-2xs text-ink-faint">{template.pageKind}</span>
                <StatusBadge tone={scopeTone(template.scope)} dot={false}>
                    {scopeLabel(template.scope)}
                </StatusBadge>
                <span className="shrink-0 font-mono text-2xs text-ink-faint">
                    {copy.templates.versionLabel(template.version)}
                </span>
                {pageId === null ? null : (
                    <StatusBadge tone="info" dot={false} className="min-w-0">
                        <span className="truncate font-mono">{copy.templates.editor.forPage(page.data?.page.path ?? copy.templates.overrides.unknownPage)}</span>
                    </StatusBadge>
                )}
                {isDefault ? (
                    <StatusBadge tone="accent" icon={VerifiedIcon}>
                        {copy.templates.editor.isDefault}
                    </StatusBadge>
                ) : site.data === undefined ? null : (
                    <Button size="sm" variant="ghost" busy={updateSite.isPending} onClick={makeDefault}>
                        {copy.templates.editor.makeDefault}
                    </Button>
                )}
                <div className="ml-auto flex shrink-0 items-center gap-2">
                    {dirty ? (
                        <span className="text-2xs text-warn">{copy.templates.editor.unsaved}</span>
                    ) : null}
                    <Segmented
                        label={copy.templates.layer.label}
                        value={layer}
                        options={layerOptions(pageId)}
                        onValueChange={setLayer}
                    />
                    <Button
                        size="sm"
                        variant="ghost"
                        icon={SmartToyIcon}
                        onClick={() => {
                            askAgent(copy.agent.ask.template(template.name, template.id));
                        }}
                    >
                        {copy.templates.askAgent}
                    </Button>
                    <Button size="sm" variant="ghost" disabled={!dirty} onClick={revert}>
                        {copy.templates.editor.revert}
                    </Button>
                    <Button size="sm" variant="primary" disabled={!dirty} busy={pending} onClick={save}>
                        {copy.templates.editor.save}
                    </Button>
                    <Button
                        size="sm"
                        variant="danger"
                        icon={DeleteIcon}
                        onClick={() => {
                            setConfirming(true);
                        }}
                    >
                        {copy.templates.editor.deleteTemplate}
                    </Button>
                </div>
            </header>
            <div className="flex min-h-0 flex-1">
                <div className="flex min-w-0 flex-1 flex-col">
                    <div className="flex shrink-0 flex-col gap-2 border-b border-hairline px-3 py-2">
                        <p className="text-xs text-ink-dim">{layerHints[layer]}</p>
                        {formError === null ? null : <p className="text-xs text-danger">{formError}</p>}
                        {layer === "global" ? (
                            <div className="grid grid-cols-2 gap-2">
                                <Field label={copy.templates.editor.name} error={fieldErrorOf(thrown, "name")}>
                                    {(control) => (
                                        <Input
                                            id={control.id}
                                            aria-describedby={control["aria-describedby"]}
                                            invalid={control.invalid}
                                            value={identityDraft.name}
                                            onChange={(event) => {
                                                setIdentity({ ...identityDraft, key, name: event.target.value });
                                            }}
                                        />
                                    )}
                                </Field>
                                <Field
                                    label={copy.templates.editor.pageKind}
                                    error={fieldErrorOf(thrown, "pageKind")}
                                >
                                    {(control) => (
                                        <Input
                                            id={control.id}
                                            aria-describedby={control["aria-describedby"]}
                                            invalid={control.invalid}
                                            mono={true}
                                            value={identityDraft.pageKind}
                                            onChange={(event) => {
                                                setIdentity({ ...identityDraft, key, pageKind: event.target.value });
                                            }}
                                        />
                                    )}
                                </Field>
                            </div>
                        ) : null}
                    </div>
                    <div className="flex shrink-0 border-b border-hairline px-1">
                        <Tabs label={copy.templates.title} items={tabs} value={tab} onValueChange={setTab} />
                    </div>
                    <div className="flex min-h-0 flex-1 flex-col">
                        <TabPanel label={copy.templates.editor.tabs.sections} active={tab === "sections"}>
                            <div className="p-3">
                                <SectionsForm draft={draft} layers={layers} error={thrown} onChange={edit} />
                            </div>
                        </TabPanel>
                        <TabPanel label={copy.templates.editor.tabs.content} active={tab === "content"}>
                            <div className="p-3">
                                <ContentForm draft={draft} layers={layers} error={thrown} onChange={edit} />
                            </div>
                        </TabPanel>
                        <TabPanel label={copy.templates.editor.tabs.links} active={tab === "links"}>
                            <div className="p-3">
                                <LinksForm draft={draft} layers={layers} error={thrown} onChange={edit} />
                            </div>
                        </TabPanel>
                        <TabPanel label={copy.templates.editor.tabs.meta} active={tab === "meta"}>
                            <div className="p-3">
                                <MetaForm draft={draft} layers={layers} error={thrown} onChange={edit} />
                            </div>
                        </TabPanel>
                        <TabPanel label={copy.templates.editor.tabs.models} active={tab === "models"}>
                            <div className="p-3">
                                <ModelsForm draft={draft} layers={layers} error={thrown} onChange={edit} />
                            </div>
                        </TabPanel>
                        <TabPanel label={copy.templates.editor.tabs.recipe} active={tab === "recipe"}>
                            <div className="p-3">
                                <RecipeForm draft={draft} layers={layers} error={thrown} onChange={edit} />
                            </div>
                        </TabPanel>
                    </div>
                </div>
                <aside className="flex w-80 shrink-0 flex-col border-l border-hairline">
                    <div className="flex shrink-0 border-b border-hairline px-1">
                        <Tabs
                            label={copy.templates.editor.aside.preview}
                            items={asideTabs}
                            value={asideTab}
                            onValueChange={setAsideTab}
                        />
                    </div>
                    <div className="flex min-h-0 flex-1 flex-col">
                        <TabPanel label={copy.templates.editor.aside.preview} active={asideTab === "preview"}>
                            <div className="p-3">
                                <PagePreview draft={draft} />
                            </div>
                        </TabPanel>
                        <TabPanel label={copy.templates.editor.aside.overrides} active={asideTab === "overrides"}>
                            <div className="p-3">
                                <OverridesPanel base={base} siteOverride={siteOverride} pageOverrides={pageOverrides} />
                            </div>
                        </TabPanel>
                    </div>
                </aside>
            </div>
            <Dialog
                open={confirming}
                onOpenChange={setConfirming}
                title={copy.templates.editor.deleteTitle}
                description={copy.templates.editor.deleteBody}
                confirmLabel={copy.templates.editor.deleteConfirm}
                cancelLabel={copy.templates.editor.cancel}
                destructive={true}
                icon={DeleteIcon}
                busy={remove.isPending}
                onConfirm={() => {
                    remove.mutate(
                        { id: template.id },
                        {
                            onSuccess: () => {
                                setConfirming(false);
                                void navigate(`/s/${siteId}/templates`);
                            },
                        },
                    );
                }}
            />
        </div>
    );
}
