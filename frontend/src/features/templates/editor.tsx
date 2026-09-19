import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router";

import { failure } from "../../data/errors.js";
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
    cx,
    DeleteIcon,
    Dialog,
    Field,
    Input,
    SkeletonRows,
    SmartToyIcon,
    StatusBadge,
    TabPanel,
    Tabs,
} from "../../ui/index.js";
import type { TabDefinition } from "../../ui/index.js";
import { askAgent } from "../agent/dock-state.js";
import { fieldErrorOf, formErrorOf } from "./controls.js";
import { ModelsForm } from "./models-form.js";
import { OverridesPanel } from "./overrides-panel.js";
import { layered, patchBetween, patchObject } from "./patch.js";
import type { LayerView } from "./provenance.js";
import { RecipeForm } from "./recipe-form.js";
import { ContentForm, LinksForm, MetaForm } from "./rules-form.js";
import { scopeLabel, scopeTone } from "./labels.js";
import { SectionsForm } from "./sections-form.js";
import type { SpecDraft } from "./spec.js";
import { draftOf, specOf } from "./spec.js";

type EditorTab = "sections" | "content" | "links" | "meta" | "models" | "recipe";

type EditLayer = "global" | "site";

const tabs: readonly TabDefinition<EditorTab>[] = [
    { value: "sections", label: copy.templates.editor.tabs.sections },
    { value: "content", label: copy.templates.editor.tabs.content },
    { value: "links", label: copy.templates.editor.tabs.links },
    { value: "meta", label: copy.templates.editor.tabs.meta },
    { value: "models", label: copy.templates.editor.tabs.models },
    { value: "recipe", label: copy.templates.editor.tabs.recipe },
];

interface LayerButtonProps {
    layer: EditLayer;
    current: EditLayer;
    label: string;
    onSelect: (layer: EditLayer) => void;
}

function LayerButton({ layer, current, label, onSelect }: LayerButtonProps): ReactElement {
    const active = layer === current;
    return (
        <button
            type="button"
            aria-pressed={active}
            onClick={() => {
                onSelect(layer);
            }}
            className={cx(
                "inline-flex h-6 items-center rounded-md px-2 text-xs font-medium transition-colors duration-100",
                active ? "bg-raised text-ink" : "text-ink-dim hover:bg-inset hover:text-ink",
            )}
        >
            {label}
        </button>
    );
}

export function TemplateEditorScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const siteId = params.siteId ?? "";
    const templateId = params.templateId ?? "";
    const detail = useTemplate(templateId);
    const update = useUpdateTemplate();
    const override = useSetOverride();
    const dropOverride = useDeleteOverride();
    const remove = useDeleteTemplate();

    const [layer, setLayer] = useState<EditLayer>("global");
    const [tab, setTab] = useState<EditorTab>("sections");
    const [confirming, setConfirming] = useState(false);
    const [identity, setIdentity] = useState<{ key: string; name: string; pageKind: string } | null>(null);
    const [edited, setEdited] = useState<{ key: string; draft: SpecDraft } | null>(null);

    const template = detail.data?.template;
    const overrides = useMemo(() => detail.data?.overrides ?? [], [detail.data]);
    const siteOverride = useMemo(
        () => overrides.find((held) => held.scope === "site" && held.targetId === siteId) ?? null,
        [overrides, siteId],
    );
    const pageOverrides = useMemo(() => overrides.filter((held) => held.scope === "page"), [overrides]);
    const sitePatch = useMemo(() => patchObject(siteOverride?.patch), [siteOverride]);
    const base = useMemo(() => (template === undefined ? null : draftOf(template.spec)), [template]);

    const key = `${template?.id ?? ""}|${template?.updatedAt ?? ""}|${layer}|${siteOverride?.updatedAt ?? ""}`;
    const start = useMemo(() => {
        if (base === null) {
            return null;
        }
        return layer === "global" ? base : layered(base, sitePatch);
    }, [base, layer, sitePatch]);

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
    const thrown = layer === "global" ? update.error : override.error;

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
    };

    const save = (): void => {
        if (template === undefined || draft === null || base === null) {
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
        const patch = patchBetween(base, draft);
        if (patch === undefined) {
            if (siteOverride !== null) {
                dropOverride.mutate({ id: siteOverride.id });
            }
            return;
        }
        override.mutate({ templateId: template.id, scope: "site", targetId: siteId, patch });
    };

    if (detail.isPending) {
        return (
            <div className="p-4">
                <SkeletonRows rows={12} label={copy.templates.loading} />
            </div>
        );
    }

    if (template === undefined || draft === null || base === null || start === null) {
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

    const layers: LayerView = { editing: layer, site: sitePatch, page: null, base };
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
                <div className="ml-auto flex shrink-0 items-center gap-2">
                    {dirty ? (
                        <span className="text-2xs text-warn">{copy.templates.editor.unsaved}</span>
                    ) : null}
                    <div className="flex items-center gap-0.5 rounded-md bg-inset p-0.5">
                        <LayerButton
                            layer="global"
                            current={layer}
                            label={copy.templates.layer.global}
                            onSelect={setLayer}
                        />
                        <LayerButton
                            layer="site"
                            current={layer}
                            label={copy.templates.layer.site}
                            onSelect={setLayer}
                        />
                    </div>
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
                        <p className="text-xs text-ink-dim">
                            {layer === "global" ? copy.templates.layer.globalHint : copy.templates.layer.siteHint}
                        </p>
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
                    <Tabs
                        className="min-h-0 flex-1"
                        label={copy.templates.title}
                        value={tab}
                        onValueChange={setTab}
                        tabs={tabs}
                    >
                        <TabPanel value="sections" className="p-3">
                            <SectionsForm draft={draft} layers={layers} error={thrown} onChange={edit} />
                        </TabPanel>
                        <TabPanel value="content" className="p-3">
                            <ContentForm draft={draft} layers={layers} error={thrown} onChange={edit} />
                        </TabPanel>
                        <TabPanel value="links" className="p-3">
                            <LinksForm draft={draft} layers={layers} error={thrown} onChange={edit} />
                        </TabPanel>
                        <TabPanel value="meta" className="p-3">
                            <MetaForm draft={draft} layers={layers} error={thrown} onChange={edit} />
                        </TabPanel>
                        <TabPanel value="models" className="p-3">
                            <ModelsForm draft={draft} layers={layers} error={thrown} onChange={edit} />
                        </TabPanel>
                        <TabPanel value="recipe" className="p-3">
                            <RecipeForm draft={draft} layers={layers} error={thrown} onChange={edit} />
                        </TabPanel>
                    </Tabs>
                </div>
                <aside className="w-80 shrink-0 overflow-auto border-l border-hairline p-3">
                    <OverridesPanel base={base} siteOverride={siteOverride} pageOverrides={pageOverrides} />
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
