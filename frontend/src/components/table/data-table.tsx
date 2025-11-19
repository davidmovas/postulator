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

export interface ServerSidePagination {
    pageIndex: number;
    pageSize: number;
    pageCount: number;
    totalCount?: number;
    onPaginationChange?: (pagination: PaginationState) => void;
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
    serverSidePagination?: ServerSidePagination;
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
    serverSidePagination,
    enableRowExpand = false,
    expandOnRowClick = false,
    renderExpandedRow,
}: DataTableProps<TData, TValue>) {
    const [sorting, setSorting] = useState<SortingState>(defaultSorting);
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
    const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
    const [rowSelection, setRowSelection] = useState<RowSelectionState>({});
    const [expandedRows, setExpandedRows] = useState<Record<string, boolean>>({});
    const [keptAliveRows, setKeptAliveRows] = useState<Record<string, boolean>>({});

    // Для серверной пагинации используем переданное состояние, иначе локальное
    const [internalPagination, setInternalPagination] = useState<PaginationState>({
        pageIndex: 0,
        pageSize: defaultPageSize,
    });

    const pagination = serverSidePagination
        ? {
            pageIndex: serverSidePagination.pageIndex,
            pageSize: serverSidePagination.pageSize
        }
        : internalPagination;

    const handlePaginationChange = serverSidePagination?.onPaginationChange
        ? (updater: any) => {
            const newPagination = typeof updater === 'function'
                ? updater(pagination)
                : updater;
            serverSidePagination.onPaginationChange?.(newPagination);
        }
        : setInternalPagination;

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
        getPaginationRowModel: serverSidePagination ? undefined : getPaginationRowModel(),
        onPaginationChange: handlePaginationChange,
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
        manualPagination: !!serverSidePagination,
        pageCount: serverSidePagination?.pageCount,
        state: {
            sorting,
            columnFilters,
            columnVisibility,
            rowSelection,
            pagination,
        },
    });

    // Helper: не разворачивать строку при клике по интерактивным элементам
    const isInteractiveClick = (target: EventTarget | null, container: HTMLElement | null) => {
        if (!(target instanceof HTMLElement) || !container) return false;
        const selectors = [
            'button',
            'a',
            'input',
            'select',
            'textarea',
            '[role="button"]',
            '[role="menuitem"]',
            '[role="menu"]',
            '[data-prevent-row-toggle]'
        ].join(',');
        let el: HTMLElement | null = target;
        while (el && el !== container) {
            if (el.matches(selectors) || el.isContentEditable) return true;
            el = el.parentElement;
        }
        return false;
    };

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
                                <React.Fragment key={row.id}>
                                    <TableRow
                                        data-state={row.getIsSelected() && "selected"}
                                        className={cn(enableRowExpand && expandOnRowClick && "cursor-pointer")}
                                        onClick={(e) => {
                                            if (!(enableRowExpand && expandOnRowClick && renderExpandedRow)) return;
                                            if (isInteractiveClick(e.target, e.currentTarget)) return;
                                            setExpandedRows((prev) => {
                                                const isOpen = !!prev[row.id];
                                                // Аккордеон: в один момент времени открыта только одна строка
                                                const next = isOpen ? {} : { [row.id]: true };
                                                if (!isOpen) {
                                                    setKeptAliveRows((k) => ({ ...k, [row.id]: true }));
                                                }
                                                return next;
                                            });
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
                                    {enableRowExpand && renderExpandedRow && (keptAliveRows[row.id] || expandedRows[row.id]) && (
                                        <TableRow className={cn(expandedRows[row.id] ? "bg-muted/30" : "bg-transparent") }>
                                            <TableCell
                                                colSpan={columns.length}
                                                className={cn(!expandedRows[row.id] && "p-0")}
                                            >
                                                <div
                                                    className={cn(
                                                        "transition-all duration-200 ease-out overflow-hidden",
                                                        expandedRows[row.id]
                                                            ? "opacity-100 max-h-[1000px]"
                                                            : "opacity-0 max-h-0"
                                                    )}
                                                    aria-hidden={!expandedRows[row.id]}
                                                >
                                                    {renderExpandedRow(row.original as TData)}
                                                </div>
                                            </TableCell>
                                        </TableRow>
                                    )}
                                </React.Fragment>
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

            {showPagination && (
                <DataTablePagination
                    table={table}
                    serverSidePagination={serverSidePagination}
                />
            )}
        </div>
    );
}