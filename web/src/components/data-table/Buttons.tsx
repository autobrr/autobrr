import { classNames } from "../../utils"

interface ButtonProps {
    className?: string;
    children: any;
    [rest: string]: any;
}

export const Button = ({ children, className, ...rest }: ButtonProps) => (
    <button
        type="button"
        className={classNames(
            className ?? "",
            "relative inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-800 text-sm font-medium rounded-md text-gray-700 dark:text-gray-500 bg-white dark:bg-gray-800 hover:bg-gray-50"
        )}
        {...rest}
    >
        {children}
    </button>
);


export const PageButton = ({ children, className, ...rest }: ButtonProps) => (
    <button
        type="button"
        className={classNames(
            className ?? "",
            "relative inline-flex items-center px-2 py-2 border border-gray-300 dark:border-gray-700 text-sm font-medium text-gray-500 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-600"
        )}
        {...rest}
    >
        {children}
    </button>
);