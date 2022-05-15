import { useQuery } from "react-query";
import { APIClient } from "../../api/APIClient";

interface StatsItemProps {
    name: string;
    value?: number;
}

const StatsItem = ({ name, value }: StatsItemProps) => (
  <div
      className="relative px-4 py-5 overflow-hidden bg-white rounded-lg shadow-lg dark:bg-gray-800"
      title="All time"
  >
      <dt>
          <p className="pb-1 text-sm font-medium text-gray-500 truncate">{name}</p>
      </dt>

      <dd className="flex items-baseline">
          <p className="text-3xl font-extrabold text-gray-900 dark:text-gray-200">{value}</p>
      </dd>
  </div>
);

export const Stats = () => {
  const { isLoading, data } = useQuery(
      "dash_release_stats",
      () => APIClient.release.stats(),
      { refetchOnWindowFocus: false }
  );

  if (isLoading)
      return null;

  return (
      <div>
          <h3 className="text-2xl font-medium leading-6 text-gray-900 dark:text-gray-200">
              Stats
          </h3>

          <dl className="grid grid-cols-1 gap-5 mt-5 sm:grid-cols-2 lg:grid-cols-3">
              <StatsItem name="Filtered Releases" value={data?.filtered_count} />
              {/* <StatsItem name="Filter Rejected Releases" stat={data?.filter_rejected_count} /> */}
              <StatsItem name="Rejected Pushes" value={data?.push_rejected_count} />
              <StatsItem name="Approved Pushes" value={data?.push_approved_count} />
          </dl>
      </div>
  );
};