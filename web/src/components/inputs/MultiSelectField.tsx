import React from "react";
import {Field} from "react-final-form";
import { MultiSelect } from "react-multi-select-component";
import {classNames, COL_WIDTHS} from "../../styles/utils";

interface Props {
    label?: string;
    options?: [] | any;
    name: string;
    className?: string;
    columns?: COL_WIDTHS;
}

const MultiSelectField: React.FC<Props> = ({
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
                className="block mb-2 text-xs font-bold tracking-wide text-gray-700 uppercase"
                htmlFor={label}
            >
                {label}
            </label>
            <Field
                name={name}
                parse={val => val && val.map((item: any) => item.value)}
                format={val =>
                    val &&
                    val.map((item: any) => options.find((o: any) => o.value === item))
                }
                render={({input, meta}) => (
                    <MultiSelect
                        {...input}
                        options={options}
                        labelledBy={name}
                    />
                )}
            />
        </div>
 );

export default MultiSelectField;
