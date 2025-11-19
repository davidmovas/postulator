"use client";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { JobCreateInput } from "@/models/jobs";
import { VirtualizedMultiSelect } from "@/components/ui/virtualized-multi-select";
import { RiWordpressFill } from "@remixicon/react";
import { TOPIC_STRATEGY_REUSE_WITH_VARIATION, TOPIC_STRATEGY_UNIQUE } from "@/constants/topics";

interface ContentStrategySectionProps {
    formData: Partial<JobCreateInput>;
    onUpdate: (updates: Partial<JobCreateInput>) => void;
    topics: any[] | null;
    categories: any[] | null;
    onSyncCategories: () => void;
}

export function ContentStrategySection({
    formData,
    onUpdate,
    topics,
    categories,
    onSyncCategories
}: ContentStrategySectionProps) {
    return (
        <Card>
            <CardHeader>
                <CardTitle>Content Strategy</CardTitle>
                <CardFooter>
                    Configure how topics and categories are selected for content generation
                </CardFooter>
            </CardHeader>
            <CardContent className="space-y-6">
                {/* Topic Strategy */}
                <div className="space-y-4">
                    <div className="flex items-center justify-between">
                        <Label className="text-base">Topic Strategy</Label>
                        <Badge variant="outline">
                            {formData.topics?.length || 0} topics selected
                        </Badge>
                    </div>

                    <RadioGroup
                        value={formData.topicStrategy}
                        onValueChange={(value) => onUpdate({ topicStrategy: value })}
                        className="space-y-3"
                    >
                        <div className="flex items-center space-x-2">
                            <RadioGroupItem value={TOPIC_STRATEGY_UNIQUE} id="unique" />
                            <Label htmlFor="unique" className="flex-1">
                                <div className="font-medium">Unique</div>
                                <div className="text-sm text-muted-foreground">
                                    Use each topic only once, in sequential order
                                </div>
                            </Label>
                        </div>

                        <div className="flex items-center space-x-2">
                            <RadioGroupItem value={TOPIC_STRATEGY_REUSE_WITH_VARIATION} id="reuse" />
                            <Label htmlFor="reuse" className="flex-1">
                                <div className="font-medium">Reuse with Variation</div>
                                <div className="text-sm text-muted-foreground">
                                    Reuse topics with AI-generated variations
                                </div>
                            </Label>
                        </div>
                    </RadioGroup>

                    {/* Topic Selection */}
                    <div className="space-y-2">
                        <div className="flex items-center justify-between">
                            <Label>Select Topics</Label>
                            <div className="flex items-center gap-2">
                                <span className="text-xs text-muted-foreground">Use all</span>
                                <Switch
                                    disabled={!topics || topics.length === 0}
                                    checked={(formData.topics?.length || 0) > 0 && !!topics && (formData.topics?.length === topics.length)}
                                    onCheckedChange={(checked) => {
                                        if (!topics) return;
                                        if (checked) {
                                            onUpdate({ topics: topics.map(t => t.id) });
                                        } else {
                                            onUpdate({ topics: [] });
                                        }
                                    }}
                                />
                            </div>
                        </div>
                        {Array.isArray(topics) && topics.length === 0 ? (
                            <div className="space-y-2">
                                <VirtualizedMultiSelect
                                    options={[]}
                                    value={[]}
                                    onChange={() => {}}
                                    placeholder="No topics available"
                                    className="border-destructive"
                                    disabled
                                />
                                <p className="text-xs text-destructive">
                                    You need to create at least one topic before creating a job.
                                </p>
                            </div>
                        ) : (
                            <>
                                <VirtualizedMultiSelect
                                    options={(topics || []).map(topic => ({
                                        value: topic.id.toString(),
                                        label: topic.title
                                    }))}
                                    value={formData.topics?.map(t => t.toString()) || []}
                                    onChange={(values) => onUpdate({
                                        topics: values.map(v => parseInt(v))
                                    })}
                                    placeholder="Search and select topics..."
                                    searchPlaceholder="Type to search topics..."
                                    disabled={!topics}
                                />
                                <p className="text-xs text-muted-foreground">
                                    {topics ? `${topics.length} topics available` : "Loading topics..."}
                                </p>
                            </>
                        )}
                    </div>
                </div>

                {/* Category Strategy */}
                <div className="space-y-4 border-t pt-4">
                    <div className="flex items-center justify-between">
                        <Label className="text-base">Category Strategy</Label>
                        <div className="flex items-center gap-2">
                            <Badge variant="outline">
                                {formData.categories?.length || 0} categories selected
                            </Badge>
                            <Button
                                variant="wordpress"
                                size="sm"
                                onClick={onSyncCategories}
                                className="h-8"
                                title="Sync categories from WordPress"
                            >
                                <RiWordpressFill className="h-4 w-4 mr-1" />
                                Sync from WordPress
                            </Button>
                        </div>
                    </div>

                    <RadioGroup
                        value={formData.categoryStrategy}
                        onValueChange={(value) => onUpdate({ categoryStrategy: value })}
                        className="space-y-3"
                    >
                        <div className="flex items-center space-x-2">
                            <RadioGroupItem value="fixed" id="fixed" />
                            <Label htmlFor="fixed" className="flex-1">
                                <div className="font-medium">Fixed</div>
                                <div className="text-sm text-muted-foreground">
                                    Always use the same categories
                                </div>
                            </Label>
                        </div>

                        <div className="flex items-center space-x-2">
                            <RadioGroupItem value="random" id="random" />
                            <Label htmlFor="random" className="flex-1">
                                <div className="font-medium">Random</div>
                                <div className="text-sm text-muted-foreground">
                                    Randomly select from available categories
                                </div>
                            </Label>
                        </div>

                        <div className="flex items-center space-x-2">
                            <RadioGroupItem value="rotate" id="rotate" />
                            <Label htmlFor="rotate" className="flex-1">
                                <div className="font-medium">Rotate</div>
                                <div className="text-sm text-muted-foreground">
                                    Cycle through categories in order
                                </div>
                            </Label>
                        </div>
                    </RadioGroup>

                    {/* Category Selection (always required regardless of strategy) */}
                    <div className="space-y-2">
                        <Label>Select Categories</Label>
                        <VirtualizedMultiSelect
                            options={(categories || []).map(category => ({
                                value: category.id.toString(),
                                label: category.name
                            }))}
                            value={formData.categories?.map(c => c.toString()) || []}
                            onChange={(values) => onUpdate({
                                categories: values.map(v => parseInt(v))
                            })}
                            placeholder="Select categories..."
                            disabled={!categories}
                        />
                        <p className="text-xs text-muted-foreground">
                            {categories ? `Select at least one category. ${categories.length} categories available` : "Loading categories..."}
                        </p>
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}