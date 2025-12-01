import { formatDistanceToNow } from "date-fns";

const WEEK_IN_MS = 7 * 24 * 60 * 60 * 1000;

const formatDateTime = (dateString: string | null | undefined): string | null => {
    if (!dateString) return null;

    try {
        return new Date(dateString).toLocaleDateString('de-DE', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            hour12: false
        }).replace(/, /, ' ');
    } catch {
        return null;
    }
};

/**
 * Smart date formatting: relative time if < 7 days, full date otherwise
 * Format: DD-MM-YYYY HH:MM (24h)
 */
const formatSmartDate = (dateString: string | null | undefined): string => {
    if (!dateString) return '-';

    try {
        const date = new Date(dateString);
        const now = new Date();
        const diff = now.getTime() - date.getTime();

        if (diff < WEEK_IN_MS) {
            return formatDistanceToNow(date, { addSuffix: true });
        }

        // Format: DD-MM-YYYY HH:MM
        const day = String(date.getDate()).padStart(2, '0');
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const year = date.getFullYear();
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');

        return `${day}-${month}-${year} ${hours}:${minutes}`;
    } catch {
        return '-';
    }
};

const toGoDateFormat = (date: Date): string => {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    const seconds = String(date.getSeconds()).padStart(2, '0');

    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
};

export { formatDateTime, formatSmartDate, toGoDateFormat };