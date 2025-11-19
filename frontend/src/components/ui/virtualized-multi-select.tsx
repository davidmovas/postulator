"use client";

import MultipleSelector, { Option } from '@/components/ui/multiselect';

interface VirtualizedMultiSelectProps {
    options: Option[];
    value: string[];
    onChange: (values: string[]) => void;
    placeholder?: string;
    searchPlaceholder?: string;
    disabled?: boolean;
    className?: string;
}

export function VirtualizedMultiSelect({
    options,
    value,
    onChange,
    placeholder = "Select items...",
    searchPlaceholder = "Search...",
    disabled = false,
    className
}: VirtualizedMultiSelectProps) {
    const selectedOptions = options.filter(option =>
        value.includes(option.value)
    );

    const handleChange = (selected: Option[]) => {
        onChange(selected.map(option => option.value));
    };

    return (
        <MultipleSelector
            value={selectedOptions}
            onChange={handleChange}
            defaultOptions={options}
            placeholder={placeholder}
            commandProps={{
                label: searchPlaceholder,
            }}
            disabled={disabled}
            className={className}
            emptyIndicator={
                <p className="text-center text-sm text-muted-foreground">
                    No results found
                </p>
            }
        />
    );
}