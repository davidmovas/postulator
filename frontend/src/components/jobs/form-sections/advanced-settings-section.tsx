"use client";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { JobCreateInput } from "@/models/jobs";

interface AdvancedSettingsSectionProps {
    formData: Partial<JobCreateInput>;
    onUpdate: (updates: Partial<JobCreateInput>) => void;
}

export function AdvancedSettingsSection({ formData, onUpdate }: AdvancedSettingsSectionProps) {
    return (
        <Card>
            <CardHeader>
                <CardTitle>Advanced Settings</CardTitle>
                <CardFooter>
                    Additional configuration options for job execution
                </CardFooter>
            </CardHeader>
            <CardContent className="space-y-6">
                {/* Validation Toggle */}
                <div className="flex items-center justify-between">
                    <div className="space-y-1">
                        <Label htmlFor="validation" className="text-base">
                            Requires Validation
                        </Label>
                        <p className="text-sm text-muted-foreground">
                            Articles will be created as drafts for manual review before publishing
                        </p>
                    </div>
                    <Switch
                        id="validation"
                        checked={formData.requiresValidation || false}
                        onCheckedChange={(checked) => onUpdate({ requiresValidation: checked })}
                    />
                </div>

                {/* Jitter Settings */}
                <div className="space-y-4 border-t pt-4">
                    <div className="flex items-center justify-between">
                        <div className="space-y-1">
                            <Label htmlFor="jitter" className="text-base">
                                Enable Jitter
                            </Label>
                            <p className="text-sm text-muted-foreground">
                                Add random delay to prevent predictable execution patterns
                            </p>
                        </div>
                        <Switch
                            id="jitter"
                            checked={formData.jitterEnabled || false}
                            onCheckedChange={(checked) => onUpdate({ jitterEnabled: checked })}
                        />
                    </div>

                    {formData.jitterEnabled && (
                        <div className="space-y-2 pl-6">
                            <Label htmlFor="jitterMinutes">Jitter Window (minutes)</Label>
                            <div className="flex items-center gap-3">
                                <Input
                                    id="jitterMinutes"
                                    type="number"
                                    min="1"
                                    max="240"
                                    value={formData.jitterMinutes || 30}
                                    onChange={(e) => onUpdate({ jitterMinutes: parseInt(e.target.value) })}
                                    className="w-20"
                                />
                                <span className="text-sm text-muted-foreground">
                                    ± {formData.jitterMinutes || 30} minutes
                                </span>
                            </div>
                            <p className="text-xs text-muted-foreground">
                                Job will execute within this time window around the scheduled time
                            </p>
                        </div>
                    )}
                </div>

                {/* Summary */}
                <div className="border-t pt-4">
                    <h4 className="font-medium mb-2">Job Summary</h4>
                    <div className="space-y-2 text-sm text-muted-foreground">
                        <div>• {formData.topicStrategy === 'unique' ? 'Unique topics' : 'Reused topics with variations'}</div>
                        <div>• {formData.categoryStrategy ? `${formData.categoryStrategy.charAt(0).toUpperCase()}${formData.categoryStrategy.slice(1)}` : 'Fixed'} categories</div>
                        <div>• {formData.requiresValidation ? 'Requires validation' : 'Auto-publish'}</div>
                        <div>• {formData.jitterEnabled ? `Jitter: ±${formData.jitterMinutes}min` : 'No jitter'}</div>
                        {formData.schedule && (
                            <div>• {`${formData.schedule.type.charAt(0).toUpperCase()}${formData.schedule.type.slice(1)}`} schedule</div>
                        )}
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}