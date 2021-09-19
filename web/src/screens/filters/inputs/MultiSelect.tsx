import React from "react";
import RMSC from "react-multi-select-component";
import { Field } from "formik";
import { classNames, COL_WIDTHS } from "../../../styles/utils";

interface Props {
    label?: string;
    options?: [] | any;
    name: string;
    className?: string;
    columns?: COL_WIDTHS;
}

const MultiSelect: React.FC<Props> = ({
    name,
    label,
    options,
    className,
    columns
}) => (
    <div
        className={classNames(
            columns ? `col-span-${columns}` : "col-span-12"
        )}
    >
        <label
            className="block uppercase tracking-wide text-gray-700 text-xs font-bold mb-2"
            htmlFor={label}
        >
            {label}
        </label>

        <Field name={name} type="select" multiple={true}>
            {({
                field,
                form: { setFieldValue },
            }: any) => {
                return (
                    <RMSC
                        {...field}
                        type="select"
                        options={options}
                        labelledBy={name}
                        value={field.value && field.value.map((item: any) => options.find((o: any) => o.value === item))}
                        onChange={(values: any) => {
                            let am = values && values.map((i: any) => i.value)

                            setFieldValue(field.name, am)
                        }}
                    />
                )
            }}
        </Field>
    </div>
);

export default MultiSelect;