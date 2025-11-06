"use client";

import { Provider } from "@/models/providers";
import { ProviderCard } from "./provider-card";

interface ProvidersListProps {
    providers: Provider[];
    onEdit: (provider: Provider) => void;
    onDelete: (provider: Provider) => void;
    onToggleStatus: (provider: Provider) => void;
    isLoading?: boolean;
}

export function ProvidersList({
    providers,
    onEdit,
    onDelete,
    onToggleStatus,
    isLoading = false
}: ProvidersListProps) {
    if (isLoading) {
        return (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {[...Array(6)].map((_, i) => (
                    <div key={i} className="animate-pulse">
                        <div className="h-48 bg-muted rounded-lg"></div>
                    </div>
                ))}
            </div>
        );
    }

    if (providers.length === 0) {
        return (
            <div className="text-center py-12">
                <div className="text-muted-foreground">
                    No providers found. Create your first provider to get started.
                </div>
            </div>
        );
    }

    return (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {providers.map((provider) => (
                <ProviderCard
                    key={provider.id}
                    provider={provider}
                    onEdit={onEdit}
                    onDelete={onDelete}
                    onToggleStatus={onToggleStatus}
                />
            ))}
        </div>
    );
}