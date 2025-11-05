import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { NumberTicker } from "@/components/ui/number-ticker";
import { cn } from "@/lib/utils";

interface StatsCardProps {
    title: string;
    value: number;
    description?: string;
    icon?: React.ReactNode;
    className?: string;
}

export function StatsCard({
    title,
    value,
    description,
    icon,
    className
}: StatsCardProps) {
    return (
        <Card className={cn(
            "hover:shadow-md transition-all duration-200 border-l-4",
            "group hover:border-l-white/90",
            "hover:translate-y-[-2px]",
            className
        )}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                    {title}
                </CardTitle>
                {icon && (
                    <div className="h-4 w-4 text-muted-foreground group-hover:text-foreground/60 transition-colors">
                        {icon}
                    </div>
                )}
            </CardHeader>
            <CardContent className="space-y-2">
                <div className="text-3xl font-bold">
                    <NumberTicker value={value} />
                </div>
                {description && (
                    <p className="text-xs text-muted-foreground">
                        {description}
                    </p>
                )}
            </CardContent>
        </Card>
    );
}