import { Switch } from "@headlessui/react";

interface CheckboxProps {
    value: boolean;
    setValue: (newValue: boolean) => void;
    label: string;
    description?: string;
}

export const Checkbox = ({ label, description, value, setValue }: CheckboxProps) => (
    <Switch.Group as="li" className="py-4 flex items-center justify-between">
        <div className="flex flex-col">
            <Switch.Label as="p" className="text-sm font-medium text-gray-900 dark:text-white" passive>
                {label}
            </Switch.Label>
            {description === undefined ? null : (
                <Switch.Description className="text-sm text-gray-500 dark:text-gray-400">
                    Enable debug mode to get more logs.
                </Switch.Description>
            )}
        </div>
        <Switch
            checked={value}
            onChange={setValue}
            className={
                `${value ? 'bg-teal-500 dark:bg-blue-500' : 'bg-gray-200 dark:bg-gray-700'
            } ml-4 relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500`}
        >
        <span
            className={`${value ? 'translate-x-5' : 'translate-x-0'} inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200`}
        />
        </Switch>
    </Switch.Group>
);