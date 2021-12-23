import React from "react";
import { Field } from "formik";

interface ErrorFieldProps {
    name: string;
    classNames?: string;
    subscribe?: any;
}

const ErrorField: React.FC<ErrorFieldProps> = ({ name, classNames }) => (
    <Field name={name} subscribe={{ touched: true, error: true }}>
        {({ meta: { touched, error } }: any) =>
            touched && error ? <span className={classNames}>{error}</span> : null
        }
    </Field>
);
export { ErrorField }