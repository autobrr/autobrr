import {Fragment} from "react";
import {useMutation} from "react-query";
import {Network} from "../../domain/interfaces";
import {Dialog, Transition} from "@headlessui/react";
import {XIcon} from "@heroicons/react/solid";
import {Field, Form} from "react-final-form";
import DEBUG from "../../components/debug";
import {SwitchGroup, TextAreaWide, TextFieldWide} from "../../components/inputs";
import {queryClient} from "../../index";

import arrayMutators from "final-form-arrays";
import { FieldArray } from "react-final-form-arrays";
import {classNames} from "../../styles/utils";
import APIClient from "../../api/APIClient";


// interface radioFieldsetOption {
//     label: string;
//     description: string;
//     value: string;
// }

// const saslTypeOptions: radioFieldsetOption[] = [
//     {label: "None", description: "None", value: ""},
//     {label: "Plain", description: "SASL plain", value: "PLAIN"},
//     {label: "NickServ", description: "/NS identify", value: "NICKSERV"},
// ];

function IrcNetworkAddForm({isOpen, toggle}: any) {
    const mutation = useMutation((network: Network) => APIClient.irc.createNetwork(network), {
        onSuccess: data => {
            queryClient.invalidateQueries(['networks']);
            toggle()
        }
    })

    const onSubmit = (data: any) => {
        console.log(data)

        // easy way to split textarea lines into array of strings for each newline.
        // parse on the field didn't really work.
        let cmds = data.connect_commands && data.connect_commands.length > 0 ? data.connect_commands.replace(/\r\n/g,"\n").split("\n") : [];
        data.connect_commands = cmds
        console.log("formated", data)

        mutation.mutate(data)
    };

    const validate = (values: any) => {
        const errors = {} as any;

        if (!values.name) {
            errors.name = "Required";
        }

        if (!values.addr) {
            errors.addr = "Required";
        }

        if (!values.nick) {
            errors.nick = "Required";
        }

        return errors;
    }

    return (
        <Transition.Root show={isOpen} as={Fragment}>
            <Dialog as="div" static className="fixed inset-0 overflow-hidden" open={isOpen} onClose={toggle}>
                <div className="absolute inset-0 overflow-hidden">
                    <Dialog.Overlay className="absolute inset-0"/>

                    <div className="fixed inset-y-0 right-0 pl-10 max-w-full flex sm:pl-16">
                        <Transition.Child
                            as={Fragment}
                            enter="transform transition ease-in-out duration-500 sm:duration-700"
                            enterFrom="translate-x-full"
                            enterTo="translate-x-0"
                            leave="transform transition ease-in-out duration-500 sm:duration-700"
                            leaveFrom="translate-x-0"
                            leaveTo="translate-x-full"
                        >
                            <div className="w-screen max-w-2xl">

                                <Form
                                    initialValues={{
                                        name: "",
                                        enabled: true,
                                        addr: "",
                                        tls: false,
                                        nick: "",
                                        pass: "",
                                        // connect_commands: "",
                                        // sasl: {
                                        //     mechanism: "",
                                        //     plain: {
                                        //         username: "",
                                        //         password: "",
                                        //     }
                                        // },
                                    }}
                                    mutators={{
                                        ...arrayMutators
                                    }}
                                    validate={validate}
                                    onSubmit={onSubmit}
                                >
                                    {({handleSubmit, values, pristine, invalid}) => {
                                        return (
                                            <form className="h-full flex flex-col bg-white shadow-xl overflow-y-scroll"
                                                  onSubmit={handleSubmit}>
                                                <div className="flex-1">
                                                    {/* Header */}
                                                    <div className="px-4 py-6 bg-gray-50 sm:px-6">
                                                        <div className="flex items-start justify-between space-x-3">
                                                            <div className="space-y-1">
                                                                <Dialog.Title
                                                                    className="text-lg font-medium text-gray-900">Add
                                                                    network</Dialog.Title>
                                                                <p className="text-sm text-gray-500">
                                                                    Add irc network.
                                                                </p>
                                                            </div>
                                                            <div className="h-7 flex items-center">
                                                                <button
                                                                    type="button"
                                                                    className="bg-white rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                                                                    onClick={toggle}
                                                                >
                                                                    <span className="sr-only">Close panel</span>
                                                                    <XIcon className="h-6 w-6" aria-hidden="true"/>
                                                                </button>
                                                            </div>
                                                        </div>
                                                    </div>
                                                    
                                                    <TextFieldWide name="name" label="Name" placeholder="Name" required={true} />

                                                    <div className="py-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">

                                                        <div
                                                            className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                                                            <SwitchGroup name="enabled" label="Enabled"/>
                                                        </div>

                                                        <div>

                                                            <TextFieldWide name="addr" label="Address" placeholder="Address:port eg irc.server.net:6697" required={true} />

                                                            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
                                                                <SwitchGroup name="tls" label="TLS"/>
                                                            </div>

                                                            <TextFieldWide name="nick" label="Nick" placeholder="Nick" required={true} />

                                                            <TextFieldWide name="password" label="Password" placeholder="Network password" />

                                                            <TextAreaWide name="connect_commands" label="Connect commands" placeholder="/msg test this" />


              {/*                                              <Field*/}
              {/*                                                  name="sasl.mechanism"*/}
              {/*                                                  type="select"*/}
              {/*                                                  render={({input}) => (*/}
              {/*                                                      <Listbox value={input.value} onChange={input.onChange}>*/}
              {/*                                                          {({open}) => (*/}
              {/*                                                              <div className="space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">*/}
              {/*                                                                  <div>*/}
              {/*                                                                      <Listbox.Label className="block text-sm font-medium text-gray-900 sm:mt-px sm:pt-2">SASL / auth</Listbox.Label>*/}
              {/*                                                                  </div>*/}
              {/*                                                                  <div className="sm:col-span-2 relative">*/}
              {/*                                                                      <Listbox.Button*/}
              {/*                                                                          className="bg-white relative w-full border border-gray-300 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">*/}
              {/*                                                                          <span className="block truncate">{input.value ? saslTypeOptions.find(c => c.value === input.value)!.label : "Choose auth method"}</span>*/}
              {/*                                                                          <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">*/}
              {/*  <SelectorIcon className="h-5 w-5 text-gray-400" aria-hidden="true"/>*/}
              {/*</span>*/}
              {/*                                                                      </Listbox.Button>*/}

              {/*                                                                      <Transition*/}
              {/*                                                                          show={open}*/}
              {/*                                                                          as={Fragment}*/}
              {/*                                                                          leave="transition ease-in duration-100"*/}
              {/*                                                                          leaveFrom="opacity-100"*/}
              {/*                                                                          leaveTo="opacity-0"*/}
              {/*                                                                      >*/}
              {/*                                                                          <Listbox.Options*/}
              {/*                                                                              static*/}
              {/*                                                                              className="absolute z-10 mt-1 w-full bg-white shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm"*/}
              {/*                                                                          >*/}
              {/*                                                                              {saslTypeOptions.map((opt: any) => (*/}
              {/*                                                                                  <Listbox.Option*/}
              {/*                                                                                      key={opt.value}*/}
              {/*                                                                                      className={({active}) =>*/}
              {/*                                                                                          classNames(*/}
              {/*                                                                                              active ? 'text-white bg-indigo-600' : 'text-gray-900',*/}
              {/*                                                                                              'cursor-default select-none relative py-2 pl-3 pr-9'*/}
              {/*                                                                                          )*/}
              {/*                                                                                      }*/}
              {/*                                                                                      value={opt.value}*/}
              {/*                                                                                  >*/}
              {/*                                                                                      {({selected, active}) => (*/}
              {/*                                                                                          <>*/}
              {/*          <span className={classNames(selected ? 'font-semibold' : 'font-normal', 'block truncate')}>*/}
              {/*            {opt.label}*/}
              {/*          </span>*/}

              {/*                                                                                              {selected ? (*/}
              {/*                                                                                                  <span*/}
              {/*                                                                                                      className={classNames(*/}
              {/*                                                                                                          active ? 'text-white' : 'text-indigo-600',*/}
              {/*                                                                                                          'absolute inset-y-0 right-0 flex items-center pr-4'*/}
              {/*                                                                                                      )}*/}
              {/*                                                                                                  >*/}
              {/*              <CheckIcon className="h-5 w-5" aria-hidden="true"/>*/}
              {/*            </span>*/}
              {/*                                                                                              ) : null}*/}
              {/*                                                                                          </>*/}
              {/*                                                                                      )}*/}
              {/*                                                                                  </Listbox.Option>*/}
              {/*                                                                              ))}*/}
              {/*                                                                          </Listbox.Options>*/}
              {/*                                                                      </Transition>*/}
              {/*                                                                  </div>*/}
              {/*                                                              </div>*/}
              {/*                                                          )}*/}
              {/*                                                      </Listbox>*/}
              {/*                                                  )} />*/}
                                                        </div>
                                                    </div>

                                                    <div className="p-6">

                                                        <FieldArray name="channels">
                                                            {({ fields }) => (
                                                                <div className="flex flex-col border-2 border-dashed p-4">
                                                                    {fields && (fields.length as any) > 0 ? (
                                                                        fields.map((name, index) => (
                                                                            <div key={name} className="flex justify-between">
                                                                                <div className="flex">
                                                                                    <Field
                                                                                        name={`${name}.name`}
                                                                                        component="input"
                                                                                        type="text"
                                                                                        placeholder="#Channel"
                                                                                        className="mr-4 focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 block w-full shadow-sm sm:text-sm rounded-md"
                                                                                    />
                                                                                    <Field
                                                                                        name={`${name}.password`}
                                                                                        component="input"
                                                                                        type="text"
                                                                                        placeholder="Password"
                                                                                        className="focus:ring-indigo-500 focus:border-indigo-500 border-gray-300 block w-full shadow-sm sm:text-sm rounded-md"
                                                                                    />
                                                                                </div>

                                                                                <button
                                                                                    type="button"
                                                                                    className="bg-white rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                                                                                    onClick={() => fields.remove(index)}
                                                                                >
                                                                                    <span className="sr-only">Remove</span>
                                                                                    <XIcon className="h-6 w-6" aria-hidden="true"/>
                                                                                </button>
                                                                            </div>
                                                                        ))
                                                                    ) : (
                                                                        <span className="text-center text-sm text-grey-darker">
                                                                            No channels!
                                                                        </span>
                                                                    )}
                                                                    <button
                                                                        type="button"
                                                                        className="border my-4 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 rounded self-center text-center"
                                                                        onClick={() => fields.push({ name: "", password: "" })}
                                                                    >
                                                                        Add Channel
                                                                    </button>
                                                                </div>
                                                            )}
                                                        </FieldArray>
                                                    </div>
                                                </div>

                                                <div
                                                    className="flex-shrink-0 px-4 border-t border-gray-200 py-5 sm:px-6">
                                                    <div className="space-x-3 flex justify-end">
                                                        <button
                                                            type="button"
                                                            className="bg-white py-2 px-4 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                                                            onClick={toggle}
                                                        >
                                                            Cancel
                                                        </button>
                                                        <button
                                                            type="submit"
                                                            disabled={pristine || invalid}
                                                            // className="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                                                            className={classNames(pristine || invalid ? "bg-indigo-300" : "bg-indigo-600 hover:bg-indigo-700","inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500")}
                                                        >
                                                            Create
                                                        </button>
                                                    </div>
                                                </div>

                                                <DEBUG values={values}/>
                                            </form>
                                        )
                                    }}
                                </Form>
                            </div>

                        </Transition.Child>
                    </div>
                </div>
            </Dialog>
        </Transition.Root>
    )
}

export default IrcNetworkAddForm;
