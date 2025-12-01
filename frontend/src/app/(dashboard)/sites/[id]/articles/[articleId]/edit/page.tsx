"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { siteService } from "@/services/sites";
import { articleService } from "@/services/articles";
import { Article } from "@/models/articles";
import { ArticleEditor } from "@/components/articles/editor/article-editor";
import { Loader2 } from "lucide-react";

export default function EditArticlePage() {
    const params = useParams();
    const siteId = parseInt(params.id as string);
    const articleId = parseInt(params.articleId as string);

    const [site, setSite] = useState<any>(null);
    const [article, setArticle] = useState<Article | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const loadData = async () => {
            setIsLoading(true);
            setError(null);

            try {
                const [siteData, articleData] = await Promise.all([
                    siteService.getSite(siteId),
                    articleService.getArticle(articleId),
                ]);

                setSite(siteData);
                setArticle(articleData);
            } catch (err) {
                setError("Failed to load article");
                console.error(err);
            } finally {
                setIsLoading(false);
            }
        };

        loadData();
    }, [siteId, articleId]);

    if (isLoading) {
        return (
            <div className="flex items-center justify-center min-h-[400px]">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
        );
    }

    if (error || !article) {
        return (
            <div className="flex flex-col items-center justify-center min-h-[400px] gap-4">
                <p className="text-destructive">{error || "Article not found"}</p>
            </div>
        );
    }

    return (
        <ArticleEditor
            siteId={siteId}
            siteName={site?.name}
            siteUrl={site?.url?.replace(/^https?:\/\//, "").replace(/\/$/, "")}
            article={article}
        />
    );
}
