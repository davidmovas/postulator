import { Button } from "@/components/ui/button";
import {
    RiArticleLine,
    RiLockPasswordLine,
    RiPencilLine,
    RiPulseLine,
    RiTimerLine,
    RiToolsLine,
    RiLightbulbLine,
    RiChatThreadLine,
} from "@remixicon/react";
import { Trash2 } from "lucide-react";

interface SiteActionsProps {
    onCheckHealth: () => void;
    onEdit: () => void;
    onChangePassword: () => void;
    onViewArticles: () => void;
    onViewJobs: () => void;
    onOpenWordPress: () => void;
    onDelete: () => void;
    onViewTopics?: () => void;
    onViewCategories?: () => void;
    isLoading?: boolean;
}

export function SiteActions({
    onCheckHealth,
    onEdit,
    onChangePassword,
    onViewArticles,
    onViewJobs,
    onOpenWordPress,
    onDelete,
    onViewTopics,
    onViewCategories,
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
            icon: RiTimerLine,
            label: "Jobs",
            onClick: onViewJobs,
            variant: "outline" as const
        },
        {
            icon: RiArticleLine,
            label: "Articles",
            onClick: onViewArticles,
            variant: "outline" as const
        },
        {
            icon: RiChatThreadLine,
            label: "Categories",
            onClick: onViewCategories,
            variant: "outline" as const
        },
        {
            icon: RiLightbulbLine,
            label: "Topics",
            onClick: onViewTopics,
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
        <div className="flex flex-wrap gap-2">
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