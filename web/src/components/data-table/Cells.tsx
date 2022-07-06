import * as React from "react";
import { formatDistanceToNowStrict } from "date-fns";
import { CheckIcon } from "@heroicons/react/solid";
import { ClockIcon, BanIcon, ExclamationCircleIcon } from "@heroicons/react/outline";

import { classNames, simplifyDate } from "../../utils";
import { Tooltip } from "../tooltips/Tooltip";

interface CellProps {
    value: string;
}

export const AgeCell = ({ value }: CellProps) => (
  <div className="text-sm text-gray-500" title={value}>
    {formatDistanceToNowStrict(new Date(value), { addSuffix: true })}
  </div>
);

export const TitleCell = ({ value }: CellProps) => (
  <div
    className="text-sm font-medium box-content text-gray-900 dark:text-gray-300 max-w-[128px] sm:max-w-[256px] md:max-w-[360px] lg:max-w-[640px] xl:max-w-[840px] overflow-auto py-4"
    title={value}
  >
    {value}
  </div>
);

interface ReleaseStatusCellProps {
    value: ReleaseActionStatus[];
}

interface StatusCellMapEntry {
    colors: string;
    icon: React.ReactElement;
}

const StatusCellMap: Record<string, StatusCellMapEntry> = {
  "PUSH_ERROR": {
    colors: "bg-pink-100 text-pink-800 hover:bg-pink-300",
    icon: <ExclamationCircleIcon className="h-5 w-5" aria-hidden="true" />
  },
  "PUSH_REJECTED": {
    colors: "bg-blue-200 dark:bg-blue-100 text-blue-400 dark:text-blue-800 hover:bg-blue-300 dark:hover:bg-blue-400",
    icon: <BanIcon className="h-5 w-5" aria-hidden="true" />
  },
  "PUSH_APPROVED": {
    colors: "bg-green-100 text-green-800 hover:bg-green-300",
    icon: <CheckIcon className="h-5 w-5" aria-hidden="true" />
  },
  "PENDING": {
    colors: "bg-yellow-100 text-yellow-800 hover:bg-yellow-200",
    icon: <ClockIcon className="h-5 w-5" aria-hidden="true" />
  }
};

// const GetReleaseStatusString = (releaseAction: ReleaseActionStatus) => {
//   const items: Array<string> = [
//     `action: ${releaseAction.action}`,
//     `type: ${releaseAction.type}`,
//     `status: ${releaseAction.status}`,
//     `time: ${simplifyDate(releaseAction.timestamp)}`
//   ];
//   if (releaseAction.client != "")
//     items.push(`client: ${releaseAction.client}`);
//   if (releaseAction.filter != "")
//     items.push(`filter: ${releaseAction.filter}`);
//   if (releaseAction.rejections.length)
//     items.push(`rejections: ${releaseAction.rejections}`);
//   return items.join(" | ");
// };

export const ReleaseStatusCell = ({ value }: ReleaseStatusCellProps) => (
  <div className="flex text-sm font-medium text-gray-900 dark:text-gray-300">
    {value.map((v, idx) => (
      <div
        key={idx}
        className={classNames(
          StatusCellMap[v.status].colors,
          "mr-1 inline-flex items-center rounded text-xs font-semibold cursor-pointer"
        )}
      >
        <Tooltip button={StatusCellMap[v.status].icon}>
          <ol className="flex flex-col">
            <li className="py-1">Status: {v.status}</li>
            <li className="py-1">Action: {v.action}</li>
            <li className="py-1">Type: {v.type}</li>
            {v.client && <li className="py-1">Client: {v.client}</li>}
            {v.filter && <li className="py-1">Filter: {v.filter}</li>}
            <li className="py-1">Time: {simplifyDate(v.timestamp)}</li>
          </ol>
        </Tooltip>
      </div>
    ))}
  </div>
);
