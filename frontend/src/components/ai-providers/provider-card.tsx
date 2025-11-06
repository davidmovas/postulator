"use client";

import { Provider } from "@/models/providers";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem, DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Edit, Trash2, Power, Key, Cpu, MoreVertical } from "lucide-react";
import { formatDateTime } from "@/lib/time";
import { PROVIDER_TYPES } from "@/constants/providers";

interface ProviderCardProps {
    provider: Provider;
    onEdit: (provider: Provider) => void;
    onDelete: (provider: Provider) => void;
    onToggleStatus: (provider: Provider) => void;
}

export function ProviderCard({ provider, onEdit, onDelete, onToggleStatus }: ProviderCardProps) {
    const providerConfig = PROVIDER_TYPES[provider.type as keyof typeof PROVIDER_TYPES] || {
        label: provider.type,
        description: `${provider.type} provider`,
        themeColor: "#6B7280",
        icon: ""
    };

    const formatApiKey = (apiKey: string): string => {
        if (!apiKey) return "Not set";
        if (apiKey.length <= 10) return apiKey;
        return `${apiKey.slice(0, 4)}...${apiKey.slice(-6)}`;
    };

    return (
        <Card className="relative overflow-hidden border-2 transition-all duration-200 hover:shadow-lg flex flex-col h-full">
            {/* Background Icon */}
            {providerConfig.icon && (
                <div className="absolute right-2 top-2 opacity-10">
                    <div
                        className="w-32 h-32 bg-contain bg-no-repeat bg-center"
                        style={{ backgroundImage: `url(${providerConfig.icon})` }}
                    />
                </div>
            )}

            <CardHeader className="pb-3 relative z-10">
                <div className="flex items-start justify-between">
                    <div className="space-y-2">
                        <CardTitle className="text-lg font-bold">
                            {provider.name}
                        </CardTitle>
                        <p className="text-sm text-muted-foreground">
                            {providerConfig.description.replace('models', 'model')}
                        </p>
                    </div>
                    <div className="flex items-center gap-2">
                        <Badge
                            variant={provider.isActive ? "default" : "secondary"}
                            className={provider.isActive ? "bg-green-100 text-green-800 hover:bg-green-100" : ""}
                        >
                            {provider.isActive ? "Active" : "Inactive"}
                        </Badge>
                    </div>
                </div>
            </CardHeader>

            <CardContent className="space-y-3 pb-3 relative z-10 flex-1">
                <div className="flex items-center gap-2 text-sm">
                    <Cpu className="h-4 w-4 text-muted-foreground" />
                    <span className="font-medium">Model:</span>
                    <span className="text-muted-foreground">{provider.model}</span>
                </div>

                <div className="flex items-center gap-2 text-sm">
                    <Key className="h-4 w-4 text-muted-foreground" />
                    <span className="font-medium">API Key:</span>
                    <span className="text-muted-foreground font-mono">
                        {formatApiKey(provider.apiKey)}
                    </span>
                </div>

                <div className="text-xs text-muted-foreground space-y-1 mt-auto">
                    <div>Created: {formatDateTime(provider.createdAt)}</div>
                    <div>Updated: {formatDateTime(provider.updatedAt)}</div>
                </div>
            </CardContent>

            <CardFooter className="flex justify-end gap-2 pt-3 relative z-10">
                <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                        <Button variant="outline" size="sm">
                            <MoreVertical className="h-4 w-4" />
                        </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => onToggleStatus(provider)}>
                            <Power className="h-4 w-4 mr-2" />
                            {provider.isActive ? "Deactivate" : "Activate"}
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => onEdit(provider)}>
                            <Edit className="h-4 w-4 mr-2" />
                            Edit
                        </DropdownMenuItem>

                        <DropdownMenuSeparator />

                        <DropdownMenuItem
                            onClick={() => onDelete(provider)}
                            className="text-destructive focus:text-destructive"
                        >
                            <Trash2 className="h-4 w-4 mr-2" />
                            Delete
                        </DropdownMenuItem>
                    </DropdownMenuContent>
                </DropdownMenu>
            </CardFooter>
        </Card>
    );
}