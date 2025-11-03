import { DataTableFilterConfig } from "@/components/table/data-table";

export const statusFilter: DataTableFilterConfig = {
    columnId: "status",
    title: "Status",
    options: [
        { label: "Active", value: "active" },
        { label: "Inactive", value: "inactive" },
        { label: "Error", value: "error" },
    ],
};

export const healthFilter: DataTableFilterConfig = {
    columnId: "healthStatus",
    title: "Health",
    options: [
        { label: "Healthy", value: "healthy" },
        { label: "Unhealthy", value: "unhealthy" },
        { label: "Unknown", value: "unknown" },
    ],
};

export const tableFilters = [statusFilter, healthFilter];