import { dto } from "@/wailsjs/wailsjs/go/models";

export interface Site {
    id: number;
    name: string;
    url: string;
    wpUsername: string;
    wpPassword: string;
    status: string;
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
        lastHealthCheck: x.lastHealthCheck,
        healthStatus: x.healthStatus,
        createdAt: x.createdAt,
        updatedAt: x.updatedAt,
    };
}