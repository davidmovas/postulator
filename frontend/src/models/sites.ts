import { dto } from "@/wailsjs/wailsjs/go/models";

export interface Site {
    id: number;
    name: string;
    url: string;
    wpUsername: string;
    wpPassword: string;
    status: string;
    autoHealthCheck: boolean;
    lastHealthCheck: string;
    healthStatus: string;
    createdAt: string;
    updatedAt: string;
}

export interface SiteCreateInput {
    name: string;
    url: string;
    wpUsername: string;
    wpPassword: string;
    autoHealthCheck: boolean;
}

export interface SiteUpdateInput extends Partial<SiteCreateInput> {
    id: number;
    status?: string;
}

export function mapSite(x: dto.Site): Site {
    return {
        id: x.id,
        name: x.name,
        url: x.url,
        wpUsername: x.wpUsername,
        wpPassword: x.wpPassword,
        status: x.status,
        autoHealthCheck: x.autoHealthCheck,
        lastHealthCheck: x.lastHealthCheck,
        healthStatus: x.healthStatus,
        createdAt: x.createdAt,
        updatedAt: x.updatedAt,
    };
}