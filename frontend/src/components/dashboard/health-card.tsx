import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

interface HealthCardProps {
    healthy: number;
    unhealthy: number;
    total: number;
}

export function HealthCard({ healthy, unhealthy, total }: HealthCardProps) {
    const healthPercentage = total > 0 ? (healthy / total) * 100 : 0;

    return (
        <Card className="hover:shadow-md transition-shadow">
            <CardHeader>
                <CardTitle className="flex items-center gap-2">
                    Site Health
                    <Badge
                        variant={healthPercentage >= 80 ? "default" : healthPercentage >= 50 ? "secondary" : "destructive"}
                    >
                        {healthPercentage.toFixed(0)}%
                    </Badge>
                </CardTitle>
            </CardHeader>
            <CardContent>
                <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">Healthy</span>
                        <span className="font-medium text-green-600">{healthy}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">Unhealthy</span>
                        <span className="font-medium text-red-600">{unhealthy}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">Total</span>
                        <span className="font-medium">{total}</span>
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}