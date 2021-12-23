import React, { InputHTMLAttributes } from 'react'
import { Switch as HeadlessSwitch } from '@headlessui/react'
import { FieldInputProps, FieldMetaProps, FieldProps, FormikProps, FormikValues, Field } from 'formik'
import { classNames } from "../../utils";

type SwitchProps<V = any> = {
    label: string
    checked: boolean
    disabled?: boolean
    onChange: (value: boolean) => void
    field?: FieldInputProps<V>
    form?: FormikProps<FormikValues>
    meta?: FieldMetaProps<V>
}

export const Switch: React.FC<SwitchProps> = ({
    label,
    checked: $checked,
    disabled = false,
    onChange: $onChange,
    field,
    form,
}) => {
    const checked = field?.checked ?? $checked

    return (
        <HeadlessSwitch.Group as="div" className="flex items-center space-x-4">
            <HeadlessSwitch.Label>{label}</HeadlessSwitch.Label>
            <HeadlessSwitch
                as="button"
                name={field?.name}
                disabled={disabled}
                checked={checked}
                onChange={value => {
                    form?.setFieldValue(field?.name ?? '', value)
                    $onChange && $onChange(value)
                }}

                className={classNames(
                    checked ? 'bg-teal-500 dark:bg-blue-500' : 'bg-gray-200 dark:bg-gray-600',
                    'ml-4 relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500'
                )}
            >
                {({ checked }) => (
                    <span
                        aria-hidden="true"
                        className={classNames(
                            checked ? 'translate-x-5' : 'translate-x-0',
                            'inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200'
                        )}
                    />
                )}
            </HeadlessSwitch>
        </HeadlessSwitch.Group>
    )
}

export type SwitchFormikProps = SwitchProps & FieldProps & InputHTMLAttributes<HTMLInputElement>

export const SwitchFormik: React.FC<SwitchProps> = args => <Switch {...args} />

interface SwitchGroupProps {
    name: string;
    label?: string;
    description?: string;
    defaultValue?: boolean;
    className?: string;
}

const SwitchGroup: React.FC<SwitchGroupProps> = ({ name, label, description, defaultValue }) => (
    <ul className="mt-2 divide-y divide-gray-200">
        <HeadlessSwitch.Group as="li" className="py-4 flex items-center justify-between">
            {label && <div className="flex flex-col">
                <HeadlessSwitch.Label as="p" className="text-sm font-medium text-gray-900 dark:text-gray-100"
                    passive>
                    {label}
                </HeadlessSwitch.Label>
                {description && (
                    <HeadlessSwitch.Description className="text-sm text-gray-500 dark:text-gray-400">
                        {description}
                    </HeadlessSwitch.Description>
                )}
            </div>
            }

            <Field name={name} type="checkbox">
                {({
                    field,
                    form: { setFieldValue },
                }: any) => (
                    <Switch
                        {...field}
                        type="button"
                        value={field.value}
                        checked={field.checked}
                        onChange={value => {
                            setFieldValue(field?.name ?? '', value)
                        }}
                        className={classNames(
                            field.value ? 'bg-teal-500 dark:bg-blue-500' : 'bg-gray-200',
                            'ml-4 relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500'
                        )}
                    >
                        {/* <span className="sr-only">{label}</span> */}
                        <span
                            aria-hidden="true"
                            className={classNames(
                                field.value ? 'translate-x-5' : 'translate-x-0',
                                'inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200'
                            )}
                        />
                    </Switch>

                )}
            </Field>

            {/* <Field
                name={name}
                defaultValue={defaultValue as any}
                render={({input: {onChange, checked, value}}) => (
                    <Switch
                        value={value}
                        checked={value}
                        onChange={onChange}
                        className={classNames(
                            value ? 'bg-teal-500' : 'bg-gray-200',
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
            /> */}
        </HeadlessSwitch.Group>
    </ul>
)

export { SwitchGroup }