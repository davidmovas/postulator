"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { Table } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DataTableFilterConfig } from "./data-table";
import { Cross2Icon } from "@radix-ui/react-icons";
import { DataTableFacetedFilter } from "@/components/table/data-table-faceted-filter";
import { DataTableViewOptions } from "@/components/table/data-table-view-options";
import React from "react";

interface DataTableToolbarProps<TData> {
    table: Table<TData>;
    searchKey?: string;
    searchPlaceholder?: string;
    filters?: DataTableFilterConfig[];
    toolbarActions?: React.ReactNode;
    enableViewOptions?: boolean;
    // Server-side search callback
    onSearchChange?: (search: string) => void;
}

export function DataTableToolbar<TData>({
        table,
        searchKey,
        searchPlaceholder = "Search...",
        filters = [],
        toolbarActions,
        enableViewOptions = true,
        onSearchChange,
    }: DataTableToolbarProps<TData>) {
    const isFiltered = table.getState().columnFilters.length > 0;
    const isSearchable = !!searchKey || !!onSearchChange;

    // Local state for controlled input with debounce for server-side search
    const [searchValue, setSearchValue] = useState("");
    const debounceRef = useRef<NodeJS.Timeout | null>(null);

    const handleSearchChange = useCallback((value: string) => {
        setSearchValue(value);

        if (onSearchChange) {
            // Server-side search with debounce
            if (debounceRef.current) {
                clearTimeout(debounceRef.current);
            }
            debounceRef.current = setTimeout(() => {
                onSearchChange(value);
            }, 300);
        } else if (searchKey) {
            // Client-side search (immediate)
            table.getColumn(searchKey)?.setFilterValue(value);
        }
    }, [onSearchChange, searchKey, table]);

    // Cleanup debounce on unmount
    useEffect(() => {
        return () => {
            if (debounceRef.current) {
                clearTimeout(debounceRef.current);
            }
        };
    }, []);

    return (
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            {/* Left side - Search and Filters */}
            <div className="flex flex-1 flex-col gap-2 sm:flex-row sm:items-center">
                {/* Search Input */}
                {isSearchable && (
                    <div className="flex items-center gap-2">
                        <Input
                            placeholder={searchPlaceholder}
                            value={onSearchChange ? searchValue : ((table.getColumn(searchKey!)?.getFilterValue() as string) ?? "")}
                            onChange={(event) => handleSearchChange(event.target.value)}
                            className="h-9 w-full sm:w-[250px]"
                        />
                    </div>
                )}

                {/* Faceted Filters */}
                {filters.length > 0 && (
                    <div className="flex flex-wrap gap-2">
                        {filters.map((filter) => {
                            const column = table.getColumn(filter.columnId);
                            if (!column) return null;

                            return (
                                <DataTableFacetedFilter
                                    key={filter.columnId}
                                    column={column}
                                    title={filter.title}
                                    options={filter.options || []}
                                />
                            );
                        })}
                    </div>
                )}

                {/* Reset Filters */}
                {isFiltered && (
                    <Button
                        variant="ghost"
                        onClick={() => table.resetColumnFilters()}
                        className="h-9 px-2 lg:px-3"
                    >
                        Reset
                        <Cross2Icon className="ml-2 h-4 w-4" />
                    </Button>
                )}
            </div>

            {/* Right side - Actions and View Options */}
            <div className="flex items-center gap-2">
                {/* Custom Toolbar Actions */}
                {toolbarActions}

                {/* Column Visibility Toggle */}
                {enableViewOptions && <DataTableViewOptions table={table} />}
            </div>
        </div>
    );
}