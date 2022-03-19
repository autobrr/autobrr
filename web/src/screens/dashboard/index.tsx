import { Stats } from "./Stats";
import { ActivityTable } from "./ActivityTable";

export const Dashboard = () => (
    <main className="py-10">
        <div className="px-4 pb-8 mx-auto max-w-7xl sm:px-6 lg:px-8">
            <Stats />
            <ActivityTable />
        </div>
    </main>
);