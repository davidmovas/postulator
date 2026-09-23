import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { Template } from "../../data/types.js";
import {
    Button,
    DeleteIcon,
    EditNoteIcon,
    IconButton,
    Menu,
    MoreHorizIcon,
    OpenInNewIcon,
    RestartAltIcon,
    RightPanelCloseIcon,
    RightPanelOpenIcon,
    Segmented,
    SmartToyIcon,
    StatusBadge,
    VerifiedIcon,
    WarningIcon,
} from "../../ui/index.js";
import type { SegmentedOption } from "../../ui/index.js";
import type { ConflictKind } from "./conflict.js";
import type { Layer } from "./patch.js";

export interface EditorBadgesProps {
    template: Template;
    isDefault: boolean;
    dirty: boolean;
}

export function EditorBadges({ template, isDefault, dirty }: EditorBadgesProps): ReactElement {
    return (
        <span className="flex items-center gap-1.5">
            <StatusBadge tone="muted" dot={false}>
                {copy.templates.versionLabel(template.version)}
            </StatusBadge>
            {isDefault ? (
                <StatusBadge tone="accent" icon={VerifiedIcon}>
                    {copy.templates.siteDefault}
                </StatusBadge>
            ) : null}
            {dirty ? (
                <StatusBadge tone="warn">
                    {copy.templates.editor.unsaved}
                </StatusBadge>
            ) : null}
        </span>
    );
}

export interface EditorActionsProps {
    layer: Layer;
    pageId: string | null;
    pagePath: string;
    dirty: boolean;
    saving: boolean;
    isDefault: boolean;
    canSetDefault: boolean;
    skeleton: boolean;
    onToggleSkeleton: () => void;
    onOpenPage: () => void;
    onAskAgent: () => void;
    onSave: () => void;
    onRevert: () => void;
    onRename: () => void;
    onSetDefault: () => void;
    onDelete: () => void;
}

export function EditorActions({
    layer,
    pageId,
    pagePath,
    dirty,
    saving,
    isDefault,
    canSetDefault,
    skeleton,
    onToggleSkeleton,
    onOpenPage,
    onAskAgent,
    onSave,
    onRevert,
    onRename,
    onSetDefault,
    onDelete,
}: EditorActionsProps): ReactElement {
    return (
        <>
            <Button
                icon={OpenInNewIcon}
                disabled={pageId === null}
                title={pageId === null ? copy.templates.editor.openPageRefused : pagePath}
                onClick={onOpenPage}
            >
                {copy.templates.editor.openPage}
            </Button>
            <Button icon={SmartToyIcon} onClick={onAskAgent}>
                {copy.templates.askAgent}
            </Button>
            <Button
                variant="primary"
                data-template-save={true}
                disabled={!dirty}
                busy={saving}
                title={dirty ? undefined : copy.templates.editor.saveNothing}
                onClick={onSave}
            >
                {copy.templates.editor.save}
            </Button>
            <Menu
                label={copy.templates.moreActions}
                trigger={
                    <IconButton icon={MoreHorizIcon} label={copy.templates.moreActions} variant="ghost" />
                }
                items={[
                    {
                        key: "revert",
                        label: copy.templates.editor.revert,
                        icon: RestartAltIcon,
                        disabled: !dirty,
                        onSelect: onRevert,
                    },
                    {
                        key: "rename",
                        label: copy.templates.editor.changeKind,
                        icon: EditNoteIcon,
                        disabled: layer !== "global",
                        onSelect: onRename,
                    },
                    {
                        key: "skeleton",
                        label: skeleton
                            ? copy.templates.editor.hideSkeleton
                            : copy.templates.editor.showSkeleton,
                        icon: skeleton ? RightPanelCloseIcon : RightPanelOpenIcon,
                        onSelect: onToggleSkeleton,
                    },
                    {
                        key: "default",
                        label: copy.templates.setDefault,
                        icon: VerifiedIcon,
                        disabled: isDefault || !canSetDefault,
                        onSelect: onSetDefault,
                    },
                    { kind: "separator", key: "sep" },
                    {
                        key: "delete",
                        label: copy.templates.editor.deleteTemplate,
                        icon: DeleteIcon,
                        danger: true,
                        onSelect: onDelete,
                    },
                ]}
            />
        </>
    );
}

export interface ConflictBarProps {
    kind: ConflictKind;
    saving: boolean;
    onReload: () => void;
    onOverwrite: () => void;
}

export function ConflictBar({ kind, saving, onReload, onOverwrite }: ConflictBarProps): ReactElement {
    return (
        <div
            role="alert"
            data-template-conflict={kind}
            className="flex h-9 shrink-0 items-center gap-3 border-b border-warn-border bg-warn-soft px-4"
        >
            <WarningIcon size={16} className="shrink-0 text-warn" />
            <span className="text-xs font-semibold text-warn">{copy.templates.editor.conflict.title}</span>
            <span className="min-w-0 flex-1 truncate text-xs text-ink-soft">
                {kind === "template"
                    ? copy.templates.editor.conflict.template
                    : copy.templates.editor.conflict.override}
            </span>
            <Button size="sm" onClick={onReload}>
                {copy.templates.editor.conflict.reload}
            </Button>
            <Button size="sm" variant="danger" busy={saving} onClick={onOverwrite}>
                {copy.templates.editor.conflict.overwrite}
            </Button>
        </div>
    );
}

function layerOptions(pageId: string | null, path: string): readonly SegmentedOption<Layer>[] {
    const listed: SegmentedOption<Layer>[] = [
        { value: "global", label: copy.templates.layer.global, title: copy.templates.layer.globalHint },
        { value: "site", label: copy.templates.layer.site, title: copy.templates.layer.siteHint },
    ];
    if (pageId !== null) {
        listed.push({
            value: "page",
            label: copy.templates.layer.page,
            title: path === "" ? copy.templates.layer.pageHint : copy.templates.editor.forPage(path),
        });
    }
    return listed;
}

export interface LayerChooserProps {
    layer: Layer;
    pageId: string | null;
    pagePath: string;
    onChoose: (layer: Layer) => void;
}

export function LayerChooser({ layer, pageId, pagePath, onChoose }: LayerChooserProps): ReactElement {
    return (
        <div className="flex items-center self-center">
            <Segmented
                label={copy.templates.layer.label}
                value={layer}
                options={layerOptions(pageId, pagePath)}
                onValueChange={onChoose}
            />
        </div>
    );
}
