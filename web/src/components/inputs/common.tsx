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
}

const CheckboxField = ({
  name,
  label,
  sublabel,
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
      <label htmlFor={name} className="font-medium text-gray-900 dark:text-gray-100">
        {label}
      </label>
      <p className="text-gray-500">{sublabel}</p>
    </div>
  </div>
);

interface CheckboxFieldIconProps {
  name: string;
  label: string;
  sublabel?: string;
  disabled?: boolean;
}

const CheckboxFieldIcon = ({
  name,
  label,
  sublabel,
  disabled
}: CheckboxFieldIconProps) => (
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
      <label htmlFor={name} className="flex font-medium text-gray-900 dark:text-gray-100">
        {label}
        <svg className="float-right ml-1 h-5 w-5 text-gray-500" width="800px" height="800px" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
          <path fill="#333" d="M512 64C264.6 64 64 264.6 64 512s200.6 448 448 448 448-200.6 448-448S759.4 64 512 64zm0 820c-205.4 0-372-166.6-372-372s166.6-372 372-372 372 166.6 372 372-166.6 372-372 372z"/>
          <path fill="#E6E6E6" d="M512 140c-205.4 0-372 166.6-372 372s166.6 372 372 372 372-166.6 372-372-166.6-372-372-372zm32 588c0 4.4-3.6 8-8 8h-48c-4.4 0-8-3.6-8-8V456c0-4.4 3.6-8 8-8h48c4.4 0 8 3.6 8 8v272zm-32-344a48.01 48.01 0 0 1 0-96 48.01 48.01 0 0 1 0 96z"/>
          <path fill="#333" d="M464 336a48 48 0 1 0 96 0 48 48 0 1 0-96 0zm72 112h-48c-4.4 0-8 3.6-8 8v272c0 4.4 3.6 8 8 8h48c4.4 0 8-3.6 8-8V456c0-4.4-3.6-8-8-8z"/>
        </svg>
      </label>
      <p className="text-gray-500">{sublabel}</p>
    </div>
  </div>
);


export { ErrorField, CheckboxField, CheckboxFieldIcon };