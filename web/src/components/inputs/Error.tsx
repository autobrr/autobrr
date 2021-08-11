import React from "react";
import { Field } from "react-final-form";

interface Props {
    name: string;
    classNames?: string;
    subscribe?: any;
}

const Error: React.FC<Props> = ({ name, classNames }) => (
    <Field
        name={name}
        subscribe={{ touched: true, error: true }}
        render={({ meta: { touched, error } }) =>
            touched && error ? <span className={classNames}>{error}</span> : null
        }
    />
);

export default Error;
