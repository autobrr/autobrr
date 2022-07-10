import React, { Fragment, useRef } from "react";
import { useMutation, useQuery } from "react-query";
import {
  NavLink,
  Route,
  Routes,
  useLocation,
  useNavigate,
  useParams
} from "react-router-dom";
import { toast } from "react-hot-toast";
import { Field, FieldArray, FieldProps, Form, Formik, FormikValues } from "formik";
import { Dialog, Transition, Switch as SwitchBasic } from "@headlessui/react";
import { ChevronDownIcon, ChevronRightIcon } from "@heroicons/react/solid";

import {
  CONTAINER_OPTIONS,
  CODECS_OPTIONS,
  RESOLUTION_OPTIONS,
  SOURCES_OPTIONS,
  ActionTypeNameMap,
  ActionTypeOptions,
  HDR_OPTIONS,
  FORMATS_OPTIONS,
  SOURCES_MUSIC_OPTIONS,
  QUALITY_MUSIC_OPTIONS,
  RELEASE_TYPE_MUSIC_OPTIONS,
  OTHER_OPTIONS,
  ORIGIN_OPTIONS,
  downloadsPerUnitOptions
} from "../../domain/constants";
import { queryClient } from "../../App";
import { APIClient } from "../../api/APIClient";
import { useToggle } from "../../hooks/hooks";
import { classNames } from "../../utils";

import {
  NumberField,
  TextField,
  SwitchGroup,
  Select,
  MultiSelect,
  DownloadClientSelect,
  IndexerMultiSelect,
  CheckboxField
} from "../../components/inputs";
import DEBUG from "../../components/debug";
import Toast from "../../components/notifications/Toast";
import { AlertWarning } from "../../components/alerts";
import { DeleteModal } from "../../components/modals";
import { TitleSubtitle } from "../../components/headings";
import { EmptyListState } from "../../components/emptystates";

interface tabType {
  name: string;
  href: string;
}

const tabs: tabType[] = [
  { name: "General", href: "" },
  { name: "Movies and TV", href: "movies-tv" },
  { name: "Music", href: "music" },
  { name: "Advanced", href: "advanced" },
  { name: "Actions", href: "actions" }
];

export interface NavLinkProps {
  item: tabType;
}

function TabNavLink({ item }: NavLinkProps) {
  const location = useLocation();
  const splitLocation = location.pathname.split("/");

  // we need to clean the / if it's a base root path
  return (
    <NavLink
      key={item.name}
      to={item.href}
      end
      className={({ isActive }) => classNames(
        "text-gray-500 hover:text-purple-600 dark:hover:text-white hover:border-purple-600 dark:hover:border-blue-500 whitespace-nowrap py-4 px-1 font-medium text-sm",
        isActive ? "border-b-2 border-purple-600 dark:border-blue-500 text-purple-600 dark:text-white" : ""
      )}
      aria-current={splitLocation[2] === item.href ? "page" : undefined}
    >
      {item.name}
    </NavLink>
  );
}

interface FormButtonsGroupProps {
  values: FormikValues;
  deleteAction: () => void;
  reset: () => void;
  dirty?: boolean;
}

const FormButtonsGroup = ({ values, deleteAction, reset }: FormButtonsGroupProps) => {
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  const cancelModalButtonRef = useRef(null);

  return (
    <div className="pt-6 divide-y divide-gray-200 dark:divide-gray-700">
      <DeleteModal
        isOpen={deleteModalIsOpen}
        toggle={toggleDeleteModal}
        buttonRef={cancelModalButtonRef}
        deleteAction={deleteAction}
        title={`Remove filter: ${values.name}`}
        text="Are you sure you want to remove this filter? This action cannot be undone."
      />

      <div className="mt-4 pt-4 flex justify-between">
        <button
          type="button"
          className="inline-flex items-center justify-center px-4 py-2 rounded-md text-red-700 dark:text-red-500 light:bg-red-100 light:hover:bg-red-200 dark:hover:text-red-400 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
          onClick={toggleDeleteModal}
        >
                    Remove
        </button>

        <div>
          {/* {dirty && <span className="mr-4 text-sm text-gray-500">Unsaved changes..</span>} */}
          <button
            type="button"
            className="light:bg-white light:border light:border-gray-300 rounded-md py-2 px-4 inline-flex justify-center text-sm font-medium text-gray-700 dark:text-gray-500 light:hover:bg-gray-50 dark:hover:text-gray-300 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            onClick={reset}
          >
            Cancel
          </button>
          <button
            type="submit"
            className="ml-4 relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
};

export default function FilterDetails() {
  const navigate = useNavigate();
  const { filterId } = useParams<{ filterId: string }>();

  const { isLoading, data: filter } = useQuery(
    ["filters", filterId],
    () => APIClient.filters.getByID(parseInt(filterId ?? "0")),
    {
      retry: false,
      refetchOnWindowFocus: false,
      onError: () => navigate("./")
    }
  );

  const updateMutation = useMutation(
    (filter: Filter) => APIClient.filters.update(filter),
    {
      onSuccess: (_, currentFilter) => {
        toast.custom((t) => (
          <Toast type="success" body={`${currentFilter.name} was updated successfully`} t={t} />
        ));
        queryClient.refetchQueries(["filters"]);
        // queryClient.invalidateQueries(["filters", currentFilter.id]);
      }
    }
  );

  const deleteMutation = useMutation((id: number) => APIClient.filters.delete(id), {
    onSuccess: () => {
      toast.custom((t) => (
        <Toast type="success" body={`${filter?.name} was deleted`} t={t} />
      ));

      // Invalidate filters just in case, most likely not necessary but can't hurt.
      queryClient.invalidateQueries(["filters"]);

      // redirect
      navigate("/filters");
    }
  });

  if (isLoading) {
    return null;
  }

  if (!filter) {
    return null;
  }

  const handleSubmit = (data: Filter) => {
    // force set method and type on webhook actions
    // TODO add options for these
    data.actions.forEach((a: Action) => {
      if (a.type === "WEBHOOK") {
        a.webhook_method = "POST";
        a.webhook_type = "JSON";
      }
    });

    updateMutation.mutate(data);
  };

  const deleteAction = () => {
    deleteMutation.mutate(filter.id);
  };

  return (
    <main>
      <header className="py-10">
        <div className="max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center">
          <h1 className="text-3xl font-bold text-black dark:text-white">
            <NavLink to="/filters">
              Filters
            </NavLink>
          </h1>
          <ChevronRightIcon className="h-6 w-6 text-gray-500" aria-hidden="true" />
          <h1 className="text-3xl font-bold text-black dark:text-white">{filter.name}</h1>
        </div>
      </header>
      <div className="max-w-screen-xl mx-auto pb-12 px-4 sm:px-6 lg:px-8">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
          <div className="pt-1 px-4 pb-6 block">
            <div className="border-b border-gray-200 dark:border-gray-700">
              <nav className="-mb-px flex space-x-6 sm:space-x-8 overflow-x-auto">
                {tabs.map((tab) => (
                  <TabNavLink item={tab} key={tab.href} />
                ))}
              </nav>
            </div>

            <Formik
              initialValues={{
                id: filter.id,
                name: filter.name,
                enabled: filter.enabled || false,
                min_size: filter.min_size,
                max_size: filter.max_size,
                delay: filter.delay,
                priority: filter.priority,
                max_downloads: filter.max_downloads,
                max_downloads_unit: filter.max_downloads_unit,
                use_regex: filter.use_regex || false,
                shows: filter.shows,
                years: filter.years,
                resolutions: filter.resolutions || [],
                sources: filter.sources || [],
                codecs: filter.codecs || [],
                containers: filter.containers || [],
                match_hdr: filter.match_hdr || [],
                except_hdr: filter.except_hdr || [],
                match_other: filter.match_other || [],
                except_other: filter.except_other || [],
                seasons: filter.seasons,
                episodes: filter.episodes,
                match_releases: filter.match_releases,
                except_releases: filter.except_releases,
                match_release_groups: filter.match_release_groups,
                except_release_groups: filter.except_release_groups,
                match_categories: filter.match_categories,
                except_categories: filter.except_categories,
                tags: filter.tags,
                except_tags: filter.except_tags,
                match_uploaders: filter.match_uploaders,
                except_uploaders: filter.except_uploaders,
                freeleech: filter.freeleech,
                freeleech_percent: filter.freeleech_percent,
                formats: filter.formats || [],
                quality: filter.quality || [],
                media: filter.media || [],
                match_release_types: filter.match_release_types || [],
                log_score: filter.log_score,
                log: filter.log,
                cue: filter.cue,
                perfect_flac: filter.perfect_flac,
                artists: filter.artists,
                albums: filter.albums,
                origins: filter.origins || [],
                indexers: filter.indexers || [],
                actions: filter.actions || []
              } as Filter}
              onSubmit={handleSubmit}
            >
              {({ values, dirty, resetForm }) => (
                <Form>
                  <Routes>
                    <Route index element={<General />} />
                    <Route path="movies-tv" element={<MoviesTv />} />
                    <Route path="music" element={<Music />} />
                    <Route path="advanced" element={<Advanced />} />
                    <Route path="actions" element={<FilterActions filter={filter} values={values} />}
                    />
                  </Routes>
                  <FormButtonsGroup values={values} deleteAction={deleteAction} dirty={dirty} reset={resetForm} />
                  <DEBUG values={values} />
                </Form>
              )}
            </Formik>
          </div>
        </div>
      </div>
    </main>
  );
}

export function General() {
  const { isLoading, data: indexers } = useQuery(
    ["filters", "indexer_list"],
    () => APIClient.indexers.getOptions(),
    { refetchOnWindowFocus: false }
  );

  const opts = indexers && indexers.length > 0 ? indexers.map(v => ({
    label: v.name,
    value: v.id
  })) : [];

  return (
    <div>
      <div className="mt-6 lg:pb-8">

        <div className="mt-6 grid grid-cols-12 gap-6">
          <TextField name="name" label="Filter name" columns={6} placeholder="eg. Filter 1" />

          <div className="col-span-6">
            {!isLoading && <IndexerMultiSelect name="indexers" options={opts} label="Indexers" columns={6} />}
          </div>
        </div>
      </div>

      <div className="mt-6 lg:pb-8">
        <TitleSubtitle title="Rules" subtitle="Specify rules on how torrents should be handled/selected" />

        <div className="mt-6 grid grid-cols-12 gap-6">
          <TextField name="min_size" label="Min size" columns={6} placeholder="" />
          <TextField name="max_size" label="Max size" columns={6} placeholder="" />
          <NumberField name="delay" label="Delay" placeholder="" />
          <NumberField name="priority" label="Priority" placeholder="" />

          <NumberField name="max_downloads" label="Max downloads" placeholder="" />
          <Select name="max_downloads_unit" label="Max downloads per" options={downloadsPerUnitOptions}  optionDefaultText="Select unit" />
        </div>
      </div>

      <div className="border-t dark:border-gray-700">
        <SwitchGroup name="enabled" label="Enabled" description="Enable or disable this filter" />
      </div>

    </div>
  );
}

export function MoviesTv() {
  return (
    <div>
      <div className="mt-6 grid grid-cols-12 gap-6">
        <TextField name="shows" label="Movies / Shows" columns={8} placeholder="eg. Movie,Show 1,Show?2" />
        <TextField name="years" label="Years" columns={4} placeholder="eg. 2018,2019-2021" />
      </div>

      <div className="mt-6 lg:pb-8">
        <TitleSubtitle title="Seasons and Episodes" subtitle="Set season and episode match constraints" />

        <div className="mt-6 grid grid-cols-12 gap-6">
          <TextField name="seasons" label="Seasons" columns={8} placeholder="eg. 1,3,2-6" />
          <TextField name="episodes" label="Episodes" columns={4} placeholder="eg. 2,4,10-20" />
        </div>
      </div>

      <div className="mt-6 lg:pb-8">
        <TitleSubtitle title="Quality" subtitle="Set resolution, source, codec and related match constraints" />

        <div className="mt-6 grid grid-cols-12 gap-6">
          <MultiSelect name="resolutions" options={RESOLUTION_OPTIONS} label="resolutions" columns={6} creatable={true} />
          <MultiSelect name="sources" options={SOURCES_OPTIONS} label="sources" columns={6} creatable={true} />
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <MultiSelect name="codecs" options={CODECS_OPTIONS} label="codecs" columns={6} creatable={true} />
          <MultiSelect name="containers" options={CONTAINER_OPTIONS} label="containers" columns={6} creatable={true} />
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <MultiSelect name="match_hdr" options={HDR_OPTIONS} label="Match HDR" columns={6} creatable={true} />
          <MultiSelect name="except_hdr" options={HDR_OPTIONS} label="Except HDR" columns={6} creatable={true} />
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <MultiSelect name="match_other" options={OTHER_OPTIONS} label="Match Other" columns={6} creatable={true} />
          <MultiSelect name="except_other" options={OTHER_OPTIONS} label="Except Other" columns={6} creatable={true} />
        </div>
      </div>
    </div>
  );
}

export function Music() {
  return (
    <div>
      <div className="mt-6 grid grid-cols-12 gap-6">
        <TextField name="artists" label="Artists" columns={4} placeholder="eg. Aritst One" />
        <TextField name="albums" label="Albums" columns={4} placeholder="eg. That Album" />
        <TextField name="years" label="Years" columns={4} placeholder="eg. 2018,2019-2021" />
      </div>

      <div className="mt-6 lg:pb-8">
        <TitleSubtitle title="Quality" subtitle="Format, source, log etc." />

        <div className="mt-6 grid grid-cols-12 gap-6">
          <MultiSelect name="formats" options={FORMATS_OPTIONS} label="Format" columns={6} />
          <MultiSelect name="quality" options={QUALITY_MUSIC_OPTIONS} label="Quality" columns={6} />
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <MultiSelect name="media" options={SOURCES_MUSIC_OPTIONS} label="Media" columns={6} />
          <MultiSelect name="match_release_types" options={RELEASE_TYPE_MUSIC_OPTIONS} label="Type" columns={6} />
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <NumberField name="log_score" label="Log score" placeholder="eg. 100" />
        </div>

      </div>

      <div className="space-y-6 sm:space-y-5 divide-y divide-gray-200">
        <div className="pt-6 sm:pt-5">
          <div role="group" aria-labelledby="label-email">
            <div className="sm:grid sm:grid-cols-3 sm:gap-4 sm:items-baseline">
              {/* <div>
                    <div className="text-base font-medium text-gray-900 sm:text-sm sm:text-gray-700" id="label-email">
                      Extra
                    </div>
                  </div> */}
              <div className="mt-4 sm:mt-0 sm:col-span-2">
                <div className="max-w-lg space-y-4">
                  <CheckboxField name="log" label="Log" sublabel="Must include Log" />
                  <CheckboxField name="cue" label="Cue" sublabel="Must include Cue"/>
                  <CheckboxField name="perfect_flac" label="Perfect FLAC" sublabel="Override all options about quality, source, format, and cue/log/log score"/>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export function Advanced() {
  return (
    <div>
      <CollapsableSection title="Releases" subtitle="Match only certain release names and/or ignore other release names">
        <TextField name="match_releases" label="Match releases" columns={6} placeholder="eg. *some?movie*,*some?show*s01*" />
        <TextField name="except_releases" label="Except releases" columns={6} placeholder="" />
        <div className="col-span-6">
          <SwitchGroup name="use_regex" label="Use Regex" />
        </div>
      </CollapsableSection>

      <CollapsableSection title="Groups" subtitle="Match only certain groups and/or ignore other groups">
        <TextField name="match_release_groups" label="Match release groups" columns={6} placeholder="eg. group1,group2" />
        <TextField name="except_release_groups" label="Except release groups" columns={6} placeholder="eg. badgroup1,badgroup2" />
      </CollapsableSection>

      <CollapsableSection title="Categories and tags" subtitle="Match or ignore categories or tags">
        <TextField name="match_categories" label="Match categories" columns={6} placeholder="eg. *category*,category1" />
        <TextField name="except_categories" label="Except categories" columns={6} placeholder="eg. *category*" />

        <TextField name="tags" label="Match tags" columns={6} placeholder="eg. tag1,tag2" />
        <TextField name="except_tags" label="Except tags" columns={6} placeholder="eg. tag1,tag2" />
      </CollapsableSection>

      <CollapsableSection title="Uploaders" subtitle="Match or ignore uploaders">
        <TextField name="match_uploaders" label="Match uploaders" columns={6} placeholder="eg. uploader1" />
        <TextField name="except_uploaders" label="Except uploaders" columns={6} placeholder="eg. anonymous" />
      </CollapsableSection>

      <CollapsableSection title="Origins" subtitle="Match Internals, scene, p2p etc if announced">
        <MultiSelect name="origins" options={ORIGIN_OPTIONS} label="Origins" columns={6} />
      </CollapsableSection>

      <CollapsableSection title="Freeleech" subtitle="Match only freeleech and freeleech percent">
        <div className="col-span-6">
          <SwitchGroup name="freeleech" label="Freeleech" />
        </div>

        <TextField name="freeleech_percent" label="Freeleech percent" columns={6} />
      </CollapsableSection>
    </div>
  );
}

interface CollapsableSectionProps {
    title: string;
    subtitle: string;
    children: React.ReactNode;
}

function CollapsableSection({ title, subtitle, children }: CollapsableSectionProps) {
  const [isOpen, toggleOpen] = useToggle(false);

  return (
    <div className="mt-6 lg:pb-6 border-b border-gray-200 dark:border-gray-700">
      <div className="flex justify-between items-center cursor-pointer" onClick={toggleOpen}>
        <div className="-ml-2 -mt-2 flex flex-wrap items-baseline">
          <h3 className="ml-2 mt-2 text-lg leading-6 font-medium text-gray-900 dark:text-gray-200">{title}</h3>
          <p className="ml-2 mt-1 text-sm text-gray-500 dark:text-gray-400 truncate">{subtitle}</p>
        </div>
        <div className="mt-3 sm:mt-0 sm:ml-4">
          <button
            type="button"
            className="inline-flex items-center px-4 py-2 border-transparent text-sm font-medium text-white"
          >
            {isOpen ? <ChevronDownIcon className="h-6 w-6 text-gray-500" aria-hidden="true" /> : <ChevronRightIcon className="h-6 w-6 text-gray-500" aria-hidden="true" />}
          </button>
        </div>
      </div>
      {isOpen && (
        <div className="mt-6 grid grid-cols-12 gap-6">
          {children}
        </div>
      )}
    </div>
  );
}

interface FilterActionsProps {
    filter: Filter;
    values: FormikValues;
}

export function FilterActions({ filter, values }: FilterActionsProps) {
  const { data } = useQuery(
    ["filters", "download_clients"],
    () => APIClient.download_clients.getAll(),
    { refetchOnWindowFocus: false }
  );

  const newAction = {
    name: "new action",
    enabled: true,
    type: "TEST",
    watch_folder: "",
    exec_cmd: "",
    exec_args: "",
    category: "",
    tags: "",
    label: "",
    save_path: "",
    paused: false,
    ignore_rules: false,
    limit_upload_speed: 0,
    limit_download_speed: 0,
    limit_ratio: 0,
    limit_seed_time: 0,
    reannounce_skip: false,
    reannounce_delete: false,
    reannounce_interval: 7,
    reannounce_max_attempts: 25,
    filter_id: filter.id,
    webhook_host: "",
    webhook_type: "",
    webhook_method: "",
    webhook_data: "",
    webhook_headers: []
    //   client_id: 0,
  };

  return (
    <div className="mt-10">
      <FieldArray name="actions">
        {({ remove, push }) => (
          <Fragment>
            <div className="-ml-4 -mt-4 mb-6 flex justify-between items-center flex-wrap sm:flex-nowrap">
              <div className="ml-4 mt-4">
                <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-gray-200">Actions</h3>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                                    Add to download clients or run custom commands.
                </p>
              </div>
              <div className="ml-4 mt-4 flex-shrink-0">
                <button
                  type="button"
                  className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                  onClick={() => push(newAction)}
                >
                                    Add new
                </button>
              </div>
            </div>

            <div className="light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
              {values.actions.length > 0 ?
                <ul className="divide-y divide-gray-200 dark:divide-gray-700">
                  {values.actions.map((action: Action, index: number) => (
                    <FilterActionsItem action={action} clients={data ?? []} idx={index} remove={remove} key={index} />
                  ))}
                </ul>
                : <EmptyListState text="No actions yet!" />
              }
            </div>
          </Fragment>
        )}
      </FieldArray>
    </div>
  );
}

interface FilterActionsItemProps {
    action: Action;
    clients: DownloadClient[];
    idx: number;
    remove: <T>(index: number) => T | undefined;
}

function FilterActionsItem({ action, clients, idx, remove }: FilterActionsItemProps) {
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);
  const [edit, toggleEdit] = useToggle(false);

  const cancelButtonRef = useRef(null);

  const TypeForm = (actionType: ActionType) => {
    switch (actionType) {
    case "TEST":
      return (
        <AlertWarning
          title="Notice"
          text="The test action does nothing except to show if the filter works."
        />
      );
    case "EXEC":
      return (
        <div>
          <div className="mt-6 grid grid-cols-12 gap-6">
            <TextField
              name={`actions.${idx}.exec_cmd`}
              label="Command"
              columns={6}
              placeholder="Path to program eg. /bin/test"
            />
            <TextField
              name={`actions.${idx}.exec_args`}
              label="Arguments"
              columns={6}
              placeholder="Arguments eg. --test"
            />
          </div>
        </div>
      );
    case "WATCH_FOLDER":
      return (
        <div className="mt-6 grid grid-cols-12 gap-6">
          <TextField
            name={`actions.${idx}.watch_folder`}
            label="Watch folder"
            columns={6}
            placeholder="Watch directory eg. /home/user/rwatch"
          />
        </div>
      );
    case "WEBHOOK":
      return (
        <div className="mt-6 grid grid-cols-12 gap-6">
          <TextField
            name={`actions.${idx}.webhook_host`}
            label="Host"
            columns={6}
            placeholder="Host eg. http://localhost/webhook"
          />
          <TextField
            name={`actions.${idx}.webhook_data`}
            label="Data (json)"
            columns={6}
            placeholder={"Request data: { \"key\": \"value\" }"}
          />
        </div>
      );
    case "QBITTORRENT":
      return (
        <div className="w-full">
          <div className="mt-6 grid grid-cols-12 gap-6">
            <DownloadClientSelect
              name={`actions.${idx}.client_id`}
              action={action}
              clients={clients}
            />

            <div className="col-span-6 sm:col-span-6">
              <TextField
                name={`actions.${idx}.save_path`}
                label="Save path"
                columns={6}
                placeholder="eg. /full/path/to/watch_folder"
              />
            </div>
          </div>

          <div className="mt-6 grid grid-cols-12 gap-6">
            <TextField name={`actions.${idx}.category`} label="Category" columns={6} placeholder="eg. category" />
            <TextField name={`actions.${idx}.tags`} label="Tags" columns={6} placeholder="eg. tag1,tag2" />
          </div>

          <CollapsableSection title="Rules" subtitle="client options">
            <div className="col-span-12">
              <div className="mt-6 grid grid-cols-12 gap-6">
                <NumberField
                  name={`actions.${idx}.limit_download_speed`}
                  label="Limit download speed (KB/s)"
                />
                <NumberField
                  name={`actions.${idx}.limit_upload_speed`}
                  label="Limit upload speed (KB/s)"
                />
              </div>

              <div className="mt-6 grid grid-cols-12 gap-6">
                <NumberField
                  name={`actions.${idx}.limit_ratio`}
                  label="Ratio limit"
                  step={0.5}
                />
                <NumberField
                  name={`actions.${idx}.limit_seed_time`}
                  label="Seed time limit (seconds)"
                />
              </div>
            </div>
            <div className="col-span-6">
              <SwitchGroup
                name={`actions.${idx}.paused`}
                label="Add paused"
                description="Add torrent as paused"
              />
              <SwitchGroup
                name={`actions.${idx}.ignore_rules`}
                label="Ignore client rules"
                description="Download if max active reached"
              />
            </div>
          </CollapsableSection>

          <CollapsableSection title="Advanced" subtitle="Advanced options">
            <div className="col-span-12">
              <div className="mt-6 grid grid-cols-12 gap-6">
                <NumberField
                  name={`actions.${idx}.reannounce_interval`}
                  label="Reannounce interval. Run every X seconds"
                />
                <NumberField
                  name={`actions.${idx}.reannounce_max_attempts`}
                  label="Run reannounce Y times"
                />
              </div>
            </div>
            <div className="col-span-6">
              <SwitchGroup
                name={`actions.${idx}.reannounce_skip`}
                label="Skip reannounce"
                description="If reannounce is not needed, skip"
              />
              <SwitchGroup
                name={`actions.${idx}.reannounce_delete`}
                label="Delete stalled"
                description="Delete stalled torrents after X attempts"
              />
            </div>
          </CollapsableSection>
        </div>
      );
    case "DELUGE_V1":
    case "DELUGE_V2":
      return (
        <div>
          <div className="mt-6 grid grid-cols-12 gap-6">
            <DownloadClientSelect
              name={`actions.${idx}.client_id`}
              action={action}
              clients={clients}
            />

            <div className="col-span-12 sm:col-span-6">
              <TextField
                name={`actions.${idx}.save_path`}
                label="Save path"
                columns={6}
              />
            </div>
          </div>

          <div className="mt-6 col-span-12 sm:col-span-6">
            <TextField
              name={`actions.${idx}.label`}
              label="Label"
              columns={6}
            />
          </div>

          <div className="mt-6 grid grid-cols-12 gap-6">
            <NumberField
              name={`actions.${idx}.limit_download_speed`}
              label="Limit download speed (KB/s)"
            />
            <NumberField
              name={`actions.${idx}.limit_upload_speed`}
              label="Limit upload speed (KB/s)"
            />
          </div>

          <div className="mt-6 grid grid-cols-12 gap-6">
            <div className="col-span-6">
              <SwitchGroup
                name={`actions.${idx}.paused`}
                label="Add paused"
              />
            </div>
          </div>
        </div>
      );
      case "TRANSMISSION":
        return (
          <div>
            <div className="mt-6 grid grid-cols-12 gap-6">
              <DownloadClientSelect
                name={`actions.${idx}.client_id`}
                action={action}
                clients={clients}
              />

              <div className="col-span-12 sm:col-span-6">
                <TextField
                  name={`actions.${idx}.save_path`}
                  label="Save path"
                  columns={6}
                />
              </div>
            </div>

            <div className="mt-6 grid grid-cols-12 gap-6">
              <div className="col-span-6">
                <SwitchGroup
                  name={`actions.${idx}.paused`}
                  label="Add paused"
                />
              </div>
            </div>
          </div>
        );
    case "RADARR":
    case "SONARR":
    case "LIDARR":
    case "WHISPARR":
      return (
        <div className="mt-6 grid grid-cols-12 gap-6">
          <DownloadClientSelect
            name={`actions.${idx}.client_id`}
            action={action}
            clients={clients}
          />
        </div>
      );

    default:
      return null;
    }
  };

  return (
    <li>
      <div
        className={classNames(
          idx % 2 === 0 ? "bg-white dark:bg-gray-800" : "bg-gray-50 dark:bg-gray-700",
          "flex items-center sm:px-6 hover:bg-gray-50 dark:hover:bg-gray-600"
        )}
      >
        <Field name={`actions.${idx}.enabled`} type="checkbox">
          {({
            field,
            form: { setFieldValue }
          }: FieldProps) => (
            <SwitchBasic
              {...field}
              type="button"
              value={field.value}
              checked={field.checked ?? false}
              onChange={(value: boolean) => {
                setFieldValue(field?.name ?? "", value);
              }}
              className={classNames(
                field.value ? "bg-teal-500 dark:bg-blue-500" : "bg-gray-200 dark:bg-gray-600",
                "relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              )}
            >
              <span className="sr-only">toggle enabled</span>
              <span
                aria-hidden="true"
                className={classNames(
                  field.value ? "translate-x-5" : "translate-x-0",
                  "inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
                )}
              />
            </SwitchBasic>
          )}
        </Field>

        <button className="px-4 py-4 w-full flex" type="button" onClick={toggleEdit}>
          <div className="min-w-0 flex-1 sm:flex sm:items-center sm:justify-between">
            <div className="truncate">
              <div className="flex text-sm">
                <p className="ml-4 font-medium text-indigo-600 dark:text-gray-100 truncate">
                  {action.name}
                </p>
              </div>
            </div>
            <div className="mt-4 flex-shrink-0 sm:mt-0 sm:ml-5">
              <div className="flex overflow-hidden -space-x-1">
                <span className="text-sm font-normal text-gray-500 dark:text-gray-400">
                  {ActionTypeNameMap[action.type]}
                </span>
              </div>
            </div>
          </div>
          <div className="ml-5 flex-shrink-0">
            <ChevronRightIcon
              className="h-5 w-5 text-gray-400"
              aria-hidden="true"
            />
          </div>
        </button>

      </div>
      {edit && (
        <div className="px-4 py-4 flex items-center sm:px-6 border dark:border-gray-600">
          <Transition.Root show={deleteModalIsOpen} as={Fragment}>
            <Dialog
              as="div"
              static
              className="fixed inset-0 overflow-y-auto"
              initialFocus={cancelButtonRef}
              open={deleteModalIsOpen}
              onClose={toggleDeleteModal}
            >
              <DeleteModal
                isOpen={deleteModalIsOpen}
                buttonRef={cancelButtonRef}
                toggle={toggleDeleteModal}
                deleteAction={() => remove(idx)}
                title="Remove filter action"
                text="Are you sure you want to remove this action? This action cannot be undone."
              />
            </Dialog>
          </Transition.Root>

          <div className="w-full">

            <div className="mt-6 grid grid-cols-12 gap-6">
              <Select
                name={`actions.${idx}.type`}
                label="Type"
                optionDefaultText="Select type"
                options={ActionTypeOptions}
              />

              <TextField name={`actions.${idx}.name`} label="Name" columns={6} />
            </div>

            {TypeForm(action.type)}

            <div className="pt-6 divide-y divide-gray-200">
              <div className="mt-4 pt-4 flex justify-between">
                <button
                  type="button"
                  className="inline-flex items-center justify-center py-2 border border-transparent font-medium rounded-md text-red-700 dark:text-red-500 hover:text-red-500 dark:hover:text-red-400 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
                  onClick={toggleDeleteModal}
                >
                  Remove
                </button>

                <div>
                  <button
                    type="button"
                    className="light:bg-white light:border light:border-gray-300 rounded-md shadow-sm py-2 px-4 inline-flex justify-center text-sm font-medium text-gray-700 dark:text-gray-500 light:hover:bg-gray-50 dark:hover:text-gray-300 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                    onClick={toggleEdit}
                  >
                    Close
                  </button>
                </div>
              </div>
            </div>

          </div>
        </div>
      )}
    </li>
  );
}
