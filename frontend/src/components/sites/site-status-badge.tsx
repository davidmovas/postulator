import { Badge } from "@/components/ui/badge";

const SiteStatusBadge = ({ status }: { status: string }) => {
    const getStatusConfig = (status: string) => {
        switch (status) {
            case "active":
                return { variant: "default" as const, label: "Active" };
            case "inactive":
                return { variant: "secondary" as const, label: "Inactive" };
            case "error":
                return { variant: "destructive" as const, label: "Error" };
            default:
                return { variant: "outline" as const, label: status };
        }
    };

    const config = getStatusConfig(status);

    return <Badge variant={config.variant}>{config.label}</Badge>;
};

export default SiteStatusBadge;