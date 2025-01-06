/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Field, useFormikContext } from "formik";
import { RadioGroup, Description, Label, Radio } from "@headlessui/react";
import { classNames } from "@utils";

export interface radioFieldsetOption {
    label: string;
    description: string;
    value: string;
    type?: string;
}

interface props {
    name: string;
    legend: string;
    options: radioFieldsetOption[];
}

interface anyObj {
    [key: string]: string
}

function RadioFieldsetWide({ name, legend, options }: props) {
  const {
    values,
    setFieldValue
  } = useFormikContext<anyObj>();

  const onChange = (value: string) => {
    setFieldValue(name, value);
  };

  return (
    <fieldset>
      <div className="space-y-2 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:items-start sm:py-4">
        <div>
          <legend className="text-sm font-medium text-gray-900 dark:text-white">
            {legend}
          </legend>
        </div>
        <div className="space-y-5 sm:col-span-2">
          <div className="space-y-5 sm:mt-0">
            <Field name={name} type="radio">
              {() => (
                <RadioGroup value={values[name]} onChange={onChange}>
                  <Label className="sr-only">
                    {legend}
                  </Label>
                  <div className="bg-white dark:bg-gray-800 rounded-md -space-y-px">
                    {options.map((setting, settingIdx) => (
                      <Radio
                        key={setting.value}
                        value={setting.value}
                        className={({ checked }) =>
                          classNames(
                            settingIdx === 0
                              ? "rounded-tl-md rounded-tr-md"
                              : "",
                            settingIdx === options.length - 1
                              ? "rounded-bl-md rounded-br-md"
                              : "",
                            checked
                              ? "border-1 bg-blue-100 dark:bg-blue-900 border-blue-400 dark:border-blue-600 z-10"
                              : "border-gray-200 dark:border-gray-700",
                            "relative border p-4 flex cursor-pointer focus:outline-none"
                          )
                        }
                      >
                        {({ checked }) => (
                          <>
                            <span
                              className={classNames(
                                checked
                                  ? "bg-blue-600 dark:bg-blue-500 border-transparent"
                                  : "bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-300",
                                "h-6 w-6 mt-1 cursor-pointer rounded-full border flex items-center justify-center flex-shrink-0"
                              )}
                              aria-hidden="true"
                            />
                            <div className="ml-3 flex flex-col w-full">
                              <Label
                                as="span"
                                className={classNames(
                                  "block text-md text-gray-900 dark:text-gray-300",
                                  checked ? "font-bold" : "font-medium"
                                )}
                              >
                                <div className="flex justify-between">
                                  {setting.label}
                                  {setting.type && <span className="rounded bg-orange-500 text-orange-900 px-1 ml-2 text-sm">{setting.type}</span>}
                                </div>
                              </Label>
                              <Description
                                as="span"
                                className="block text-sm text-gray-700 dark:text-gray-400"
                              >
                                {setting.description}
                              </Description>
                            </div>
                          </>
                        )}
                      </Radio>
                    ))}
                  </div>
                </RadioGroup>
              )}
            </Field>
          </div>
        </div>
      </div>
    </fieldset>
  );
}

export { RadioFieldsetWide };
