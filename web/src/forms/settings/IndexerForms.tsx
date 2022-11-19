import { Fragment, useState } from "react";
import { toast } from "react-hot-toast";
import { useMutation, useQuery } from "react-query";
import Select, { components, ControlProps, InputProps, MenuProps, OptionProps } from "react-select";
import type { FieldProps } from "formik";
import { Field, Form, Formik, FormikValues } from "formik";

import { XMarkIcon } from "@heroicons/react/24/solid";
import { Dialog, Transition } from "@headlessui/react";

import { sleep } from "../../utils";
import { queryClient } from "../../App";
import DEBUG from "../../components/debug";
import { APIClient } from "../../api/APIClient";
import { PasswordFieldWide, SwitchGroupWide, TextFieldWide } from "../../components/inputs";
import { SlideOver } from "../../components/panels";
import Toast from "../../components/notifications/Toast";
import { SelectFieldCreatable } from "../../components/inputs/select_wide";

const Input = (props: InputProps) => (
  <components.Input 
    {...props} 
    inputClassName="outline-none border-none shadow-none focus:ring-transparent"
    className="text-gray-400 dark:text-gray-100"
    children={props.children}
  />
);

const Control = (props: ControlProps) => (
  <components.Control 
    {...props} 
    className="p-1 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
    children={props.children}
  />
);

const Menu = (props: MenuProps) => (
  <components.Menu 
    {...props}
    className="dark:bg-gray-800 border border-gray-300 dark:border-gray-700 dark:text-gray-400 rounded-md shadow-sm cursor-pointer"
    children={props.children}
  />
);

const Option = (props: OptionProps) => (
  <components.Option 
    {...props}
    className="dark:text-gray-400 dark:bg-gray-800 dark:hover:bg-gray-900 dark:focus:bg-gray-900 cursor-pointer"
    children={props.children}
  />
);

// const isRequired = (message: string) => (value?: string | undefined) => (!!value ? undefined : message);

function validateField(s: IndexerSetting) {
  return (value?: string | undefined) => {
    if (s.required) {
      if (s.default !== "") {
        if (value && s.default === value) {
          return "Default value, please edit";
        }
      }
      return !!value ? undefined : "Required";
    }
  };
}

const IrcSettingFields = (ind: IndexerDefinition, indexer: string) => {
  if (indexer !== "") {
    return (
      <Fragment>
        {ind && ind.irc && ind.irc.settings && (
          <div className="border-t border-gray-200 dark:border-gray-700 py-5">
            <div className="px-4 space-y-1">
              <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">IRC</Dialog.Title>
              <p className="text-sm text-gray-500 dark:text-gray-200">
                Networks and channels are configured automatically in the background.
              </p>
            </div>

            {ind.irc.settings.map((f: IndexerSetting, idx: number) => {
              switch (f.type) {
              case "text":
                return <TextFieldWide name={`irc.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} validate={validateField(f)} />;
              case "secret":
                if (f.name === "invite_command") {
                  return <PasswordFieldWide name={`irc.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} defaultVisible={true} defaultValue={f.default} validate={validateField(f)} />;
                }
                return <PasswordFieldWide name={`irc.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} defaultValue={f.default} validate={validateField(f)} />;
              }
              return null;
            })}
          </div>
        )}
      </Fragment>
    );
  }
};

const FeedSettingFields = (ind: IndexerDefinition, indexer: string) => {
  if (indexer !== "") {
    return (
      <Fragment>
        {ind && ind.torznab && ind.torznab.settings && (
          <div className="">
            <div className="px-4 space-y-1">
              <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">Torznab</Dialog.Title>
              <p className="text-sm text-gray-500 dark:text-gray-200">
                Torznab feed
              </p>
            </div>

            <TextFieldWide name="name" label="Name" defaultValue="" />

            {ind.torznab.settings.map((f: IndexerSetting, idx: number) => {
              switch (f.type) {
              case "text":
                return <TextFieldWide name={`feed.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} validate={validateField(f)} />;
              case "secret":
                return <PasswordFieldWide name={`feed.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} defaultValue={f.default} validate={validateField(f)} />;
              }
              return null;
            })}
          </div>
        )}
      </Fragment>
    );
  }
};

const RSSFeedSettingFields = (ind: IndexerDefinition, indexer: string) => {
  if (indexer !== "") {
    return (
      <Fragment>
        {ind && ind.rss && ind.rss.settings && (
          <div className="">
            <div className="px-4 space-y-1">
              <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">RSS</Dialog.Title>
              <p className="text-sm text-gray-500 dark:text-gray-200">
                RSS feed
              </p>
            </div>

            <TextFieldWide name="name" label="Name" defaultValue="" />

            {ind.rss.settings.map((f: IndexerSetting, idx: number) => {
              switch (f.type) {
              case "text":
                return <TextFieldWide name={`feed.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} validate={validateField(f)} />;
              case "secret":
                return <PasswordFieldWide name={`feed.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} defaultValue={f.default} validate={validateField(f)} />;
              }
              return null;
            })}
          </div>
        )}
      </Fragment>
    );
  }
};

const SettingFields = (ind: IndexerDefinition, indexer: string) => {
  if (indexer !== "") {
    return (
      <div key="opt">
        {ind && ind.settings && ind.settings.map((f, idx: number) => {
          switch (f.type) {
          case "text":
            return (
              <TextFieldWide name={`settings.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} validate={validateField(f)} />
            );
          case "secret":
            return (
              <PasswordFieldWide name={`settings.${f.name}`} label={f.label} required={f.required} key={idx} help={f.help} validate={validateField(f)} />
            );
          }
          return null;
        })}
        <div hidden={true}>
          <TextFieldWide name="name" label="Name" defaultValue={ind?.name} />
        </div>
      </div>
    );
  }
};

type SelectValue = {
  label: string;
  value: string;
};

interface AddProps {
  isOpen: boolean;
  toggle: () => void;
}

export function IndexerAddForm({ isOpen, toggle }: AddProps) {
  const [indexer, setIndexer] = useState<IndexerDefinition>({} as IndexerDefinition);

  const { data } = useQuery(
    "indexerDefinition",
    () => APIClient.indexers.getSchema(),
    {
      enabled: isOpen,
      refetchOnWindowFocus: false
    }
  );

  const mutation = useMutation(
    (indexer: Indexer) => APIClient.indexers.create(indexer), {
      onSuccess: () => {
        queryClient.invalidateQueries(["indexer"]);
        toast.custom((t) => <Toast type="success" body="Indexer was added" t={t} />);
        sleep(1500);
        toggle();
      },
      onError: () => {
        toast.custom((t) => <Toast type="error" body="Indexer could not be added" t={t} />);
      }
    });

  const ircMutation = useMutation(
    (network: IrcNetworkCreate) => APIClient.irc.createNetwork(network)
  );

  const feedMutation = useMutation(
    (feed: FeedCreate) => APIClient.feeds.create(feed)
  );

  const onSubmit = (formData: FormikValues) => {
    const ind = data && data.find(i => i.identifier === formData.identifier);
    if (!ind)
      return;

    if (formData.implementation === "torznab") {
      const createFeed: FeedCreate = {
        name: formData.name,
        enabled: false,
        type: "TORZNAB",
        url: formData.feed.url,
        api_key: formData.feed.api_key,
        interval: 30,
        timeout: 60,
        indexer_id: 0
      };

      mutation.mutate(formData as Indexer, {
        onSuccess: (indexer) => {
          // @eslint-ignore
          createFeed.indexer_id = indexer.id;

          feedMutation.mutate(createFeed);
        }
      });
      return;

    } else if (formData.implementation === "rss") {
      const createFeed: FeedCreate = {
        name: formData.name,
        enabled: false,
        type: "RSS",
        url: formData.feed.url,
        interval: 30,
        timeout: 60,
        indexer_id: 0
      };

      mutation.mutate(formData as Indexer, {
        onSuccess: (indexer) => {
          // @eslint-ignore
          createFeed.indexer_id = indexer.id;

          feedMutation.mutate(createFeed);
        }
      });
      return;

    } else if (formData.implementation === "irc") {
      const channels: IrcChannel[] = [];
      if (ind.irc?.channels.length) {
        ind.irc.channels.forEach(element => {
          channels.push({
            id: 0,
            enabled: true,
            name: element,
            password: "",
            detached: false,
            monitoring: false
          });
        });
      }

      const network: IrcNetworkCreate = {
        name: ind.irc.network,
        pass: "",
        enabled: false,
        connected: false,
        server: ind.irc.server,
        port: ind.irc.port,
        tls: ind.irc.tls,
        nick: formData.irc.nick,
        auth: {
          mechanism: "NONE"
          // account: formData.irc.auth.account,
          // password: formData.irc.auth.password
        },
        invite_command: formData.irc.invite_command,
        channels: channels
      };

      if (formData.irc.auth.account !== "" && formData.irc.auth.password !== "") {
        network.auth.mechanism = "SASL_PLAIN";
        network.auth.account = formData.irc.auth.account;
        network.auth.password = formData.irc.auth.password;
      }

      mutation.mutate(formData as Indexer, {
        onSuccess: () => {
          ircMutation.mutate(network);
        }
      });
    }
  };

  return (
    <Transition.Root show={isOpen} as={Fragment}>
      <Dialog as="div" static className="fixed inset-0 overflow-hidden" open={isOpen} onClose={toggle}>
        <div className="absolute inset-0 overflow-hidden">
          <Dialog.Overlay className="absolute inset-0" />

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
              <div className="w-screen max-w-2xl dark:border-gray-700 border-l">
                <Formik
                  enableReinitialize={true}
                  initialValues={{
                    enabled: true,
                    identifier: "",
                    implementation: "irc",
                    name: "",
                    irc: {},
                    settings: {}
                  }}
                  onSubmit={onSubmit}
                >
                  {({ values }) => (
                    <Form className="h-full flex flex-col bg-white dark:bg-gray-800 shadow-xl overflow-y-scroll">
                      <div className="flex-1">
                        <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
                          <div className="flex items-start justify-between space-x-3">
                            <div className="space-y-1">
                              <Dialog.Title className="text-lg font-medium text-gray-900 dark:text-white">
                                Add indexer
                              </Dialog.Title>
                              <p className="text-sm text-gray-500 dark:text-gray-200">
                                Add indexer.
                              </p>
                            </div>
                            <div className="h-7 flex items-center">
                              <button
                                type="button"
                                className="bg-white dark:bg-gray-700 rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                                onClick={toggle}
                              >
                                <span className="sr-only">Close panel</span>
                                <XMarkIcon className="h-6 w-6" aria-hidden="true" />
                              </button>
                            </div>
                          </div>
                        </div>

                        <div className="py-6 space-y-4 divide-y divide-gray-200 dark:divide-gray-700">
                          <div className="py-4 flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:py-4">
                            <div>
                              <label
                                htmlFor="identifier"
                                className="block text-sm font-medium text-gray-900 dark:text-white"
                              >
                                Indexer
                              </label>
                            </div>
                            <div className="sm:col-span-2">
                              <Field name="identifier" type="select">
                                {({ field, form: { setFieldValue, resetForm } }: FieldProps) => (
                                  <Select {...field}
                                    isClearable={true}
                                    isSearchable={true}
                                    components={{ Input, Control, Menu, Option }}
                                    placeholder="Choose an indexer"
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
                                    value={field?.value && field.value.value}
                                    onChange={(option: unknown) => {
                                      resetForm();

                                      if (option != null) {
                                        const opt = option as SelectValue;
                                        setFieldValue("name", opt.label ?? "");
                                        setFieldValue(field.name, opt.value ?? "");

                                        const ind = data && data.find(i => i.identifier === opt.value);
                                        if (ind) {
                                          setIndexer(ind);
                                          setFieldValue("implementation", ind.implementation);

                                          if (ind.irc && ind.irc.settings) {
                                            setFieldValue("base_url", ind.urls[0]);
                                            ind.irc.settings.forEach((s) => {
                                              setFieldValue(`irc.${s.name}`, s.default ?? "");
                                            });
                                          }
                                        }
                                      }
                                    }}
                                    options={data && data.sort((a, b) => a.name.localeCompare(b.name)).map(v => ({
                                      label: v.name,
                                      value: v.identifier
                                    }))} 
                                  />
                                )}
                              </Field>

                            </div>
                          </div>

                          <SwitchGroupWide name="enabled" label="Enabled" />

                          {indexer.implementation == "irc" && (
                            <SelectFieldCreatable
                              name="base_url"
                              label="Base URL"
                              help="Override baseurl if it's blocked by your ISP."
                              options={indexer.urls.map(u => ({ value: u ?? u, label: u ?? u, key: u ?? u })) }
                            />
                          )}

                          {SettingFields(indexer, values.identifier)}

                        </div>

                        {IrcSettingFields(indexer, values.identifier)}
                        {FeedSettingFields(indexer, values.identifier)}
                        {RSSFeedSettingFields(indexer, values.identifier)}
                      </div>

                      <div
                        className="flex-shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
                        <div className="space-x-3 flex justify-end">
                          <button
                            type="button"
                            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                            onClick={toggle}
                          >
                            Cancel
                          </button>
                          <button
                            type="submit"
                            className="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                          >
                            Save
                          </button>
                        </div>
                      </div>

                      <DEBUG values={values} />
                    </Form>
                  )}
                </Formik>
              </div>

            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition.Root>
  );
}

interface UpdateProps {
    isOpen: boolean;
    toggle: () => void;
    indexer: IndexerDefinition;
}

export function IndexerUpdateForm({ isOpen, toggle, indexer }: UpdateProps) {
  const mutation = useMutation((indexer: Indexer) => APIClient.indexers.update(indexer), {
    onSuccess: () => {
      queryClient.invalidateQueries(["indexer"]);
      toast.custom((t) => <Toast type="success" body={`${indexer.name} was updated successfully`} t={t} />);
      sleep(1500);

      toggle();
    }
  });

  const deleteMutation = useMutation((id: number) => APIClient.indexers.delete(id), {
    onSuccess: () => {
      queryClient.invalidateQueries(["indexer"]);
      toast.custom((t) => <Toast type="success" body={`${indexer.name} was deleted.`} t={t} />);

      toggle();
    }
  });

  const onSubmit = (data: unknown) => {
    // TODO clear data depending on type
    mutation.mutate(data as Indexer);
  };

  const deleteAction = () => {
    deleteMutation.mutate(indexer.id ?? 0);
  };

  const renderSettingFields = (settings: IndexerSetting[]) => {
    if (settings === undefined) {
      return null;
    }

    return (
      <div key="opt">
        {settings.map((f: IndexerSetting, idx: number) => {
          switch (f.type) {
          case "text":
            return (
              <TextFieldWide name={`settings.${f.name}`} label={f.label} key={idx} help={f.help} />
            );
          case "secret":
            return (
              <PasswordFieldWide name={`settings.${f.name}`} label={f.label} key={idx} help={f.help} />
            );
          }
          return null;
        })}
      </div>
    );
  };

  const initialValues = {
    id: indexer.id,
    name: indexer.name,
    enabled: indexer.enabled,
    identifier: indexer.identifier,
    implementation: indexer.implementation,
    base_url: indexer.base_url,
    settings: indexer.settings?.reduce(
      (o: Record<string, string>, obj: IndexerSetting) => ({
        ...o,
        [obj.name]: obj.value
      } as Record<string, string>),
      {} as Record<string, string>
    )
  };

  return (
    <SlideOver
      type="UPDATE"
      title="Indexer"
      isOpen={isOpen}
      toggle={toggle}
      deleteAction={deleteAction}
      onSubmit={onSubmit}
      initialValues={initialValues}
    >
      {() => (
        <div className="py-2 space-y-6 sm:py-0 sm:space-y-0 divide-y divide-gray-200 dark:divide-gray-700">
          <div className="space-y-1 p-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4">
            <label
              htmlFor="name"
              className="block text-sm font-medium text-gray-900 dark:text-white sm:mt-px sm:pt-2"
            >
              Name
            </label>
            <Field name="name">
              {({ field, meta }: FieldProps) => (
                <div className="sm:col-span-2">
                  <input
                    type="text"
                    {...field}
                    className="block w-full shadow-sm dark:bg-gray-800 sm:text-sm dark:text-white focus:ring-blue-500 focus:border-blue-500 border-gray-300 dark:border-gray-700 rounded-md"
                  />
                  {meta.touched && meta.error && <span>{meta.error}</span>}
                </div>
              )}
            </Field>
          </div>
          <SwitchGroupWide name="enabled" label="Enabled" />

          {indexer.implementation == "irc" && (
            <SelectFieldCreatable
              name="base_url"
              label="Base URL"
              help="Override baseurl if it's blocked by your ISP."
              options={indexer.urls.map(u => ({ value: u ?? u, label: u ?? u, key: u ?? u })) }
            />
          )}

          {renderSettingFields(indexer.settings)}
        </div>
      )}
    </SlideOver>
  );
}