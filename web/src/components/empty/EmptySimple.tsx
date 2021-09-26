import { PlusIcon } from "@heroicons/react/solid";

interface props {
    title: string;
    subtitle: string;
    buttonText: string;
    buttonAction: any;
}

const EmptySimple = ({ title, subtitle, buttonText, buttonAction }: props) => (
    <div className="text-center py-8">
        <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">{title}</h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-200">{subtitle}</p>
        <div className="mt-6">
            <button
                type="button"
                onClick={buttonAction}
                className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
            >
                <PlusIcon className="-ml-1 mr-2 h-5 w-5" aria-hidden="true" />
                {buttonText}
            </button>
        </div>
    </div>
)

export default EmptySimple;