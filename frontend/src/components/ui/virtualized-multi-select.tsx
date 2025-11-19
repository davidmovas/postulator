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
    maxSelected?: number;
    hidePlaceholderWhenSelected?: boolean;
}

export function VirtualizedMultiSelect({
    options,
    value,
    onChange,
    placeholder = "Select items...",
    searchPlaceholder = "Search...",
    disabled = false,
    className,
    maxSelected,
    hidePlaceholderWhenSelected = true,
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
            options={options}
            placeholder={placeholder}
            commandProps={{
                label: searchPlaceholder,
            }}
            disabled={disabled}
            className={className}
            maxSelected={maxSelected}
            hidePlaceholderWhenSelected={hidePlaceholderWhenSelected}
            badgeClassName="bg-primary/50 text-primary-foreground border-primary/60 shadow-sm hover:bg-primary/70 hover:border-primary/80 hover:shadow-md transition-all duration-200 transform hover:scale-102"
            emptyIndicator={
                <p className="text-center text-sm text-muted-foreground py-4">
                    No results found
                </p>
            }

            loadingIndicator={
                <div className="flex items-center justify-center py-4">
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary"></div>
                    <span className="ml-2 text-sm text-muted-foreground">Loading...</span>
                </div>
            }
        />
    );
}