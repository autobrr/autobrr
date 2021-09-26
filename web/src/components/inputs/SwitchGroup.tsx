import React from "react";
import { Switch } from "@headlessui/react";
import { Field } from "react-final-form";
import { classNames } from "../../styles/utils";

interface Props {
    name: string;
    label: string;
    description?: string;
    defaultValue?: boolean;
    className?: string;
}

const SwitchGroup: React.FC<Props> = ({ name, label, description, defaultValue }) => (
    <ul className="mt-2 divide-y divide-gray-200 dark:divide-gray-700">
        <Switch.Group as="li" className="py-4 flex items-center justify-between">
            <div className="flex flex-col">
                <Switch.Label as="p" className="text-sm font-medium text-gray-900 dark:text-white"
                    passive>
                    {label}
                </Switch.Label>
                {description && (
                    <Switch.Description className="text-sm text-gray-500 dark:text-gray-700">
                        {description}
                    </Switch.Description>
                )}
            </div>

            <Field
                name={name}
                defaultValue={defaultValue as any}
                render={({ input: { onChange, checked, value } }) => (
                    <Switch
                        value={value}
                        checked={value}
                        onChange={onChange}
                        className={classNames(
                            value ? 'bg-teal-500 dark:bg-blue-500' : 'bg-gray-200 dark:bg-gray-500',
                            'ml-4 relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500'
                        )}
                    >
                        <span className="sr-only">Use setting</span>
                        <span
                            aria-hidden="true"
                            className={classNames(
                                value ? 'translate-x-5' : 'translate-x-0',
                                'inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200'
                            )}
                        />
                    </Switch>
                )}
            />
        </Switch.Group>
    </ul>
)

export default SwitchGroup;