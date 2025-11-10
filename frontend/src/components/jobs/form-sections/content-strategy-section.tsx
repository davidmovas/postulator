"use client";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { JobCreateInput } from "@/models/jobs";
import { VirtualizedMultiSelect } from "@/components/ui/virtualized-multi-select";

// Simple inline WordPress icon (ri-wordpress-fill approximation)
function WordPressFillIcon(props: React.SVGProps<SVGSVGElement>) {
    return (
        <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true" {...props}>
            <path d="M12 2C6.477 2 2 6.477 2 12c0 4.243 2.64 7.87 6.37 9.334L5.48 9.873a4.06 4.06 0 0 1-.106-.922c0-1.224.711-2.138 1.826-2.138.77 0 1.133.535 1.133 1.175 0 .716-.455 1.786-.69 2.776-.196.833.414 1.511 1.232 1.511 1.477 0 2.614-1.934 2.614-4.218 0-1.737-1.17-3.037-3.296-3.037-2.4 0-3.894 1.794-3.894 3.803 0 .691.204 1.179.523 1.556.146.173.167.241.113.438-.037.144-.125.491-.162.629-.053.204-.217.277-.4.202-1.119-.457-1.638-1.68-1.638-3.052 0-2.272 1.915-4.993 5.717-4.993 3.053 0 5.056 2.209 5.056 4.58 0 3.136-1.744 5.482-4.313 5.482-.862 0-1.673-.466-1.949-.994l-.53 2.017c-.191.734-.713 1.65-1.064 2.21A9.999 9.999 0 1 0 12 2z" />
        </svg>
    );
}

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
                        <div className="flex items-center justify-between">
                            <Label>Select Topics</Label>
                            <div className="flex items-center gap-2">
                                <span className="text-xs text-muted-foreground">Use all</span>
                                <Switch
                                    checked={(formData.topics?.length || 0) > 0 && (formData.topics?.length === topics.length)}
                                    onCheckedChange={(checked) => {
                                        if (checked) {
                                            onUpdate({ topics: topics.map(t => t.id) });
                                        } else {
                                            onUpdate({ topics: [] });
                                        }
                                    }}
                                />
                            </div>
                        </div>
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
                                variant="wordpress"
                                size="sm"
                                onClick={onSyncCategories}
                                className="h-8"
                                title="Sync categories from WordPress"
                            >
                                <WordPressFillIcon className="h-4 w-4 mr-1" />
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

                    {/* Category Selection (always required regardless of strategy) */}
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
                            Select at least one category. {categories.length} categories available
                        </p>
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}