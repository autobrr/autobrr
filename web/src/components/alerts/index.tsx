import { ExclamationTriangleIcon } from "@heroicons/react/24/outline";

interface props {
  title?: string;
  text: string;
}

export function AlertWarning({ title, text }: props) {
  return (
    <div className="my-4 rounded-md bg-yellow-50 dark:bg-yellow-100 p-4 border border-yellow-300 dark:border-none">
      <div className="flex">
        <div className="flex-shrink-0">
          <ExclamationTriangleIcon
            className="h-5 w-5 text-yellow-400 dark:text-yellow-600"
            aria-hidden="true"
          />
        </div>
        <div className="ml-3">
          {title ? (
            <h3 className="mb-1 text-md font-medium text-yellow-800">{title}</h3>
          ) : null}
          <div className="text-sm text-yellow-800">
            <p>{text}</p>
          </div>
        </div>
      </div>
    </div>
  );
}

export { ErrorPage } from "./ErrorPage";