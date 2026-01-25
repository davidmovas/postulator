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
    mode?: "edit" | "override";
    baseConfig?: ContextConfig;
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

    const sortedFields = useMemo(() => {
        const sorted = [...fields];
        sorted.sort((a, b) => {
            if (a.required && !b.required) return -1;
            if (!a.required && b.required) return 1;
            const groupOrder = ["content", "constraints", "site", "settings", "style", "advanced"];
            const aIdx = groupOrder.indexOf(a.group || "content");
            const bIdx = groupOrder.indexOf(b.group || "content");
            return aIdx - bIdx;
        });
        return sorted;
    }, [fields]);

    const { constraintFields, otherFields } = useMemo(() => {
        const constraints = sortedFields.filter(f => f.group === "constraints");
        const others = sortedFields.filter(f => f.group !== "constraints");
        return { constraintFields: constraints, otherFields: others };
    }, [sortedFields]);

    const updateField = (key: string, value: Partial<ContextFieldValue>) => {
        const currentValue = config[key] || defaultConfig[key] || { enabled: false };
        onChange({
            ...config,
            [key]: { ...currentValue, ...value },
        });
    };

    const getFieldValue = (key: string): ContextFieldValue | undefined => {
        return config[key] || baseConfig?.[key] || defaultConfig[key];
    };

    if (isLoading) {
        return (
            <div className={compact ? "py-2" : "p-4"}>
                <span className="text-sm text-muted-foreground">Loading...</span>
            </div>
        );
    }

    if (sortedFields.length === 0) {
        return (
            <div className={compact ? "py-2" : "p-4"}>
                <span className="text-sm text-muted-foreground">
                    No context fields available.
                </span>
            </div>
        );
    }

    const requiredFields = otherFields.filter(f => f.required);
    const optionalFields = otherFields.filter(f => !f.required);

    return (
        <div className={compact ? "space-y-3" : "space-y-4"}>
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

            {requiredFields.length > 0 && (optionalFields.length > 0 || constraintFields.length > 0) && (
                <Separator />
            )}

            {constraintFields.length > 0 && (
                <div className="space-y-2">
                    <Label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                        Link Constraints
                    </Label>
                    <div className="grid grid-cols-2 gap-3">
                        {constraintFields.map((field) => (
                            <div key={field.key} className="space-y-1">
                                <Label className="text-xs">{field.label}</Label>
                                <Input
                                    type="number"
                                    min={0}
                                    max={50}
                                    value={getFieldValue(field.key)?.value ?? field.defaultValue ?? ""}
                                    onChange={(e) => updateField(field.key, { enabled: true, value: e.target.value })}
                                    disabled={disabled}
                                    className="h-8"
                                />
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {constraintFields.length > 0 && optionalFields.length > 0 && (
                <Separator />
            )}

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

    if (field.type === "input" || field.type === "number") {
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
                        type={field.type === "number" ? "number" : "text"}
                        value={fieldValue}
                        onChange={(e) => onChange({ value: e.target.value })}
                        placeholder={field.defaultValue || field.description}
                        disabled={disabled}
                        className={`ml-7 w-[calc(100%-1.75rem)] ${compact ? "h-8 text-sm" : ""}`}
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

    return null;
}

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
