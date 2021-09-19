import { Switch as HeadlessSwitch } from '@headlessui/react'
import { FieldInputProps, FieldMetaProps, FieldProps, FormikProps, FormikValues } from 'formik'
import React, { InputHTMLAttributes } from 'react'
import { classNames } from "../../../styles/utils";

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
                    checked ? 'bg-teal-500' : 'bg-gray-200',
                    'ml-4 relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-light-blue-500'
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