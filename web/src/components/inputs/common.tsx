import { Field, FieldProps } from "formik";
import { classNames } from "../../utils";

interface ErrorFieldProps {
    name: string;
    classNames?: string;
}

const ErrorField = ({ name, classNames }: ErrorFieldProps) => (
  <div>
    <Field name={name} subscribe={{ touched: true, error: true }}>
      {({ meta: { touched, error } }: FieldProps) =>
        touched && error ? <span className={classNames}>{error}</span> : null
      }
    </Field>
  </div>
);

interface CheckboxFieldProps {
    name: string;
    label: string;
    sublabel?: string;
    disabled?: boolean;
    tooltip?: JSX.Element;
}

const  CheckboxField = ({
  name,
  label,
  sublabel,
  tooltip,
  disabled
}: CheckboxFieldProps) => (
  <div className="relative flex items-start">
    <div className="flex items-center h-5">
      <Field
        id={name}
        name={name}
        type="checkbox" 
        className={classNames(
          "focus:ring-blue-500 h-4 w-4 text-blue-600 border-gray-300 rounded", 
          disabled ? "bg-gray-200 dark:bg-gray-700 dark:border-gray-700" : ""
        )}
        disabled={disabled}
      />
    </div>
    <div className="ml-3 text-sm">
      <label htmlFor={name} className="flex mb-2 block text-xs font-bold text-gray-700 dark:text-gray-200 uppercase tracking-wide">
        <div id={name} className="flex">
          {label}
          {tooltip}
        </div>
      </label>
      <p className="text-gray-500">{sublabel}</p>
    </div>
  </div>
);

export { ErrorField, CheckboxField };