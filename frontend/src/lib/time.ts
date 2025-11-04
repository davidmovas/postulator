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

export { formatDateTime };