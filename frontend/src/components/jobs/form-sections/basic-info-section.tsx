"use client";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { SearchableSelect } from "@/components/ui/searchable-select";
import { Button } from "@/components/ui/button";
import { JobCreateInput } from "@/models/jobs";
import { generateJobName } from "@/lib/job-name-generator";
import { Sparkles } from "lucide-react";

interface BasicInfoSectionProps {
    formData: Partial<JobCreateInput>;
    onUpdate: (updates: Partial<JobCreateInput>) => void;
    prompts: any[] | null;
    providers: any[] | null;
    site?: any;
    sites?: any[];
    categories?: any[] | null;
    topics?: any[] | null;
}

export function BasicInfoSection({
    formData,
    onUpdate,
    prompts,
    providers,
    site,
    sites,
    categories,
    topics
}: BasicInfoSectionProps) {
    const promptsLoading = prompts == null;
    const providersLoading = providers == null;
    const noPrompts = !promptsLoading && prompts!.length === 0;
    const noProviders = !providersLoading && providers!.length === 0;

    const handleGenerateName = () => {
        const categoryNames = categories?.map(c => c.name) || [];
        const topicTitles = topics?.map(t => t.title) || [];
        const promptName = prompts?.find(p => p.id === formData.promptId)?.name;

        const generatedName = generateJobName(categoryNames, topicTitles, promptName);
        onUpdate({ name: generatedName });
    };

    return (
        <Card>
            <CardHeader>
                <CardTitle>Basic Information</CardTitle>
                <CardFooter>
                    Configure the basic settings for your content generation job
                </CardFooter>
            </CardHeader>
            <CardContent className="space-y-4">
                {/* Job Name with Generator */}
                <div className="space-y-2">
                    <div className="flex items-center justify-between">
                        <Label htmlFor="name">Job Name</Label>
                        <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            onClick={handleGenerateName}
                            disabled={!categories || !topics}
                            className="h-8"
                        >
                            <Sparkles className="h-3 w-3 mr-1" />
                            Generate
                        </Button>
                    </div>
                    <Input
                        id="name"
                        placeholder="e.g., Daily Blog Posts, Weekly Newsletters"
                        value={formData.name || ""}
                        onChange={(e) => onUpdate({ name: e.target.value })}
                    />
                    <p className="text-xs text-muted-foreground">
                        {!categories || !topics
                            ? "Select categories and topics to enable name generation"
                            : "Click Generate to create a meaningful job name automatically"
                        }
                    </p>
                </div>

                {/* Site Selection (только для глобальной страницы) */}
                {sites && (
                    <div className="space-y-2">
                        <Label htmlFor="site">Target Site</Label>
                        <SearchableSelect
                            options={sites.map((s) => ({ value: s.id.toString(), label: s.name }))}
                            value={formData.siteId?.toString()}
                            onChange={(val) => onUpdate({ siteId: parseInt(val) })}
                            placeholder="Search and select a site..."
                            searchPlaceholder="Type to search sites..."
                        />
                    </div>
                )}

                {/* Site Display (только для страницы сайта) */}
                {site && (
                    <div className="space-y-2">
                        <Label>Target Site</Label>
                        <div className="p-2 bg-muted rounded-md">
                            {site.name} ({site.url})
                        </div>
                    </div>
                )}

                {/* AI Provider Selection */}
                <div className="space-y-2">
                    <Label htmlFor="provider" className={noProviders ? "text-destructive" : undefined}>
                        AI Provider
                    </Label>
                    <Select
                        value={formData.aiProviderId?.toString()}
                        onValueChange={(value) => onUpdate({ aiProviderId: parseInt(value) })}
                        disabled={providersLoading || noProviders}
                    >
                        <SelectTrigger className={noProviders ? "border-destructive" : undefined}>
                            <SelectValue placeholder={
                                noProviders
                                    ? "No providers found. Please create one first"
                                    : "Select an AI provider"
                            } />
                        </SelectTrigger>
                        <SelectContent>
                            {(providers || []).map(provider => (
                                <SelectItem key={provider.id} value={provider.id.toString()}>
                                    {provider.name} ({provider.model})
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                    {noProviders && (
                        <div className="flex items-center gap-2 text-xs text-destructive">
                            You need to create an AI provider first.
                            <a href="/ai-providers" className="underline">Go to Providers</a>
                        </div>
                    )}
                </div>

                {/* Prompt Selection */}
                <div className="space-y-2">
                    <Label htmlFor="prompt" className={noPrompts ? "text-destructive" : undefined}>
                        AI Prompt
                    </Label>
                    <Select
                        value={formData.promptId?.toString()}
                        onValueChange={(value) => onUpdate({ promptId: parseInt(value) })}
                        disabled={promptsLoading || noPrompts}
                    >
                        <SelectTrigger className={noPrompts ? "border-destructive" : undefined}>
                            <SelectValue placeholder={
                                noPrompts
                                    ? "No prompts found. Please create one first"
                                    : "Select a prompt"
                            } />
                        </SelectTrigger>
                        <SelectContent>
                            {(prompts || []).map(prompt => (
                                <SelectItem key={prompt.id} value={prompt.id.toString()}>
                                    {prompt.name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                    {noPrompts && (
                        <div className="flex items-center gap-2 text-xs text-destructive">
                            You need to create a prompt first.
                            <a href="/prompts" className="underline">Go to Prompts</a>
                        </div>
                    )}
                </div>
            </CardContent>
        </Card>
    );
}