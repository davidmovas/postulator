export default function Loading() {
    return (
        <div className="fixed inset-0 z-40 grid place-items-center bg-[color:var(--background)]/80 backdrop-blur-sm">
            <div className="flex flex-col items-center gap-3">
                <div className="size-10 rounded-full border-2 border-[var(--foreground)]/20 border-t-[var(--foreground)] animate-spin" aria-hidden="true" />
                <span className="text-sm text-[color:var(--foreground)]/80">Loading…</span>
            </div>
        </div>
    );
}
