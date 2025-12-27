"use client";

import { useEffect, useState } from "react";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Collapsible,
    CollapsibleContent,
    CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { ChevronDown, ChevronRight } from "lucide-react";
import {
    ContextConfig,
    ContextFieldDefinition,
    ContextFieldValue,
    PromptCategory,
} from "@/models/prompts";
import { promptService } from "@/services/prompts";

interface ContextConfigEditorProps {
    category: PromptCategory;
    config: ContextConfig;
    onChange: (config: ContextConfig) => void;
    disabled?: boolean;
}

interface FieldGroup {
    name: string;
    label: string;
    fields: ContextFieldDefinition[];
}

const GROUP_LABELS: Record<string, string> = {
    content: "Content",
    site: "Site Info",
    settings: "Settings",
    style: "Style",
    advanced: "Advanced",
};

export function ContextConfigEditor({
    category,
    config,
    onChange,
    disabled = false,
}: ContextConfigEditorProps) {
    const [fields, setFields] = useState<ContextFieldDefinition[]>([]);
    const [defaultConfig, setDefaultConfig] = useState<ContextConfig>({});
    const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set(["content", "site"]));
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        async function loadFields() {
            setIsLoading(true);
            try {
                const response = await promptService.getContextFields(category);
                setFields(response.fields);
                setDefaultConfig(response.defaultConfig);

                // Initialize config with defaults if empty
                if (Object.keys(config).length === 0) {
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

    const updateField = (key: string, value: Partial<ContextFieldValue>) => {
        const currentValue = config[key] || defaultConfig[key] || { enabled: false };
        onChange({
            ...config,
            [key]: { ...currentValue, ...value },
        });
    };

    // Group fields by their group property
    const groups: FieldGroup[] = [];
    const groupMap = new Map<string, ContextFieldDefinition[]>();

    for (const field of fields) {
        const groupName = field.group || "content";
        if (!groupMap.has(groupName)) {
            groupMap.set(groupName, []);
        }
        groupMap.get(groupName)!.push(field);
    }

    const groupOrder = ["content", "site", "settings", "style", "advanced"];
    for (const name of groupOrder) {
        const groupFields = groupMap.get(name);
        if (groupFields && groupFields.length > 0) {
            groups.push({
                name,
                label: GROUP_LABELS[name] || name,
                fields: groupFields,
            });
        }
    }

    const toggleGroup = (groupName: string) => {
        const newExpanded = new Set(expandedGroups);
        if (newExpanded.has(groupName)) {
            newExpanded.delete(groupName);
        } else {
            newExpanded.add(groupName);
        }
        setExpandedGroups(newExpanded);
    };

    if (isLoading) {
        return (
            <div className="p-4 text-center text-muted-foreground">
                Loading context fields...
            </div>
        );
    }

    if (fields.length === 0) {
        return (
            <div className="p-4 text-center text-muted-foreground">
                No context fields available for this category.
            </div>
        );
    }

    return (
        <div className="space-y-2">
            {groups.map((group) => (
                <Collapsible
                    key={group.name}
                    open={expandedGroups.has(group.name)}
                    onOpenChange={() => toggleGroup(group.name)}
                >
                    <CollapsibleTrigger className="flex items-center gap-2 w-full p-2 hover:bg-muted/50 rounded-md">
                        {expandedGroups.has(group.name) ? (
                            <ChevronDown className="h-4 w-4" />
                        ) : (
                            <ChevronRight className="h-4 w-4" />
                        )}
                        <span className="font-medium text-sm">{group.label}</span>
                        <span className="text-xs text-muted-foreground">
                            ({group.fields.length} fields)
                        </span>
                    </CollapsibleTrigger>
                    <CollapsibleContent className="pl-6 space-y-3 pt-2">
                        {group.fields.map((field) => (
                            <ContextField
                                key={field.key}
                                field={field}
                                value={config[field.key] || defaultConfig[field.key]}
                                onChange={(value) => updateField(field.key, value)}
                                disabled={disabled}
                            />
                        ))}
                    </CollapsibleContent>
                </Collapsible>
            ))}
        </div>
    );
}

interface ContextFieldProps {
    field: ContextFieldDefinition;
    value: ContextFieldValue | undefined;
    onChange: (value: Partial<ContextFieldValue>) => void;
    disabled?: boolean;
}

function ContextField({ field, value, onChange, disabled }: ContextFieldProps) {
    const isEnabled = value?.enabled ?? false;
    const fieldValue = value?.value ?? field.defaultValue ?? "";

    switch (field.type) {
        case "checkbox":
            return (
                <div className="flex items-start gap-3">
                    <Checkbox
                        id={field.key}
                        checked={isEnabled}
                        onCheckedChange={(checked) => onChange({ enabled: !!checked })}
                        disabled={disabled || field.required}
                    />
                    <div className="space-y-1">
                        <Label
                            htmlFor={field.key}
                            className="text-sm font-medium cursor-pointer"
                        >
                            {field.label}
                            {field.required && (
                                <span className="text-muted-foreground ml-1">(required)</span>
                            )}
                        </Label>
                        <p className="text-xs text-muted-foreground">{field.description}</p>
                    </div>
                </div>
            );

        case "input":
            return (
                <div className="space-y-2">
                    <div className="flex items-center gap-3">
                        <Checkbox
                            id={`${field.key}-enabled`}
                            checked={isEnabled}
                            onCheckedChange={(checked) => onChange({ enabled: !!checked })}
                            disabled={disabled || field.required}
                        />
                        <Label htmlFor={field.key} className="text-sm font-medium">
                            {field.label}
                        </Label>
                    </div>
                    {isEnabled && (
                        <Input
                            id={field.key}
                            value={fieldValue}
                            onChange={(e) => onChange({ value: e.target.value })}
                            placeholder={field.defaultValue}
                            disabled={disabled}
                            className="ml-7"
                        />
                    )}
                    <p className="text-xs text-muted-foreground ml-7">{field.description}</p>
                </div>
            );

        case "select":
            return (
                <div className="space-y-2">
                    <div className="flex items-center gap-3">
                        <Checkbox
                            id={`${field.key}-enabled`}
                            checked={isEnabled}
                            onCheckedChange={(checked) => onChange({ enabled: !!checked })}
                            disabled={disabled || field.required}
                        />
                        <Label htmlFor={field.key} className="text-sm font-medium">
                            {field.label}
                        </Label>
                    </div>
                    {isEnabled && (
                        <Select
                            value={fieldValue}
                            onValueChange={(value) => onChange({ value })}
                            disabled={disabled}
                        >
                            <SelectTrigger className="ml-7 w-[calc(100%-1.75rem)]">
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
                    <p className="text-xs text-muted-foreground ml-7">{field.description}</p>
                </div>
            );

        case "textarea":
            return (
                <div className="space-y-2">
                    <div className="flex items-center gap-3">
                        <Checkbox
                            id={`${field.key}-enabled`}
                            checked={isEnabled}
                            onCheckedChange={(checked) => onChange({ enabled: !!checked })}
                            disabled={disabled || field.required}
                        />
                        <Label htmlFor={field.key} className="text-sm font-medium">
                            {field.label}
                        </Label>
                    </div>
                    {isEnabled && (
                        <Textarea
                            id={field.key}
                            value={fieldValue}
                            onChange={(e) => onChange({ value: e.target.value })}
                            placeholder={field.description}
                            disabled={disabled}
                            className="ml-7 min-h-[80px]"
                            rows={3}
                        />
                    )}
                    <p className="text-xs text-muted-foreground ml-7">{field.description}</p>
                </div>
            );

        default:
            return null;
    }
}
