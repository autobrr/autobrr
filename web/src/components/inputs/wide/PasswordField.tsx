import { Field } from "react-final-form";
import Error from "../Error";
import { classNames } from "../../../styles/utils";
import { useToggle } from "../../../hooks/hooks";
import { EyeIcon, EyeOffIcon } from "@heroicons/react/solid";

interface Props {
    name: string;
    label?: string;
    placeholder?: string;
    defaultValue?: string;
    help?: string;
    required?: boolean;
}

function PasswordField({ name, label, placeholder, defaultValue, help, required }: Props) {
    const [isVisible, toggleVisibility] = useToggle(false)

    return (
        <div
            className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
            <div>

                <label htmlFor={name} className="block text-sm font-medium text-gray-900 dark:text-white sm:mt-px sm:pt-2">
                    {label} {required && <span className="text-gray-500">*</span>}
                </label>
            </div>
            <div className="sm:col-span-2">
                <Field
                    name={name}
                    defaultValue={defaultValue}
                    render={({ input, meta }) => (
                        <div className="relative">
                            <input
                                {...input}
                                id={name}
                                type={isVisible ? "text" : "password"}
                                className={classNames(meta.touched && meta.error ? "focus:ring-red-500 focus:border-red-500 border-red-500" : "focus:ring-indigo-500 dark:focus:ring-blue-500 focus:border-indigo-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700", "block w-full dark:bg-gray-800 shadow-sm dark:text-gray-100 sm:text-sm rounded-md")}
                                placeholder={placeholder}
                            />
                            <div className="absolute inset-y-0 right-0 px-3 flex items-center" onClick={toggleVisibility}>
                                {!isVisible ? <EyeIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" /> : <EyeOffIcon className="h-5 w-5 text-gray-400 hover:text-gray-500" aria-hidden="true" />}
                            </div>
                        </div>
                    )}
                />
                {help && (
                    <p className="mt-2 text-sm text-gray-500" id="email-description">{help}</p>
                )}
                <Error name={name} classNames="block text-red-500 mt-2" />
            </div>
        </div>
    )
}

export default PasswordField;
