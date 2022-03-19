import { Field } from "formik";
import { classNames } from "../../utils";
import { EyeIcon, EyeOffIcon } from "@heroicons/react/solid";
import { useToggle } from "../../hooks/hooks";

type COL_WIDTHS = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;

interface TextFieldProps {
    name: string;
    defaultValue?: string;
    label?: string;
    placeholder?: string;
    columns?: COL_WIDTHS;
    autoComplete?: string;
    hidden?: boolean;
}

export const TextField = ({
    name,
    defaultValue,
    label,
    placeholder,
    columns,
    autoComplete,
    hidden,
}: TextFieldProps) => (
    <div
        className={classNames(
            hidden ? "hidden" : "",
            columns ? `col-span-${columns}` : "col-span-12",
        )}
    >
        {label && (
            <label htmlFor={name} className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
                {label}
            </label>
        )}
        <Field name={name}>
            {({
                field,
                meta,
            }: any) => (
                <div>
                    <input
                        {...field}
                        id={name}
                        type="text"
                        defaultValue={defaultValue}
                        autoComplete={autoComplete}
                        className="mt-2 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100"
                        placeholder={placeholder}
                    />

                    {meta.touched && meta.error && (
                        <div className="error">{meta.error}</div>
                    )}
                </div>
            )}
        </Field>
    </div>
)

interface PasswordFieldProps {
    name: string;
    label?: string;
    placeholder?: string;
    columns?: COL_WIDTHS;
    autoComplete?: string;
    defaultValue?: string;
    help?: string;
    required?: boolean;
}

export const PasswordField = ({
    name,
    label,
    placeholder,
    defaultValue,
    columns,
    autoComplete,
    help,
    required
}: PasswordFieldProps) => {
    const [isVisible, toggleVisibility] = useToggle(false)

    return (
        <div
            className={classNames(
                columns ? `col-span-${columns}` : "col-span-12"
            )}
        >
            {label && (
                <label htmlFor={name} className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
                    {label} {required && <span className="text-gray-500">*</span>}
                </label>
            )}
            <Field name={name} defaultValue={defaultValue}>
                {({
                    field,
                    meta,
                }: any) => (
                    <div className="sm:col-span-2 relative">
                        <input
                            {...field}
                            id={name}
                            type={isVisible ? "text" : "password"}
                            autoComplete={autoComplete}
                            className={classNames(meta.touched && meta.error ? "focus:ring-red-500 focus:border-red-500 border-red-500" : "focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700", "mt-2 block w-full dark:bg-gray-800 dark:text-gray-100 rounded-md")}
                            placeholder={placeholder}
                        />

                        <div className="absolute inset-y-0 right-0 px-3 flex items-center" onClick={toggleVisibility}>
                            {!isVisible ? <EyeIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" /> : <EyeOffIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" />}
                        </div>

                        {help && (
                            <p className="mt-2 text-sm text-gray-500" id="email-description">{help}</p>
                        )}

                        {meta.touched && meta.error && (
                            <div className="error">{meta.error}</div>
                        )}
                    </div>
                )}
            </Field>
        </div>
    )
}

interface NumberFieldProps {
    name: string;
    label?: string;
    placeholder?: string;
}

export const NumberField = ({
    name,
    label,
    placeholder
}: NumberFieldProps) => (
    <div className="col-span-12 sm:col-span-6">
        <label htmlFor={name} className="block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
            {label}
        </label>

        <Field name={name} type="number">
            {({
                field,
                meta,
            }: any) => (
                <div className="sm:col-span-2">
                    <input
                        type="number"
                        {...field}
                        className={classNames(
                            meta.touched && meta.error
                                ? "focus:ring-red-500 focus:border-red-500 border-red-500"
                                : "focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 border-gray-300",
                            "mt-2 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 dark:text-gray-100 rounded-md"
                        )}
                        placeholder={placeholder}
                    />
                    {meta.touched && meta.error && (
                        <div className="error">{meta.error}</div>
                    )}
                </div>

            )}
        </Field>
    </div>
);
