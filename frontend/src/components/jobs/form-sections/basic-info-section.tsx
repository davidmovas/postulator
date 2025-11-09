"use client";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { JobCreateInput } from "@/models/jobs";

interface BasicInfoSectionProps {
    formData: Partial<JobCreateInput>;
    onUpdate: (updates: Partial<JobCreateInput>) => void;
    prompts: any[];
    providers: any[];
    site?: any;
    sites?: any[];
}

export function BasicInfoSection({ formData, onUpdate, prompts, providers, site, sites }: BasicInfoSectionProps) {
    return (
        <Card>
            <CardHeader>
                <CardTitle>Basic Information</CardTitle>
                <CardFooter>
                    Configure the basic settings for your content generation job
                </CardFooter>
            </CardHeader>
            <CardContent className="space-y-4">
                {/* Job Name */}
                <div className="space-y-2">
                    <Label htmlFor="name">Job Name</Label>
                    <Input
                        id="name"
                        placeholder="e.g., Daily Blog Posts, Weekly Newsletters"
                        value={formData.name || ""}
                        onChange={(e) => onUpdate({ name: e.target.value })}
                    />
                </div>

                {/* Site Selection (только для глобальной страницы) */}
                {sites && (
                    <div className="space-y-2">
                        <Label htmlFor="site">Target Site</Label>
                        <Select
                            value={formData.siteId?.toString()}
                            onValueChange={(value) => onUpdate({ siteId: parseInt(value) })}
                        >
                            <SelectTrigger>
                                <SelectValue placeholder="Select a site" />
                            </SelectTrigger>
                            <SelectContent>
                                {sites.map(site => (
                                    <SelectItem key={site.id} value={site.id.toString()}>
                                        {site.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
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

                {/* Prompt Selection */}
                <div className="space-y-2">
                    <Label htmlFor="prompt">AI Prompt</Label>
                    <Select
                        value={formData.promptId?.toString()}
                        onValueChange={(value) => onUpdate({ promptId: parseInt(value) })}
                    >
                        <SelectTrigger>
                            <SelectValue placeholder="Select a prompt" />
                        </SelectTrigger>
                        <SelectContent>
                            {prompts.map(prompt => (
                                <SelectItem key={prompt.id} value={prompt.id.toString()}>
                                    {prompt.name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>

                {/* AI Provider Selection */}
                <div className="space-y-2">
                    <Label htmlFor="provider">AI Provider</Label>
                    <Select
                        value={formData.aiProviderId?.toString()}
                        onValueChange={(value) => onUpdate({ aiProviderId: parseInt(value) })}
                    >
                        <SelectTrigger>
                            <SelectValue placeholder="Select an AI provider" />
                        </SelectTrigger>
                        <SelectContent>
                            {providers.map(provider => (
                                <SelectItem key={provider.id} value={provider.id.toString()}>
                                    {provider.name} ({provider.model})
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
            </CardContent>
        </Card>
    );
}