import { Button } from "@/components/ui/button";
import { CalendarIcon } from "lucide-react";
import {
    Button as AriaButton,
    DateRangePicker,
    Dialog,
    Group,
    Popover,
} from "react-aria-components";
import { cn } from "@/lib/utils";
import { RangeCalendar } from "@/components/ui/calendar-rac";
import { DateInput, dateInputStyle } from "@/components/ui/datefield-rac";
import { parseDate, CalendarDate } from "@internationalized/date";

interface StatsFiltersProps {
    dateRange: { from: Date; to: Date } | undefined;
    onDateRangeChange: (range: { from: Date; to: Date } | undefined) => void;
}

export function StatsFilters({ dateRange, onDateRangeChange }: StatsFiltersProps) {
    const quickRanges = [
        { label: "7D", days: 7 },
        { label: "30D", days: 30 },
        { label: "90D", days: 90 }
    ];

    // Конвертация Date в CalendarDate для React Aria
    const dateToCalendarDate = (date: Date): CalendarDate => {
        return parseDate(date.toISOString().split('T')[0]);
    };

    // Конвертация CalendarDate в Date
    const calendarDateToDate = (calendarDate: CalendarDate): Date => {
        return new Date(calendarDate.year, calendarDate.month - 1, calendarDate.day);
    };

    // Текущее значение в формате React Aria
    const ariaValue = dateRange ? {
        start: dateToCalendarDate(dateRange.from),
        end: dateToCalendarDate(dateRange.to)
    } : undefined;

    return (
        <div className="flex flex-wrap items-center justify-end gap-4 w-full">
            {/* Быстрый выбор периода */}
            <div className="flex gap-2">
                {quickRanges.map((range) => (
                    <Button
                        key={range.label}
                        variant="outline"
                        size="sm"
                        onClick={() => {
                            const to = new Date();
                            const from = new Date();
                            from.setDate(to.getDate() - range.days);

                            onDateRangeChange({
                                from,
                                to
                            });
                        }}
                    >
                        {range.label}
                    </Button>
                ))}
            </div>

            {/* Кастомный выбор дат с React Aria */}
            <DateRangePicker
                value={ariaValue}
                onChange={(value) => {
                    if (value) {
                        onDateRangeChange({
                            from: calendarDateToDate(value.start),
                            to: calendarDateToDate(value.end)
                        });
                    }
                }}
            >
                <div className="flex">
                    <Group className={cn(dateInputStyle, "pe-9 w-[280px]")}>
                        <DateInput slot="start" unstyled />
                        <span aria-hidden="true" className="px-2 text-muted-foreground/70">
                            -
                        </span>
                        <DateInput slot="end" unstyled />
                    </Group>
                    <AriaButton className="z-10 -ms-9 -me-px flex w-9 items-center justify-center rounded-e-md text-muted-foreground/80 transition-[color,box-shadow] outline-none hover:text-foreground data-focus-visible:border-ring data-focus-visible:ring-[3px] data-focus-visible:ring-ring/50">
                        <CalendarIcon size={16} />
                    </AriaButton>
                </div>
                <Popover
                    className="z-50 rounded-md border bg-background text-popover-foreground shadow-lg outline-hidden data-entering:animate-in data-exiting:animate-out data-[entering]:fade-in-0 data-[entering]:zoom-in-95 data-[exiting]:fade-out-0 data-[exiting]:zoom-out-95"
                    offset={4}
                >
                    <Dialog className="max-h-[inherit] overflow-auto p-2">
                        <RangeCalendar />
                    </Dialog>
                </Popover>
            </DateRangePicker>
        </div>
    );
}