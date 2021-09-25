interface props {
    text: string;
    buttonText?: string;
    buttonOnClick?: any;
}

export function EmptyListState({ text, buttonText, buttonOnClick }: props) {
    return (
        <div className="px-4 py-12 flex flex-col items-center">
            <p className="text-center text-gray-500 dark:text-white">{text}</p>
            {buttonText && buttonOnClick && (
                <button
                    type="button"
                    className="relative inline-flex items-center px-4 py-2 mt-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                    onClick={buttonOnClick}
                >
                    {buttonText}
                </button>
            )}
        </div>
    )
}