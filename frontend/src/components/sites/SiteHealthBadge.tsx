import { Badge } from "@/components/ui/badge";

const HealthIndicator = ({ status }: { status: string }) => {
    const getHealthConfig = (health: string) => {
        switch (health) {
            case "healthy":
                return { variant: "success" as const, label: "Healthy" };
            case "unhealthy":
                return { variant: "destructive" as const, label: "Unhealthy" };
            case "checking":
                return { variant: "secondary" as const, label: "Checking...", className: "animate-pulse" };
            case "unknown":
                return { variant: "secondary" as const, label: "Unknown" };
            default:
                return { variant: "outline" as const, label: "Unknown" };
        }
    };

    const config = getHealthConfig(status);

    return (
        <Badge variant={config.variant} className={config.className}>
            {config.label}
        </Badge>
    );
};

export default HealthIndicator;