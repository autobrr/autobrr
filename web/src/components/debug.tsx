import { FC } from "react";

interface DebugProps {
    values: unknown;
}

const DEBUG: FC<DebugProps> = ({ values }) => {
  if (process.env.NODE_ENV !== "development") {
    return null;
  }

  return (
    <div className="w-full p-2 flex flex-col mt-6 bg-gray-100 dark:bg-gray-900">
      <pre className="dark:text-gray-400 break-all whitespace-pre-wrap">{JSON.stringify(values, null, 2)}</pre>
    </div>
  );
};

export default DEBUG;
