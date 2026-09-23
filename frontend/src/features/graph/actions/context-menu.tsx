import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import type { ReactElement } from "react";

import type { Point } from "../../../canvas/viewport.js";
import { copy } from "../../../copy/index.js";
import { AddIcon, AddLinkIcon, CenterFocusWeakIcon, DeleteIcon, UnfoldLessIcon, UnfoldMoreIcon } from "../../../ui/index.js";
import type { IconComponent } from "../../../ui/index.js";

export interface MenuTarget {
    id: string;
    at: Point;
    expanded: boolean;
    hasChildren: boolean;
}

export interface ContextMenuProps {
    target: MenuTarget | null;
    onClose: () => void;
    onAddChild: (id: string) => void;
    onConnect: (id: string) => void;
    onToggle: (id: string) => void;
    onCenter: (id: string) => void;
    onDelete: (id: string) => void;
}

interface ItemProps {
    icon: IconComponent;
    label: string;
    danger?: boolean;
    onSelect: () => void;
}

function Item({ icon: Icon, label, danger = false, onSelect }: ItemProps): ReactElement {
    return (
        <DropdownMenu.Item
            className={`flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1 text-xs outline-none data-[highlighted]:bg-inset ${danger ? "text-danger" : "text-ink-soft data-[highlighted]:text-ink"}`}
            onSelect={onSelect}
        >
            <Icon size={14} />
            {label}
        </DropdownMenu.Item>
    );
}

export function EntityContextMenu({ target, onClose, onAddChild, onConnect, onToggle, onCenter, onDelete }: ContextMenuProps): ReactElement {
    return (
        <DropdownMenu.Root
            open={target !== null}
            onOpenChange={(open) => {
                if (!open) {
                    onClose();
                }
            }}
        >
            <DropdownMenu.Trigger asChild={true}>
                <span
                    aria-hidden={true}
                    className="pointer-events-none absolute h-px w-px"
                    style={{ left: target?.at.x ?? 0, top: target?.at.y ?? 0 }}
                />
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
                <DropdownMenu.Content
                    align="start"
                    sideOffset={2}
                    aria-label={copy.graph.menu.label}
                    className="z-30 min-w-44 rounded-md border border-edge bg-raised p-1 data-[state=open]:animate-fade-in"
                >
                    {target === null ? null : (
                        <>
                            <Item icon={AddIcon} label={copy.graph.menu.addChild} onSelect={() => onAddChild(target.id)} />
                            <Item icon={AddLinkIcon} label={copy.graph.menu.connect} onSelect={() => onConnect(target.id)} />
                            {target.hasChildren ? (
                                <Item
                                    icon={target.expanded ? UnfoldLessIcon : UnfoldMoreIcon}
                                    label={target.expanded ? copy.graph.menu.fold : copy.graph.menu.unfold}
                                    onSelect={() => onToggle(target.id)}
                                />
                            ) : null}
                            <Item icon={CenterFocusWeakIcon} label={copy.graph.menu.center} onSelect={() => onCenter(target.id)} />
                            <DropdownMenu.Separator className="my-1 h-px bg-hairline" />
                            <Item icon={DeleteIcon} label={copy.graph.menu.delete} danger={true} onSelect={() => onDelete(target.id)} />
                        </>
                    )}
                </DropdownMenu.Content>
            </DropdownMenu.Portal>
        </DropdownMenu.Root>
    );
}
