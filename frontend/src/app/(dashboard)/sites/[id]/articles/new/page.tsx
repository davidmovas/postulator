"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { siteService } from "@/services/sites";
import { ArticleEditor } from "@/components/articles/editor/article-editor";

export default function NewArticlePage() {
    const params = useParams();
    const siteId = parseInt(params.id as string);

    const [site, setSite] = useState<any>(null);

    useEffect(() => {
        const loadSite = async () => {
            const siteData = await siteService.getSite(siteId);
            setSite(siteData);
        };
        loadSite();
    }, [siteId]);

    return (
        <ArticleEditor
            siteId={siteId}
            siteName={site?.name}
            siteUrl={site?.url?.replace(/^https?:\/\//, "").replace(/\/$/, "")}
        />
    );
}
