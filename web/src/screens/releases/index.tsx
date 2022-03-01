import { ReleaseTable } from "./ReleaseTable";

export const Releases = () => (
    <main>
        <header className="py-10">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex justify-between">
                <h1 className="text-3xl font-bold text-black dark:text-white capitalize">Releases</h1>
            </div>
        </header>
        <div className="px-4 pb-8 mx-auto max-w-7xl sm:px-6 lg:px-8">
            <ReleaseTable />
        </div>
    </main>
);