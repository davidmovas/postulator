export default async function SiteCategoriesPage({
    params
} : {
    params: Promise<{ id: string }>
}) {
    const { id } = await params;
    const siteId = parseInt(id);

    return (
        <div className="p-6">
            <h1 className="text-2xl font-bold mb-4">Categories</h1>
            <p>Manage your site categories here.</p>
        </div>
    );
}