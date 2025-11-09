"use client";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { JobCreateInput } from "@/models/jobs";
import { RotateCcw, Filter } from "lucide-react";
import { VirtualizedMultiSelect } from "@/components/ui/virtualized-multi-select";

interface ContentStrategySectionProps {
    formData: Partial<JobCreateInput>;
    onUpdate: (updates: Partial<JobCreateInput>) => void;
    topics: any[];
    categories: any[];
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
                            <RadioGroupItem value="unique" id="unique" />
                            <Label htmlFor="unique" className="flex-1">
                                <div className="font-medium">Unique</div>
                                <div className="text-sm text-muted-foreground">
                                    Use each topic only once, in sequential order
                                </div>
                            </Label>
                        </div>

                        <div className="flex items-center space-x-2">
                            <RadioGroupItem value="reuse_with_variation" id="variation" />
                            <Label htmlFor="variation" className="flex-1">
                                <div className="font-medium">Reuse with Variation</div>
                                <div className="text-sm text-muted-foreground">
                                    Reuse topics with AI-generated variations
                                </div>
                            </Label>
                        </div>
                    </RadioGroup>

                    {/* Topic Selection */}
                    <div className="space-y-2">
                        <Label>Select Topics</Label>
                        <VirtualizedMultiSelect
                            options={topics.map(topic => ({
                                value: topic.id.toString(),
                                label: topic.title
                            }))}
                            value={formData.topics?.map(t => t.toString()) || []}
                            onChange={(values) => onUpdate({
                                topics: values.map(v => parseInt(v))
                            })}
                            placeholder="Search and select topics..."
                            searchPlaceholder="Type to search topics..."
                        />
                        <p className="text-xs text-muted-foreground">
                            {topics.length} topics available
                        </p>
                    </div>
                </div>

                {/* Category Strategy */}
                <div className="space-y-4 border-t pt-4">
                    <div className="flex items-center justify-between">
                        <Label className="text-base">Category Strategy</Label>
                        <div className="flex items-center gap-2">
                            <Badge variant="outline">
                                {formData.categories?.length || 0} categories
                            </Badge>
                            <Button
                                variant="outline"
                                size="sm"
                                onClick={onSyncCategories}
                                className="h-8"
                            >
                                <RotateCcw className="h-3 w-3 mr-1" />
                                Sync
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

                    {/* Category Selection (только для Fixed стратегии) */}
                    {formData.categoryStrategy === 'fixed' && (
                        <div className="space-y-2">
                            <Label>Select Categories</Label>
                            <VirtualizedMultiSelect
                                options={categories.map(category => ({
                                    value: category.id.toString(),
                                    label: category.name
                                }))}
                                value={formData.categories?.map(c => c.toString()) || []}
                                onChange={(values) => onUpdate({
                                    categories: values.map(v => parseInt(v))
                                })}
                                placeholder="Select categories..."
                            />
                            <p className="text-xs text-muted-foreground">
                                {categories.length} categories available
                            </p>
                        </div>
                    )}
                </div>
            </CardContent>
        </Card>
    );
}