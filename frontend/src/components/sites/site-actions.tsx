import { Button } from "@/components/ui/button";
import {
    RiLockPasswordLine,
    RiPencilLine,
    RiPulseLine,
    RiToolsLine,
} from "@remixicon/react";
import { Trash2 } from "lucide-react";

interface SiteActionsProps {
    onCheckHealth: () => void;
    onEdit: () => void;
    onChangePassword: () => void;
    onOpenWordPress: () => void;
    onDelete: () => void;
    isLoading?: boolean;
}

export function SiteActions({
    onCheckHealth,
    onEdit,
    onChangePassword,
    onOpenWordPress,
    onDelete,
    isLoading
}: SiteActionsProps) {
    const actions = [
        {
            icon: RiPulseLine,
            label: "Check Health",
            onClick: onCheckHealth,
            variant: "outline" as const,
            disabled: isLoading
        },
        {
            icon: RiPencilLine,
            label: "Edit",
            onClick: onEdit,
            variant: "outline" as const
        },
        {
            icon: RiLockPasswordLine,
            label: "Set Password",
            onClick: onChangePassword,
            variant: "outline" as const
        },
        {
            icon: RiToolsLine,
            label: "Admin Panel",
            onClick: onOpenWordPress,
            variant: "outline" as const
        },
        {
            icon: Trash2,
            label: "Delete Site",
            onClick: onDelete,
            variant: "destructive" as const
        }
    ];

    return (
        <div className="flex flex-wrap justify-end gap-2">
            {actions.map((action, index) => (
                <Button
                    key={index}
                    variant={action.variant}
                    onClick={action.onClick}
                    disabled={action.disabled}
                    className="flex items-center gap-2 h-9"
                >
                    <action.icon className="w-4 h-4" />
                    {action.label}
                </Button>
            ))}
        </div>
    );
}