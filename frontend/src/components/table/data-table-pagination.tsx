"use client";

import { Table } from "@tanstack/react-table";
import {
    ChevronLeftIcon,
    ChevronRightIcon,
    ChevronFirstIcon,
    ChevronLastIcon,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import {
    Pagination,
    PaginationContent,
    PaginationItem,
} from "@/components/ui/pagination";

interface DataTablePaginationProps<TData> {
    table: Table<TData>;
    pageSizeOptions?: number[];
}

export function DataTablePagination<TData>({
        table,
        pageSizeOptions = [5, 10, 25, 50, 100],
    }: DataTablePaginationProps<TData>) {
    const currentPage = table.getState().pagination.pageIndex + 1;
    const totalPages = table.getPageCount();
    const pageSize = table.getState().pagination.pageSize;
    const totalRows = table.getFilteredRowModel().rows.length;

    const startRow = table.getState().pagination.pageIndex * pageSize + 1;
    const endRow = Math.min(startRow + pageSize - 1, totalRows);

    return (
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            {/* Rows per page selector */}
            <div className="flex items-center gap-2">
                <Label htmlFor="rows-per-page" className="text-sm whitespace-nowrap">
                    Rows per page
                </Label>
                <Select
                    value={pageSize.toString()}
                    onValueChange={(value) => {
                        table.setPageSize(Number(value));
                    }}
                >
                    <SelectTrigger id="rows-per-page" className="h-9 w-[70px]">
                        <SelectValue placeholder={pageSize} />
                    </SelectTrigger>
                    <SelectContent side="top">
                        {pageSizeOptions.map((size) => (
                            <SelectItem key={size} value={size.toString()}>
                                {size}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            </div>

            {/* Page info and navigation */}
            <div className="flex items-center gap-6">
                {/* Row count info */}
                <div className="text-sm text-muted-foreground whitespace-nowrap">
          <span className="font-medium text-foreground">
            {startRow}-{endRow}
          </span>{" "}
                    of{" "}
                    <span className="font-medium text-foreground">{totalRows}</span>
                </div>

                {/* Pagination buttons */}
                <Pagination>
                    <PaginationContent>
                        {/* First page */}
                        <PaginationItem>
                            <Button
                                variant="outline"
                                size="icon"
                                className="h-9 w-9"
                                onClick={() => table.setPageIndex(0)}
                                disabled={!table.getCanPreviousPage()}
                                aria-label="Go to first page"
                            >
                                <ChevronFirstIcon className="h-4 w-4" />
                            </Button>
                        </PaginationItem>

                        {/* Previous page */}
                        <PaginationItem>
                            <Button
                                variant="outline"
                                size="icon"
                                className="h-9 w-9"
                                onClick={() => table.previousPage()}
                                disabled={!table.getCanPreviousPage()}
                                aria-label="Go to previous page"
                            >
                                <ChevronLeftIcon className="h-4 w-4" />
                            </Button>
                        </PaginationItem>

                        {/* Page indicator */}
                        <div className="flex items-center gap-1 px-2 text-sm">
                            <span className="font-medium">{currentPage}</span>
                            <span className="text-muted-foreground">of</span>
                            <span className="font-medium">{totalPages}</span>
                        </div>

                        {/* Next page */}
                        <PaginationItem>
                            <Button
                                variant="outline"
                                size="icon"
                                className="h-9 w-9"
                                onClick={() => table.nextPage()}
                                disabled={!table.getCanNextPage()}
                                aria-label="Go to next page"
                            >
                                <ChevronRightIcon className="h-4 w-4" />
                            </Button>
                        </PaginationItem>

                        {/* Last page */}
                        <PaginationItem>
                            <Button
                                variant="outline"
                                size="icon"
                                className="h-9 w-9"
                                onClick={() => table.setPageIndex(table.getPageCount() - 1)}
                                disabled={!table.getCanNextPage()}
                                aria-label="Go to last page"
                            >
                                <ChevronLastIcon className="h-4 w-4" />
                            </Button>
                        </PaginationItem>
                    </PaginationContent>
                </Pagination>
            </div>
        </div>
    );
}