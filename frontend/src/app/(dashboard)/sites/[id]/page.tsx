export default async function SitePage({
    params
} : {
    params: Promise<{ id: string }>
}) {
    return (
        <div className="p-6">
            <h1 className="text-2xl font-bold mb-4">Site</h1>
            <p>Manage your site here.</p>
        </div>
    );
}