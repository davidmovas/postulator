import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { DeleteIcon, Dialog } from "../../ui/index.js";
import { IdentityDialog } from "./identity.js";

export interface EditorDialogsProps {
    name: string;
    pageKind: string;
    renaming: boolean;
    renameBusy: boolean;
    renameError: unknown;
    onRenameOpenChange: (open: boolean) => void;
    onRename: (identity: { name: string; pageKind: string }) => void;
    leaving: boolean;
    onLeaveOpenChange: (open: boolean) => void;
    onLeave: () => void;
    deleting: boolean;
    deleteBusy: boolean;
    onDeleteOpenChange: (open: boolean) => void;
    onDelete: () => void;
}

export function EditorDialogs({
    name,
    pageKind,
    renaming,
    renameBusy,
    renameError,
    onRenameOpenChange,
    onRename,
    leaving,
    onLeaveOpenChange,
    onLeave,
    deleting,
    deleteBusy,
    onDeleteOpenChange,
    onDelete,
}: EditorDialogsProps): ReactElement {
    return (
        <>
            <IdentityDialog
                open={renaming}
                name={name}
                pageKind={pageKind}
                busy={renameBusy}
                error={renameError}
                onOpenChange={onRenameOpenChange}
                onApply={onRename}
            />
            <Dialog
                open={leaving}
                onOpenChange={onLeaveOpenChange}
                title={copy.templates.editor.leaveTitle}
                description={copy.templates.editor.leaveBody}
                confirmLabel={copy.templates.editor.leaveConfirm}
                cancelLabel={copy.templates.editor.leaveCancel}
                destructive={true}
                onConfirm={onLeave}
            />
            <Dialog
                open={deleting}
                onOpenChange={onDeleteOpenChange}
                title={copy.templates.editor.deleteTitle}
                description={copy.templates.editor.deleteBody}
                confirmLabel={copy.templates.editor.deleteConfirm}
                cancelLabel={copy.templates.editor.cancel}
                destructive={true}
                icon={DeleteIcon}
                busy={deleteBusy}
                onConfirm={onDelete}
            />
        </>
    );
}
