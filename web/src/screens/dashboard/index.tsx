import { Stats } from "./Stats";
import { ActivityTable } from "./ActivityTable";

export const Dashboard = () => (
  <main className="py-10">
    <div className="max-w-screen-xl mx-auto pb-6 px-4 sm:px-6 lg:pb-16 lg:px-8">
      <Stats />
      <ActivityTable />
    </div>
  </main>
);