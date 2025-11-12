"use client";

import React, { useEffect, useState } from "react";
import {
    ColumnDef,
    ColumnFiltersState,
    flexRender,
    getCoreRowModel,
    getFilteredRowModel,
    getPaginationRowModel,
    getSortedRowModel,
    PaginationState,
    SortingState,
    useReactTable,
    VisibilityState,
    getFacetedUniqueValues,
    RowSelectionState,
} from "@tanstack/react-table";
import { ChevronDownIcon, ChevronUpIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { DataTablePagination } from "@/components/table/data-table-pagination";
import { DataTableToolbar } from "@/components/table/data-table-toolbar";

export interface DataTableFilterConfig {
    columnId: string;
    title: string;
    options?: Array<{
        label: string;
        value: string;
        icon?: React.ComponentType<{ className?: string }>;
    }>;
}

export interface DataTableProps<TData, TValue> {
    columns: ColumnDef<TData, TValue>[];
    data: TData[];
    searchKey?: string;
    searchPlaceholder?: string;
    filters?: DataTableFilterConfig[];
    showPagination?: boolean;
    defaultPageSize?: number;
    toolbarActions?: React.ReactNode;
    isLoading?: boolean;
    emptyMessage?: string;
    onRowSelectionChange?: (selectedRows: TData[]) => void;
    defaultSorting?: SortingState;
    enableViewOption?: boolean;
    rowSelectionResetKey?: number | string;
    // Optional expandable rows support
    enableRowExpand?: boolean;
    expandOnRowClick?: boolean;
    renderExpandedRow?: (data: TData) => React.ReactNode;
}

export function DataTable<TData, TValue>({
    columns,
    data,
    searchKey,
    searchPlaceholder = "Search...",
    filters = [],
    showPagination = true,
    defaultPageSize = 25,
    toolbarActions,
    isLoading = false,
    emptyMessage = "No results",
    onRowSelectionChange,
    defaultSorting = [],
    enableViewOption = true,
    rowSelectionResetKey,
    enableRowExpand = false,
    expandOnRowClick = false,
    renderExpandedRow,
}: DataTableProps<TData, TValue>) {
    const [sorting, setSorting] = useState<SortingState>(defaultSorting);
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
    const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
    const [rowSelection, setRowSelection] = useState<RowSelectionState>({});
    const [expandedRows, setExpandedRows] = useState<Record<string, boolean>>({});
    const [pagination, setPagination] = useState<PaginationState>({
        pageIndex: 0,
        pageSize: defaultPageSize,
    });

    useEffect(() => {
        if (rowSelectionResetKey !== undefined) {
            setRowSelection({});
            if (onRowSelectionChange) {
                onRowSelectionChange([] as unknown as TData[]);
            }
        }
    }, [rowSelectionResetKey]);

    const table = useReactTable({
        data,
        columns,
        getCoreRowModel: getCoreRowModel(),
        getPaginationRowModel: showPagination ? getPaginationRowModel() : undefined,
        onPaginationChange: setPagination,
        onSortingChange: setSorting,
        getSortedRowModel: getSortedRowModel(),
        onColumnFiltersChange: setColumnFilters,
        getFilteredRowModel: getFilteredRowModel(),
        onColumnVisibilityChange: setColumnVisibility,
        onRowSelectionChange: (updater) => {
            setRowSelection(updater);
            if (onRowSelectionChange) {
                const newSelection = typeof updater === 'function' ? updater(rowSelection) : updater;
                const selectedRows = Object.keys(newSelection)
                    .filter(key => newSelection[key])
                    .map(key => data[parseInt(key)]);
                onRowSelectionChange(selectedRows);
            }
        },
        getFacetedUniqueValues: getFacetedUniqueValues(),
        state: {
            sorting,
            columnFilters,
            columnVisibility,
            rowSelection,
            pagination,
        },
    });

    return (
        <div className="space-y-4">
            <DataTableToolbar
                table={table}
                searchKey={searchKey}
                searchPlaceholder={searchPlaceholder}
                filters={filters}
                toolbarActions={toolbarActions}
                enableViewOptions={enableViewOption}
            />

            <div className="overflow-hidden rounded-md border bg-background">
                <Table>
                    <TableHeader>
                        {table.getHeaderGroups().map((headerGroup) => (
                            <TableRow key={headerGroup.id} className="hover:bg-transparent">
                                {headerGroup.headers.map((header, index) => {
                                    const isLastColumn = header.column.id === "actions";
                                    const isSelectColumn = header.column.id === "select";

                                    return (
                                        <TableHead
                                            key={header.id}
                                            className={cn(
                                                "h-11",
                                                isLastColumn && "text-right",
                                                isSelectColumn && "w-[40px] px-4 text-center"
                                            )}
                                        >
                                            {header.isPlaceholder ? null : header.column.getCanSort() ? (
                                                <div
                                                    className={cn(
                                                        "flex h-full cursor-pointer items-center justify-between gap-2 select-none",
                                                        isLastColumn && "flex-row-reverse"
                                                    )}
                                                    onClick={header.column.getToggleSortingHandler()}
                                                    onKeyDown={(e) => {
                                                        if (
                                                            header.column.getCanSort() &&
                                                            (e.key === "Enter" || e.key === " ")
                                                        ) {
                                                            e.preventDefault();
                                                            header.column.getToggleSortingHandler()?.(e);
                                                        }
                                                    }}
                                                    tabIndex={0}
                                                >
                                                    {flexRender(
                                                        header.column.columnDef.header,
                                                        header.getContext()
                                                    )}
                                                    {{
                                                        asc: (
                                                            <ChevronUpIcon
                                                                className="shrink-0 opacity-60"
                                                                size={16}
                                                                aria-hidden="true"
                                                            />
                                                        ),
                                                        desc: (
                                                            <ChevronDownIcon
                                                                className="shrink-0 opacity-60"
                                                                size={16}
                                                                aria-hidden="true"
                                                            />
                                                        ),
                                                    }[header.column.getIsSorted() as string] ?? null}
                                                </div>
                                            ) : (
                                                <div className={cn(
                                                    "flex items-center",
                                                    isLastColumn && "justify-end"
                                                )}>
                                                    {flexRender(
                                                        header.column.columnDef.header,
                                                        header.getContext()
                                                    )}
                                                </div>
                                            )}
                                        </TableHead>
                                    );
                                })}
                            </TableRow>
                        ))}
                    </TableHeader>
                    <TableBody>
                        {isLoading ? (
                            <TableRow>
                                <TableCell colSpan={columns.length} className="h-24 text-center">
                                    Loading...
                                </TableCell>
                            </TableRow>
                        ) : table.getRowModel().rows?.length ? (
                            table.getRowModel().rows.map((row) => (
                                <>
                                    <TableRow
                                        key={row.id}
                                        data-state={row.getIsSelected() && "selected"}
                                        className={cn(enableRowExpand && expandOnRowClick && "cursor-pointer")}
                                        onClick={() => {
                                            if (enableRowExpand && expandOnRowClick && renderExpandedRow) {
                                                setExpandedRows((prev) => ({
                                                    ...prev,
                                                    [row.id]: !prev[row.id],
                                                }));
                                            }
                                        }}
                                    >
                                        {row.getVisibleCells().map((cell, index) => {
                                            const isLastColumn = cell.column.id === "actions";
                                            const isSelectColumn = cell.column.id === "select";

                                            return (
                                                <TableCell
                                                    key={cell.id}
                                                    className={cn(
                                                        isLastColumn && "text-right",
                                                        isSelectColumn && "w-[40px] px-2 text-center"
                                                    )}
                                                >
                                                    {flexRender(
                                                        cell.column.columnDef.cell,
                                                        cell.getContext()
                                                    )}
                                                </TableCell>
                                            );
                                        })}
                                    </TableRow>
                                    {enableRowExpand && renderExpandedRow && expandedRows[row.id] && (
                                        <TableRow className="bg-muted/30">
                                            <TableCell colSpan={columns.length}>
                                                {renderExpandedRow(row.original as TData)}
                                            </TableCell>
                                        </TableRow>
                                    )}
                                </>
                            ))
                        ) : (
                            <TableRow>
                                <TableCell
                                    colSpan={columns.length}
                                    className="h-24 text-center"
                                >
                                    {emptyMessage}
                                </TableCell>
                            </TableRow>
                        )}
                    </TableBody>
                </Table>
            </div>

            {showPagination && <DataTablePagination table={table} />}
        </div>
    );
}