import { Field, useFormState } from "react-final-form";
import { RadioGroup } from "@headlessui/react";
import { classNames } from "../../../styles/utils";
import { Fragment } from "react";
import { radioFieldsetOption } from "../RadioFieldset";

interface props {
  name: string;
  legend: string;
  options: radioFieldsetOption[];
}

function RadioFieldsetWide({ name, legend, options }: props) {
  const { values } = useFormState();

  return (
    <fieldset>
      <div className="space-y-2 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:items-start sm:px-6 sm:py-5">
        <div>
          <legend className="text-sm font-medium text-gray-900 dark:text-white">
            {legend}
          </legend>
        </div>
        <div className="space-y-5 sm:col-span-2">
          <div className="space-y-5 sm:mt-0">
            <Field
              name={name}
              type="radio"
              render={({ input }) => (
                <RadioGroup value={values[name]} onChange={input.onChange}>
                  <RadioGroup.Label className="sr-only">
                    Privacy setting
                  </RadioGroup.Label>
                  <div className="bg-white dark:bg-gray-800 rounded-md -space-y-px">
                    {options.map((setting, settingIdx) => (
                      <RadioGroup.Option
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
                              ? "bg-indigo-50 dark:bg-gray-700 border-indigo-200 dark:border-blue-600 z-10"
                              : "border-gray-200 dark:border-gray-700",
                            "relative border p-4 flex cursor-pointer focus:outline-none"
                          )
                        }
                      >
                        {({ active, checked }) => (
                          <Fragment>
                            <span
                              className={classNames(
                                checked
                                  ? "bg-indigo-600 dark:bg-blue-600 border-transparent"
                                  : "bg-white border-gray-300 dark:border-gray-300",
                                active
                                  ? "ring-2 ring-offset-2 ring-indigo-500 dark:ring-blue-500"
                                  : "",
                                "h-4 w-4 mt-0.5 cursor-pointer rounded-full border flex items-center justify-center"
                              )}
                              aria-hidden="true"
                            >
                              <span className="rounded-full bg-white w-1.5 h-1.5" />
                            </span>
                            <div className="ml-3 flex flex-col">
                              <RadioGroup.Label
                                as="span"
                                className={classNames(
                                  checked ? "text-indigo-900 dark:text-blue-500" : "text-gray-900 dark:text-gray-300",
                                  "block text-sm font-medium"
                                )}
                              >
                                {setting.label}
                              </RadioGroup.Label>
                              <RadioGroup.Description
                                as="span"
                                className={classNames(
                                  checked ? "text-indigo-700 dark:text-blue-500" : "text-gray-500",
                                  "block text-sm"
                                )}
                              >
                                {setting.description}
                              </RadioGroup.Description>
                            </div>
                          </Fragment>
                        )}
                      </RadioGroup.Option>
                    ))}
                  </div>
                </RadioGroup>
              )}
            />
          </div>
        </div>
      </div>
    </fieldset>
  );
}

export default RadioFieldsetWide;
