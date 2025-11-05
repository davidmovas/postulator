import { ColumnDef } from "@tanstack/react-table";
import { SiteStats } from "@/models/stats";
import { format } from "date-fns";

export const statsColumns: ColumnDef<SiteStats>[] = [
    {
        accessorKey: "date",
        header: "Date",
        cell: ({ row }) => {
            const date = row.getValue("date") as string;
            return format(new Date(date), 'MMM dd, yyyy');
        },
    },
    {
        accessorKey: "articlesPublished",
        header: "Published",
        cell: ({ row }) => {
            const value = row.getValue("articlesPublished") as number;
            return <span className="text-green-600 font-medium">{value}</span>;
        },
    },
    {
        accessorKey: "articlesFailed",
        header: "Failed",
        cell: ({ row }) => {
            const value = row.getValue("articlesFailed") as number;
            return <span className="text-red-600 font-medium">{value}</span>;
        },
    },
    {
        accessorKey: "totalWords",
        header: "Words",
        cell: ({ row }) => {
            const value = row.getValue("totalWords") as number;
            return value.toLocaleString();
        },
    },
    {
        accessorKey: "internalLinksCreated",
        header: "Internal Links",
        cell: ({ row }) => {
            return row.getValue("internalLinksCreated") as number;
        },
    },
    {
        accessorKey: "externalLinksCreated",
        header: "External Links",
        cell: ({ row }) => {
            return row.getValue("externalLinksCreated") as number;
        },
    },
];