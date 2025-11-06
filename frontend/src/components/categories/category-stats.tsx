import { Card, CardContent } from "@/components/ui/card";
import { RiBarChart2Line } from "@remixicon/react";

interface CategoryStatsProps {
    categoryId: number;
}

export function CategoryStats({ categoryId }: CategoryStatsProps) {
    // TODO: Implement category statistics
    return (
        <Card>
            <CardContent className="pt-6">
                <div className="text-center py-8">
                    <RiBarChart2Line className="w-12 h-12 text-muted-foreground/50 mx-auto mb-4" />
                    <h3 className="text-lg font-medium mb-2">Category Statistics</h3>
                    <p className="text-muted-foreground text-sm">
                        Statistics for this category will appear here.
                    </p>
                </div>
            </CardContent>
        </Card>
    );
}