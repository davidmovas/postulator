"use client";

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
}

export function DataTableToolbar<TData>({
        table,
        searchKey,
        searchPlaceholder = "Search...",
        filters = [],
        toolbarActions,
        enableViewOptions = true,
    }: DataTableToolbarProps<TData>) {
    const isFiltered = table.getState().columnFilters.length > 0;
    const isSearchable = !!searchKey;

    return (
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            {/* Left side - Search and Filters */}
            <div className="flex flex-1 flex-col gap-2 sm:flex-row sm:items-center">
                {/* Search Input */}
                {isSearchable && (
                    <div className="flex items-center gap-2">
                        <Input
                            placeholder={searchPlaceholder}
                            value={(table.getColumn(searchKey)?.getFilterValue() as string) ?? ""}
                            onChange={(event) =>
                                table.getColumn(searchKey)?.setFilterValue(event.target.value)
                            }
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