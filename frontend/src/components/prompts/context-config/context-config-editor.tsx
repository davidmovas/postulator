"use client";

import { useEffect, useState, useMemo } from "react";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Lock } from "lucide-react";
import {
    ContextConfig,
    ContextFieldDefinition,
    ContextFieldValue,
    PromptCategory,
} from "@/models/prompts";
import { promptService } from "@/services/prompts";

export interface ContextConfigEditorProps {
    category: PromptCategory;
    config: ContextConfig;
    onChange: (config: ContextConfig) => void;
    disabled?: boolean;
    /**
     * Mode:
     * - "edit" = for prompt creation/editing (show all fields, allow enabling/disabling)
     * - "override" = for usage time (only show enabled fields from base config, allow value changes)
     */
    mode?: "edit" | "override";
    /** Base config from prompt (used in override mode) */
    baseConfig?: ContextConfig;
    /** Compact display for inline usage */
    compact?: boolean;
}

export function ContextConfigEditor({
    category,
    config,
    onChange,
    disabled = false,
    mode = "edit",
    baseConfig,
    compact = false,
}: ContextConfigEditorProps) {
    const [fields, setFields] = useState<ContextFieldDefinition[]>([]);
    const [defaultConfig, setDefaultConfig] = useState<ContextConfig>({});
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        async function loadFields() {
            setIsLoading(true);
            try {
                const response = await promptService.getContextFields(category);
                setFields(response.fields);
                setDefaultConfig(response.defaultConfig);

                // Initialize config with defaults if empty (only in edit mode)
                if (mode === "edit" && Object.keys(config).length === 0) {
                    onChange(response.defaultConfig);
                }
            } catch (error) {
                console.error("Failed to load context fields:", error);
            } finally {
                setIsLoading(false);
            }
        }
        loadFields();
    }, [category]);

    // Sort fields: required first, then by group
    const sortedFields = useMemo(() => {
        const sorted = [...fields];
        sorted.sort((a, b) => {
            // Required fields first
            if (a.required && !b.required) return -1;
            if (!a.required && b.required) return 1;
            // Then by group order
            const groupOrder = ["content", "site", "settings", "style", "advanced"];
            const aIdx = groupOrder.indexOf(a.group || "content");
            const bIdx = groupOrder.indexOf(b.group || "content");
            return aIdx - bIdx;
        });
        return sorted;
    }, [fields]);

    // In override mode, filter to only show enabled fields from base config
    const visibleFields = useMemo(() => {
        if (mode === "override" && baseConfig) {
            return sortedFields.filter(field => {
                const baseValue = baseConfig[field.key];
                return field.required || baseValue?.enabled;
            });
        }
        return sortedFields;
    }, [sortedFields, mode, baseConfig]);

    const updateField = (key: string, value: Partial<ContextFieldValue>) => {
        const currentValue = config[key] || defaultConfig[key] || { enabled: false };
        onChange({
            ...config,
            [key]: { ...currentValue, ...value },
        });
    };

    const getFieldValue = (key: string): ContextFieldValue | undefined => {
        return config[key] || (mode === "override" ? baseConfig?.[key] : defaultConfig[key]);
    };

    if (isLoading) {
        return (
            <div className={compact ? "py-2" : "p-4"}>
                <span className="text-sm text-muted-foreground">Loading...</span>
            </div>
        );
    }

    if (visibleFields.length === 0) {
        return (
            <div className={compact ? "py-2" : "p-4"}>
                <span className="text-sm text-muted-foreground">
                    No context fields available.
                </span>
            </div>
        );
    }

    // Separate required and optional fields
    const requiredFields = visibleFields.filter(f => f.required);
    const optionalFields = visibleFields.filter(f => !f.required);

    return (
        <div className={compact ? "space-y-3" : "space-y-4"}>
            {/* Required Fields */}
            {requiredFields.length > 0 && (
                <div className="space-y-3">
                    {!compact && (
                        <div className="flex items-center gap-2">
                            <Lock className="h-3.5 w-3.5 text-muted-foreground" />
                            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                                Required
                            </span>
                        </div>
                    )}
                    <div className={compact ? "space-y-2" : "space-y-3"}>
                        {requiredFields.map((field) => (
                            <ContextField
                                key={field.key}
                                field={field}
                                value={getFieldValue(field.key)}
                                onChange={(value) => updateField(field.key, value)}
                                disabled={disabled}
                                compact={compact}
                                mode={mode}
                            />
                        ))}
                    </div>
                </div>
            )}

            {/* Separator */}
            {requiredFields.length > 0 && optionalFields.length > 0 && (
                <Separator />
            )}

            {/* Optional Fields */}
            {optionalFields.length > 0 && (
                <div className="space-y-3">
                    {!compact && (
                        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                            Optional
                        </span>
                    )}
                    <div className={compact ? "space-y-2" : "space-y-3"}>
                        {optionalFields.map((field) => (
                            <ContextField
                                key={field.key}
                                field={field}
                                value={getFieldValue(field.key)}
                                onChange={(value) => updateField(field.key, value)}
                                disabled={disabled}
                                compact={compact}
                                mode={mode}
                            />
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
}

interface ContextFieldProps {
    field: ContextFieldDefinition;
    value: ContextFieldValue | undefined;
    onChange: (value: Partial<ContextFieldValue>) => void;
    disabled?: boolean;
    compact?: boolean;
    mode: "edit" | "override";
}

function ContextField({ field, value, onChange, disabled, compact, mode }: ContextFieldProps) {
    const isEnabled = value?.enabled ?? false;
    const fieldValue = value?.value ?? field.defaultValue ?? "";
    const isRequired = field.required;

    // Optional fields can be toggled in both modes, required fields are always enabled
    const canToggle = !isRequired;

    if (field.type === "checkbox") {
        return (
            <div className={`flex items-center gap-3 ${compact ? "py-1" : "py-1.5"}`}>
                <Checkbox
                    id={field.key}
                    checked={isEnabled}
                    onCheckedChange={(checked) => onChange({ enabled: !!checked })}
                    disabled={disabled || !canToggle}
                />
                <div className="flex-1 min-w-0">
                    <Label
                        htmlFor={field.key}
                        className={`cursor-pointer ${compact ? "text-sm" : ""}`}
                    >
                        {field.label}
                    </Label>
                    {!compact && (
                        <p className="text-xs text-muted-foreground mt-0.5">
                            {field.description}
                        </p>
                    )}
                </div>
                {isRequired && (
                    <Badge variant="outline" className="text-[10px] shrink-0">
                        Required
                    </Badge>
                )}
            </div>
        );
    }

    if (field.type === "input") {
        return (
            <div className={`space-y-2 ${compact ? "py-1" : "py-1.5"}`}>
                <div className="flex items-center gap-3">
                    <Checkbox
                        id={`${field.key}-enabled`}
                        checked={isEnabled}
                        onCheckedChange={(checked) => onChange({ enabled: !!checked })}
                        disabled={disabled || !canToggle}
                    />
                    <Label htmlFor={field.key} className={compact ? "text-sm" : ""}>
                        {field.label}
                    </Label>
                    {isRequired && (
                        <Badge variant="outline" className="text-[10px]">
                            Required
                        </Badge>
                    )}
                </div>
                {isEnabled && (
                    <Input
                        id={field.key}
                        value={fieldValue}
                        onChange={(e) => onChange({ value: e.target.value })}
                        placeholder={field.defaultValue || field.description}
                        disabled={disabled}
                        className={`ml-7 ${compact ? "h-8 text-sm" : ""}`}
                    />
                )}
            </div>
        );
    }

    if (field.type === "select") {
        return (
            <div className={`space-y-2 ${compact ? "py-1" : "py-1.5"}`}>
                <div className="flex items-center gap-3">
                    <Checkbox
                        id={`${field.key}-enabled`}
                        checked={isEnabled}
                        onCheckedChange={(checked) => onChange({ enabled: !!checked })}
                        disabled={disabled || !canToggle}
                    />
                    <Label htmlFor={field.key} className={compact ? "text-sm" : ""}>
                        {field.label}
                    </Label>
                    {isRequired && (
                        <Badge variant="outline" className="text-[10px]">
                            Required
                        </Badge>
                    )}
                </div>
                {isEnabled && (
                    <Select
                        value={fieldValue}
                        onValueChange={(val) => onChange({ value: val })}
                        disabled={disabled}
                    >
                        <SelectTrigger className={`ml-7 w-[calc(100%-1.75rem)] ${compact ? "h-8 text-sm" : ""}`}>
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            {field.options?.map((option) => (
                                <SelectItem key={option.value} value={option.value}>
                                    {option.label}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                )}
            </div>
        );
    }

    // textarea type - typically not used in prompts, but support it
    return null;
}

/**
 * Compact inline display of enabled context fields as badges
 * Used to show current config at a glance
 */
export function ContextConfigBadges({
    config,
    className = "",
}: {
    config: ContextConfig;
    className?: string;
}) {
    const enabledKeys = Object.entries(config)
        .filter(([_, v]) => v.enabled)
        .map(([k]) => k);

    if (enabledKeys.length === 0) {
        return null;
    }

    return (
        <div className={`flex flex-wrap gap-1 ${className}`}>
            {enabledKeys.map((key) => (
                <Badge key={key} variant="secondary" className="text-xs font-normal">
                    {key}
                </Badge>
            ))}
        </div>
    );
}
