"use client";

import { useEffect, useState, Suspense } from "react";
import { useQueryId } from "@/hooks/use-query-param";
import { siteService } from "@/services/sites";
import { ArticleEditor } from "@/components/articles/editor/article-editor";

function NewArticlePageContent() {
    const siteId = useQueryId();

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

export default function NewArticlePage() {
    return (
        <Suspense fallback={
            <div className="flex items-center justify-center min-h-[400px]">
                <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
            </div>
        }>
            <NewArticlePageContent />
        </Suspense>
    );
}
