import type { FieldProps } from "formik";
import { Field } from "formik";
import Select, { components, ControlProps, InputProps, MenuProps, OptionProps } from "react-select";
import { OptionBasicTyped } from "../../domain/constants";
import CreatableSelect from "react-select/creatable";

interface SelectFieldProps<T> {
  name: string;
  label: string;
  help?: string;
  placeholder?: string;
  options: OptionBasicTyped<T>[]
}

export function SelectFieldCreatable<T>({ name, label, help, placeholder, options }: SelectFieldProps<T>) {
  return (
    <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          htmlFor={name}
          className="block text-sm font-medium text-gray-900 dark:text-white sm:pt-2"
        >
          {label}
        </label>
      </div>
      <div className="sm:col-span-2">
        <Field name={name} type="select">
          {({
            field,
            form: { setFieldValue }
          }: FieldProps) => (
            <CreatableSelect
              {...field}
              id={name}
              isClearable={true}
              isSearchable={true}
              components={{
                Input,
                Control,
                Menu,
                Option
              }}
              placeholder={placeholder ?? "Choose an option"}
              styles={{
                singleValue: (base) => ({
                  ...base,
                  color: "unset"
                })
              }}
              theme={(theme) => ({
                ...theme,
                spacing: {
                  ...theme.spacing,
                  controlHeight: 30,
                  baseUnit: 2
                }
              })}
              // value={field?.value ? field.value : options.find(o => o.value == field?.value)}
              value={field?.value ? { value: field.value, label: field.value  } : field.value}
              onChange={(option) => {
                if (option === null) {
                  setFieldValue(field.name, "");
                  return;
                } else {
                  setFieldValue(field.name, option.value ?? "");
                }
              }}
              options={[...[...options, { value: field.value, label: field.value  }].reduce((map, obj) => map.set(obj.value, obj), new Map()).values()]}
            />
          )}
        </Field>
        {help && (
          <p className="mt-2 text-sm text-gray-500" id={`${name}-description`}>{help}</p>
        )}
      </div>
    </div>
  );
}

const Input = (props: InputProps) => {
  return (
    <components.Input
      {...props}
      inputClassName="outline-none border-none shadow-none focus:ring-transparent"
      className="text-gray-400 dark:text-gray-100"
      children={props.children}
    />
  );
};

const Control = (props: ControlProps) => {
  return (
    <components.Control
      {...props}
      className="p-1 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
      children={props.children}
    />
  );
};

const Menu = (props: MenuProps) => {
  return (
    <components.Menu
      {...props}
      className="dark:bg-gray-800 border border-gray-300 dark:border-gray-700 dark:text-gray-400 rounded-md shadow-sm"
      children={props.children}
    />
  );
};

const Option = (props: OptionProps) => {
  return (
    <components.Option
      {...props}
      className="dark:text-gray-400 dark:bg-gray-800 dark:hover:bg-gray-900 dark:focus:bg-gray-900"
      children={props.children}
    />
  );
};

export function SelectField<T>({ name, label, help, placeholder, options }: SelectFieldProps<T>) {
  return (
    <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          htmlFor={name}
          className="block text-sm font-medium text-gray-900 dark:text-white sm:pt-2"
        >
          {label}
        </label>
      </div>
      <div className="sm:col-span-2">
        <Field name={name} type="select">
          {({
            field,
            form: { setFieldValue }
          }: FieldProps) => (
            <Select
              {...field}
              id={name}
              components={{
                Input,
                Control,
                Menu,
                Option
              }}
              placeholder={placeholder ?? "Choose an option"}
              styles={{
                singleValue: (base) => ({
                  ...base,
                  color: "unset"
                })
              }}
              theme={(theme) => ({
                ...theme,
                spacing: {
                  ...theme.spacing,
                  controlHeight: 30,
                  baseUnit: 2
                }
              })}
              // value={field?.value ? field.value : options.find(o => o.value == field?.value)}
              value={field?.value ? { value: field.value, label: field.value  } : field.value}
              onChange={(option) => {
                if (option === null) {
                  setFieldValue(field.name, "");
                  return;
                } else {
                  setFieldValue(field.name, option.value ?? "");
                }
              }}
              options={[...[...options, { value: field.value, label: field.value  }].reduce((map, obj) => map.set(obj.value, obj), new Map()).values()]}
            />
          )}
        </Field>
        {help && (
          <p className="mt-2 text-sm text-gray-500" id={`${name}-description`}>{help}</p>
        )}
      </div>
    </div>
  );
}

export function SelectFieldBasic<T>({ name, label, help, placeholder, options }: SelectFieldProps<T>) {
  return (
    <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
      <div>
        <label
          htmlFor={name}
          className="block text-sm font-medium text-gray-900 dark:text-white sm:pt-2"
        >
          {label}
        </label>
      </div>
      <div className="sm:col-span-2">
        <Field name={name} type="select">
          {({
            field,
            form: { setFieldValue }
          }: FieldProps) => (
            <Select
              {...field}
              id={name}
              components={{
                Input,
                Control,
                Menu,
                Option
              }}
              placeholder={placeholder ?? "Choose an option"}
              styles={{
                singleValue: (base) => ({
                  ...base,
                  color: "unset"
                })
              }}
              theme={(theme) => ({
                ...theme,
                spacing: {
                  ...theme.spacing,
                  controlHeight: 30,
                  baseUnit: 2
                }
              })}
              value={field?.value && options.find(o => o.value == field?.value)}
              onChange={(option) => {
                if (option === null) {
                  setFieldValue(field.name, "");
                  return;
                } else {
                  setFieldValue(field.name, option.value ?? "");
                }
              }}
              options={options}
            />
          )}
        </Field>
        {help && (
          <p className="mt-2 text-sm text-gray-500" id={`${name}-description`}>{help}</p>
        )}
      </div>
    </div>
  );
}
