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
import { ServerSidePagination } from "./data-table";

interface DataTablePaginationProps<TData> {
    table: Table<TData>;
    pageSizeOptions?: number[];
    serverSidePagination?: ServerSidePagination;
}

export function DataTablePagination<TData>({
    table,
    pageSizeOptions = [25, 50, 100],
    serverSidePagination,
}: DataTablePaginationProps<TData>) {
    const currentPage = serverSidePagination
        ? serverSidePagination.pageIndex + 1
        : table.getState().pagination.pageIndex + 1;
    const totalPages = serverSidePagination?.pageCount || table.getPageCount();
    const pageSize = serverSidePagination?.pageSize || table.getState().pagination.pageSize;
    const totalRows = serverSidePagination?.totalCount || table.getFilteredRowModel().rows.length;

    const startRow = (serverSidePagination?.pageIndex ?? table.getState().pagination.pageIndex) * pageSize + 1;
    const endRow = Math.min(startRow + pageSize - 1, totalRows);

    const handlePageSizeChange = (value: string) => {
        const newSize = Number(value);
        // For server-side pagination, directly call the callback with new values
        if (serverSidePagination?.onPaginationChange) {
            serverSidePagination.onPaginationChange({
                pageIndex: 0, // Reset to first page when changing size
                pageSize: newSize,
            });
        } else {
            table.setPageSize(newSize);
        }
    };

    const handlePageChange = (newPageIndex: number) => {
        if (serverSidePagination?.onPaginationChange) {
            serverSidePagination.onPaginationChange({
                pageIndex: newPageIndex,
                pageSize,
            });
        } else {
            table.setPageIndex(newPageIndex);
        }
    };

    return (
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            {/* Rows per page selector */}
            <div className="flex items-center gap-2">
                <Label htmlFor="rows-per-page" className="text-sm whitespace-nowrap">
                    Rows per page
                </Label>
                <Select
                    value={pageSize.toString()}
                    onValueChange={handlePageSizeChange}
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
                                onClick={() => handlePageChange(0)}
                                disabled={currentPage <= 1}
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
                                onClick={() => handlePageChange(currentPage - 2)}
                                disabled={currentPage <= 1}
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
                                onClick={() => handlePageChange(currentPage)}
                                disabled={currentPage >= totalPages}
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
                                onClick={() => handlePageChange(totalPages - 1)}
                                disabled={currentPage >= totalPages}
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