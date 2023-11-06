/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { ReactNode, Suspense, useEffect, useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { NavLink, Route, Routes, useLocation, useNavigate, useParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import { Form, Formik, FormikValues, useFormikContext } from "formik";
import { z } from "zod";
import { toFormikValidationSchema } from "zod-formik-adapter";
import { ChevronDownIcon, ChevronRightIcon } from "@heroicons/react/24/solid";

import {
  CODECS_OPTIONS,
  CONTAINER_OPTIONS,
  downloadsPerUnitOptions,
  FORMATS_OPTIONS,
  HDR_OPTIONS,
  LANGUAGE_OPTIONS,
  ORIGIN_OPTIONS,
  OTHER_OPTIONS,
  QUALITY_MUSIC_OPTIONS,
  RELEASE_TYPE_MUSIC_OPTIONS,
  RESOLUTION_OPTIONS,
  SOURCES_MUSIC_OPTIONS,
  SOURCES_OPTIONS,
  tagsMatchLogicOptions
} from "@app/domain/constants";
import { APIClient } from "@api/APIClient";
import { useToggle } from "@hooks/hooks";
import { classNames } from "@utils";

import {
  CheckboxField,
  IndexerMultiSelect,
  MultiSelect,
  NumberField,
  RegexField,
  Select,
  SwitchGroup,
  TextField
} from "@components/inputs";
import DEBUG from "@components/debug";
import Toast from "@components/notifications/Toast";
import { DeleteModal } from "@components/modals";
import { TitleSubtitle } from "@components/headings";
import { RegexTextAreaField, TextAreaAutoResize } from "@components/inputs/input";
import { FilterActions } from "./Action";
import { filterKeys } from "./List";
import { External } from "@screens/filters/External";
import { SectionLoader } from "@components/SectionLoader";
import { DocsLink } from "@components/ExternalLink";

interface tabType {
  name: string;
  href: string;
}

const tabs: tabType[] = [
  { name: "General", href: "" },
  { name: "Movies and TV", href: "movies-tv" },
  { name: "Music", href: "music" },
  { name: "Advanced", href: "advanced" },
  { name: "External", href: "external" },
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
        "hover:text-blue-600 dark:hover:text-white hover:border-blue-600 dark:hover:border-blue-500 whitespace-nowrap py-4 px-1 font-medium text-sm",
        isActive ? "border-b-2 border-blue-600 dark:border-blue-500 text-blue-600 dark:text-white" : "text-gray-500"
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
  isLoading: boolean;
}

const FormButtonsGroup = ({ values, deleteAction, reset, isLoading }: FormButtonsGroupProps) => {
  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  const cancelModalButtonRef = useRef(null);

  return (
    <div className="mt-12 border-t border-gray-200 dark:border-gray-700">
      <DeleteModal
        isOpen={deleteModalIsOpen}
        isLoading={isLoading}
        toggle={toggleDeleteModal}
        buttonRef={cancelModalButtonRef}
        deleteAction={deleteAction}
        title={`Remove filter: ${values.name}`}
        text="Are you sure you want to remove this filter? This action cannot be undone."
      />

      <div className="mt-4 pt-4 flex justify-between">
        <button
          type="button"
          className="inline-flex items-center justify-center px-4 py-2 rounded-md sm:text-sm bg-red-700 dark:bg-red-900 hover:dark:bg-red-700 hover:bg-red-800 text-white focus:outline-none"
          onClick={toggleDeleteModal}
        >
          Remove
        </button>

        <div>
          {/* {dirty && <span className="mr-4 text-sm text-gray-500">Unsaved changes..</span>} */}
          <button
            type="button"
            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
            onClick={(e) => {
              e.preventDefault();
              reset();

              toast.custom((t) => <Toast type="success" body="Reset all filter values." t={t} />);
            }}
          >
            Reset form values
          </button>
          <button
            type="submit"
            className="ml-4 relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
};

const FormErrorNotification = () => {
  const { isValid, isValidating, isSubmitting, errors } = useFormikContext();

  useEffect(() => {
    if (!isValid && !isValidating && isSubmitting) {
      console.log("validation errors: ", errors);
      toast.custom((t) => <Toast type="error" body={`Validation error. Check fields: ${Object.keys(errors)}`} t={t} />);
    }
  }, [isSubmitting, isValid, isValidating]);

  return null;
};

const allowedClientType = ["QBITTORRENT", "DELUGE_V1", "DELUGE_V2", "RTORRENT", "TRANSMISSION", "PORLA", "RADARR", "SONARR", "LIDARR", "WHISPARR", "READARR", "SABNZBD"];

const actionSchema = z.object({
  enabled: z.boolean(),
  name: z.string(),
  type: z.enum(["QBITTORRENT", "DELUGE_V1", "DELUGE_V2", "RTORRENT", "TRANSMISSION", "PORLA", "RADARR", "SONARR", "LIDARR", "WHISPARR", "READARR", "SABNZBD", "TEST", "EXEC", "WATCH_FOLDER", "WEBHOOK"]),
  client_id: z.number().optional(),
  exec_cmd: z.string().optional(),
  exec_args: z.string().optional(),
  watch_folder: z.string().optional(),
  category: z.string().optional(),
  tags: z.string().optional(),
  label: z.string().optional(),
  save_path: z.string().optional(),
  paused: z.boolean().optional(),
  ignore_rules: z.boolean().optional(),
  limit_upload_speed: z.number().optional(),
  limit_download_speed: z.number().optional(),
  limit_ratio: z.number().optional(),
  limit_seed_time: z.number().optional(),
  reannounce_skip: z.boolean().optional(),
  reannounce_delete: z.boolean().optional(),
  reannounce_interval: z.number().optional(),
  reannounce_max_attempts: z.number().optional(),
  webhook_host: z.string().optional(),
  webhook_type: z.string().optional(),
  webhook_method: z.string().optional(),
  webhook_data: z.string().optional()
}).superRefine((value, ctx) => {
  if (allowedClientType.includes(value.type)) {
    if (value.client_id === 0) {
      ctx.addIssue({
        message: "Must select client",
        code: z.ZodIssueCode.custom,
        path: ["client_id"]
      });
    }
  }
});

const externalFilterSchema = z.object({
  enabled: z.boolean(),
  index: z.number(),
  name: z.string(),
  type: z.enum(["EXEC", "WEBHOOK"]),
  exec_cmd: z.string().optional(),
  exec_args: z.string().optional(),
  exec_expect_status: z.number().optional(),
  webhook_host: z.string().optional(),
  webhook_type: z.string().optional(),
  webhook_method: z.string().optional(),
  webhook_data: z.string().optional(),
  webhook_expect_status: z.number().optional(),
  webhook_retry_status: z.string().optional(),
  webhook_retry_attempts: z.number().optional(),
  webhook_retry_delay_seconds: z.number().optional(),
  webhook_retry_max_jitter_seconds: z.number().optional()
});

const indexerSchema = z.object({
  id: z.number(),
  name: z.string().optional()
});

// Define the schema for the entire object
const schema = z.object({
  name: z.string(),
  indexers: z.array(indexerSchema).min(1, { message: "Must select at least one indexer" }),
  actions: z.array(actionSchema),
  external: z.array(externalFilterSchema)
});

export function FilterDetails() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { filterId } = useParams<{ filterId: string }>();

  if (filterId === "0" || undefined) {
    navigate("/filters");
  }

  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
  const id = parseInt(filterId!);

  const { isLoading, data: filter } = useQuery({
    queryKey: filterKeys.detail(id),
    queryFn: ({ queryKey }) => APIClient.filters.getByID(queryKey[2]),
    refetchOnWindowFocus: false,
    onError: () => {
      navigate("/filters");
    }
  });

  const updateMutation = useMutation({
    mutationFn: (filter: Filter) => APIClient.filters.update(filter),
    onSuccess: (newFilter, variables) => {
      queryClient.setQueryData(filterKeys.detail(variables.id), newFilter);

      queryClient.setQueryData<Filter[]>(filterKeys.lists(), (previous) => {
        if (previous) {
          return previous.map((filter: Filter) => (filter.id === variables.id ? newFilter : filter));
        }
      });

      toast.custom((t) => (
        <Toast type="success" body={`${newFilter.name} was updated successfully`} t={t} />
      ));
    }
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => APIClient.filters.delete(id),
    onSuccess: () => {
      // Invalidate filters just in case, most likely not necessary but can't hurt.
      queryClient.invalidateQueries({ queryKey: filterKeys.lists() });
      queryClient.invalidateQueries({ queryKey: filterKeys.detail(id) });

      toast.custom((t) => (
        <Toast type="success" body={`${filter?.name} was deleted`} t={t} />
      ));

      // redirect
      navigate("/filters");
    }
  });

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
      } else {
        a.webhook_method = "";
        a.webhook_type = "";
      }
    });

    updateMutation.mutate(data);
  };

  const deleteAction = () => {
    deleteMutation.mutate(filter.id);
  };

  return (
    <main>
      <div className="my-6 max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center">
        <h1 className="text-3xl font-bold text-black dark:text-white">
          <NavLink to="/filters">
            Filters
          </NavLink>
        </h1>
        <ChevronRightIcon className="h-6 w-6 text-gray-500" aria-hidden="true" />
        <h1 className="text-3xl font-bold text-black dark:text-white truncate" title={filter.name}>{filter.name}</h1>
      </div>
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
                enabled: filter.enabled,
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
                smart_episode: filter.smart_episode,
                match_releases: filter.match_releases,
                except_releases: filter.except_releases,
                match_release_groups: filter.match_release_groups,
                except_release_groups: filter.except_release_groups,
                match_release_tags: filter.match_release_tags,
                except_release_tags: filter.except_release_tags,
                use_regex_release_tags: filter.use_regex_release_tags,
                match_description: filter.match_description,
                except_description: filter.except_description,
                use_regex_description: filter.use_regex_description,
                match_categories: filter.match_categories,
                except_categories: filter.except_categories,
                tags: filter.tags,
                except_tags: filter.except_tags,
                tags_match_logic: filter.tags_match_logic,
                except_tags_match_logic: filter.except_tags_match_logic,
                match_uploaders: filter.match_uploaders,
                except_uploaders: filter.except_uploaders,
                match_language: filter.match_language || [],
                except_language: filter.except_language || [],
                freeleech: filter.freeleech,
                freeleech_percent: filter.freeleech_percent,
                formats: filter.formats || [],
                quality: filter.quality || [],
                media: filter.media || [],
                match_release_types: filter.match_release_types || [],
                log_score: filter.log_score,
                record_label: filter.record_label,
                log: filter.log,
                cue: filter.cue,
                perfect_flac: filter.perfect_flac,
                artists: filter.artists,
                albums: filter.albums,
                origins: filter.origins || [],
                except_origins: filter.except_origins || [],
                indexers: filter.indexers || [],
                actions: filter.actions || [],
                external: filter.external || []
              } as Filter}
              onSubmit={handleSubmit}
              enableReinitialize={true}
              validationSchema={toFormikValidationSchema(schema)}
            >
              {({ values, dirty, resetForm }) => (
                <Form>
                  <FormErrorNotification />
                  <Suspense fallback={<SectionLoader $size="large" />}>
                    <Routes>
                      <Route index element={<General />} />
                      <Route path="movies-tv" element={<MoviesTv />} />
                      <Route path="music" element={<Music values={values} />} />
                      <Route path="advanced" element={<Advanced values={values} />} />
                      <Route path="external" element={<External />} />
                      <Route path="actions" element={<FilterActions filter={filter} values={values} />} />
                    </Routes>
                  </Suspense>
                  <FormButtonsGroup
                    values={values}
                    deleteAction={deleteAction}
                    dirty={dirty}
                    reset={resetForm}
                    isLoading={isLoading}
                  />
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
  const { isLoading, data: indexers } = useQuery({
    queryKey: ["filters", "indexer_list"],
    queryFn: APIClient.indexers.getOptions,
    refetchOnWindowFocus: false
  });

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
        <TitleSubtitle title="Rules" subtitle="Specify rules on how torrents should be handled/selected." />

        <div className="mt-6 grid grid-cols-12 gap-6">
          <TextField
            name="min_size"
            label="Min size"
            columns={6}
            placeholder="eg. 100MiB, 80GB"
            tooltip={
              <div>
                <p>Supports units such as MB, MiB, GB, etc.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <TextField
            name="max_size"
            label="Max size"
            columns={6}
            placeholder="eg. 100MiB, 80GB"
            tooltip={
              <div>
                <p>Supports units such as MB, MiB, GB, etc.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <NumberField
            name="delay"
            label="Delay"
            placeholder="Number of seconds to delay actions"
            tooltip={
              <div>
                <p>Number of seconds to wait before running actions.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <NumberField
            name="priority"
            label="Priority"
            placeholder="Higher number = higher priority"
            tooltip={
              <div>
                <p>Filters are checked in order of priority. Higher number = higher priority.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <NumberField
            name="max_downloads"
            label="Max downloads"
            placeholder="Takes any number (0 is infinite)"
            tooltip={
              <div>
                <p>Number of max downloads as specified by the respective unit.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <Select
            name="max_downloads_unit"
            label="Max downloads per"
            options={downloadsPerUnitOptions}
            optionDefaultText="Select unit"
            tooltip={
              <div>
                <p>The unit of time for counting the maximum downloads per filter.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
        </div>
      </div>

      <div className="border-t dark:border-gray-700">
        <SwitchGroup name="enabled" label="Enabled" description="Enable or disable this filter." />
      </div>
    </div>
  );
}

export function MoviesTv() {
  return (
    <div>
      <div className="mt-6 grid grid-cols-12 gap-6">
        <TextAreaAutoResize
          name="shows"
          label="Movies / Shows"
          columns={8}
          placeholder="eg. Movie,Show 1,Show?2"
          tooltip={
            <div>
              <p>You can use basic filtering like wildcards <code>*</code> or replace single characters with <code>?</code></p>
              <DocsLink href="https://autobrr.com/filters#tvmovies" />
            </div>
          }
        />
        <TextField
          name="years"
          label="Years"
          columns={4}
          placeholder="eg. 2018,2019-2021"
          tooltip={
            <div>
              <p>This field takes a range of years and/or comma separated single years.</p>
              <DocsLink href="https://autobrr.com/filters#tvmovies" />
            </div>
          }
        />
      </div>
      <div className="mt-6 lg:pb-8">
        <TitleSubtitle
          title="Seasons and Episodes"
          subtitle="Set season and episode match constraints."
        />

        <div className="mt-6 grid grid-cols-12 gap-6">
          <TextField
            name="seasons"
            label="Seasons"
            columns={8}
            placeholder="eg. 1,3,2-6"
            tooltip={
              <div>
                <p>See docs for information about how to <b>only</b> grab season packs:</p>
                <DocsLink href="https://autobrr.com/filters/examples#only-season-packs" />
              </div>
            }
          />
          <TextField
            name="episodes"
            label="Episodes"
            columns={4}
            placeholder="eg. 2,4,10-20"
            tooltip={
              <div>
                <p>See docs for information about how to <b>only</b> grab episodes:</p>
                <DocsLink href="https://autobrr.com/filters/examples/#skip-season-packs" />
              </div>
            }
          />
        </div>

        <div className="mt-6">
          <CheckboxField
            name="smart_episode"
            label="Smart Episode"
            sublabel="Do not match episodes older than the last one matched."
          />{" "}
          {/*Do not match older or already existing episodes.*/}
        </div>
      </div>

      <div className="mt-6 lg:pb-8">
        <TitleSubtitle
          title="Quality"
          subtitle="Set resolution, source, codec and related match constraints."
        />

        <div className="mt-6 grid grid-cols-12 gap-6">
          <MultiSelect
            name="resolutions"
            options={RESOLUTION_OPTIONS}
            label="resolutions"
            columns={6}
            creatable={true}
            tooltip={
              <div>
                <p>Will match releases which contain any of the selected resolutions.</p>
                <DocsLink href="https://autobrr.com/filters#quality" />
              </div>
            }
          />
          <MultiSelect
            name="sources"
            options={SOURCES_OPTIONS}
            label="sources"
            columns={6}
            creatable={true}
            tooltip={
              <div>
                <p>Will match releases which contain any of the selected sources.</p>
                <DocsLink href="https://autobrr.com/filters#quality" />
              </div>
            }
          />
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <MultiSelect
            name="codecs"
            options={CODECS_OPTIONS}
            label="codecs"
            columns={6}
            creatable={true}
            tooltip={
              <div>
                <p>Will match releases which contain any of the selected codecs.</p>
                <DocsLink href="https://autobrr.com/filters#quality" />
              </div>
            }
          />
          <MultiSelect
            name="containers"
            options={CONTAINER_OPTIONS}
            label="containers"
            columns={6}
            creatable={true}
            tooltip={
              <div>
                <p>Will match releases which contain any of the selected containers.</p>
                <DocsLink href="https://autobrr.com/filters#quality" />
              </div>
            }
          />
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <MultiSelect
            name="match_hdr"
            options={HDR_OPTIONS}
            label="Match HDR"
            columns={6}
            creatable={true}
            tooltip={
              <div>
                <p>Will match releases which contain any of the selected HDR designations.</p>
                <DocsLink href="https://autobrr.com/filters#quality" />
              </div>
            }
          />
          <MultiSelect
            name="except_hdr"
            options={HDR_OPTIONS}
            label="Except HDR"
            columns={6}
            creatable={true}
            tooltip={
              <div>
                <p>Won't match releases which contain any of the selected HDR designations (takes priority over Match HDR).</p>
                <DocsLink href="https://autobrr.com/filters#quality" />
              </div>
            }
          />
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <MultiSelect
            name="match_other"
            options={OTHER_OPTIONS}
            label="Match Other"
            columns={6}
            creatable={true}
            tooltip={
              <div>
                <p>Will match releases which contain any of the selected designations.</p>
                <DocsLink href="https://autobrr.com/filters#quality" />
              </div>
            }
          />
          <MultiSelect
            name="except_other"
            options={OTHER_OPTIONS}
            label="Except Other"
            columns={6}
            creatable={true}
            tooltip={
              <div>
                <p>Won't match releases which contain any of the selected Other designations (takes priority over Match Other).</p>
                <DocsLink href="https://autobrr.com/filters#quality" />
              </div>
            }
          />
        </div>
      </div>
    </div>
  );
}

export function Music({ values }: AdvancedProps) {
  return (
    <div>
      <div className="mt-6 grid grid-cols-6 gap-6">
        <TextAreaAutoResize
          name="artists"
          label="Artists"
          columns={3}
          placeholder="eg. Artist One"
          tooltip={
            <div>
              <p>You can use basic filtering like wildcards <code>*</code> or replace single characters with <code>?</code></p>
              <DocsLink href="https://autobrr.com/filters#music" />
            </div>
          }
        />
        <TextAreaAutoResize
          name="albums"
          label="Albums"
          columns={3}
          placeholder="eg. That Album"
          tooltip={
            <div>
              <p>You can use basic filtering like wildcards <code>*</code> or replace single characters with <code>?</code></p>
              <DocsLink href="https://autobrr.com/filters#music" />
            </div>
          }
        />
        <TextAreaAutoResize
          name="record_label"
          label="Record Labels"
          columns={3}
          placeholder="eg. Interscope"
          tooltip={
            <div>
              <p>Comma separated list of record labels to match against.</p>
              <p>Only OPS and RED are supported for now.</p>
            </div>
          }
        />
        <TextField
          name="years"
          label="Years"
          columns={3}
          placeholder="eg. 2018,2019-2021"
          tooltip={
            <div>
              <p>This field takes a range of years and/or comma separated single years.</p>
              <DocsLink href="https://autobrr.com/filters#music" />
            </div>
          }
        />
      </div>

      <div className="mt-6 lg:pb-8">
        <TitleSubtitle title="Quality" subtitle="Format, source, log etc." />

        <div className="mt-6 grid grid-cols-12 gap-6">
          <MultiSelect
            name="formats"
            options={FORMATS_OPTIONS}
            label="Format"
            columns={6}
            disabled={values.perfect_flac}
            tooltip={
              <div>
                <p>Will only match releases with any of the selected formats. This is overridden by Perfect FLAC.</p>
                <DocsLink href="https://autobrr.com/filters#quality-1" />
              </div>
            }
          />
          <MultiSelect
            name="quality"
            options={QUALITY_MUSIC_OPTIONS}
            label="Quality"
            columns={6}
            disabled={values.perfect_flac}
            tooltip={
              <div>
                <p>Will only match releases with any of the selected qualities. This is overridden by Perfect FLAC.</p>
                <DocsLink href="https://autobrr.com/filters#quality-1" />
              </div>
            }
          />
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <MultiSelect
            name="media"
            options={SOURCES_MUSIC_OPTIONS}
            label="Media"
            columns={6}
            disabled={values.perfect_flac}
            tooltip={
              <div>
                <p>Will only match releases with any of the selected sources. This is overridden by Perfect FLAC.</p>
                <DocsLink href="https://autobrr.com/filters#quality-1" />
              </div>
            }
          />
          <MultiSelect
            name="match_release_types"
            options={RELEASE_TYPE_MUSIC_OPTIONS}
            label="Type"
            columns={6}
            tooltip={
              <div>
                <p>Will only match releases with any of the selected types.</p>
                <DocsLink href="https://autobrr.com/filters#quality-1" />
              </div>
            }
          />
        </div>

        <div className="mt-6 grid grid-cols-12 gap-6">
          <NumberField
            name="log_score"
            label="Log score"
            placeholder="eg. 100"
            min={0}
            max={100}
            disabled={values.perfect_flac}
            tooltip={
              <div>
                <p>Log scores go from 0 to 100. This is overridden by Perfect FLAC.</p>
                <DocsLink href="https://autobrr.com/filters#quality-1" />
              </div>
            }
          />
        </div>
      </div>
      <div className="sm:grid sm:grid-cols-3 sm:gap-4 sm:items-baseline">
        <div className="mt-4 sm:mt-0 sm:col-span-2">
          <div className="max-w-lg space-y-4">
            <CheckboxField
              name="log"
              label="Log"
              sublabel="Must include Log."
              disabled={values.perfect_flac}
            />
            <CheckboxField
              name="cue"
              label="Cue"
              sublabel="Must include Cue."
              disabled={values.perfect_flac}
            />
            <CheckboxField
              name="perfect_flac"
              label="Perfect FLAC"
              sublabel="Override all options about quality, source, format, and cue/log/log score."
              tooltip={
                <div>
                  <p>Override all options about quality, source, format, and cue/log/log score.</p>
                  <DocsLink href="https://autobrr.com/filters#quality-1" />
                </div>
              }
            />
          </div>
        </div>
      </div>
    </div>
  );
}

interface AdvancedProps {
  values: FormikValues;
}

export function Advanced({ values }: AdvancedProps) {
  return (
    <div>
      <CollapsableSection
        defaultOpen
        title="Releases"
        subtitle="Match only certain release names and/or ignore other release names."
      >
        <div className="grid col-span-12 gap-6">
          <WarningAlert text="autobrr has extensive filtering built-in - only use this if nothing else works. If you need help, please ask." />

          <RegexTextAreaField
            name="match_releases"
            label="Match releases"
            useRegex={values.use_regex}
            columns={6}
            placeholder="eg. *some?movie*,*some?show*s01*"
            tooltip={
              <div>
                <p>This field has full regex support (Golang flavour).</p>
                <DocsLink href="https://autobrr.com/filters#advanced" />
                <br />
                <br />
                <p>Remember to tick <b>Use Regex</b> below if using more than <code>*</code> and <code>?</code>.</p>
              </div>
            }
          />
          <RegexTextAreaField
            name="except_releases"
            label="Except releases"
            useRegex={values.use_regex}
            columns={6}
            placeholder="eg. *bad?movie*,*bad?show*s03*"
            tooltip={
              <div>
                <p>This field has full regex support (Golang flavour).</p>
                <DocsLink href="https://autobrr.com/filters#advanced" />
                <br />
                <br />
                <p>Remember to tick <b>Use Regex</b> below if using more than <code>*</code> and <code>?</code>.</p>
              </div>
            }
          />

          {values.match_releases ? (
            <WarningAlert
              alert="Ask yourself:"
              text={
                <>
                  Do you have a good reason to use <strong>Match releases</strong> instead of one of the other tabs?
                </>
              }
              colors="text-cyan-700 bg-cyan-100 dark:bg-cyan-200 dark:text-cyan-800"
            />
          ) : null}
          {values.except_releases ? (
            <WarningAlert
              alert="Ask yourself:"
              text={
                <>
                  Do you have a good reason to use <strong>Except releases</strong> instead of one of the other tabs?
                </>
              }
              colors="text-fuchsia-700 bg-fuchsia-100 dark:bg-fuchsia-200 dark:text-fuchsia-800"
            />
          ) : null}
          <div className="col-span-6">
            <SwitchGroup name="use_regex" label="Use Regex" />
          </div>
        </div>
      </CollapsableSection>

      <CollapsableSection
        defaultOpen={true}
        title="Groups"
        subtitle="Match only certain groups and/or ignore other groups."
      >
        <TextAreaAutoResize
          name="match_release_groups"
          label="Match release groups"
          columns={6}
          placeholder="eg. group1,group2"
          tooltip={
            <div>
              <p>Comma separated list of release groups to match.</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
        <TextAreaAutoResize
          name="except_release_groups"
          label="Except release groups"
          columns={6}
          placeholder="eg. badgroup1,badgroup2"
          tooltip={
            <div>
              <p>Comma separated list of release groups to ignore (takes priority over Match releases).</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
      </CollapsableSection>

      <CollapsableSection
        defaultOpen={true}
        title="Categories and tags"
        subtitle="Match or ignore categories or tags."
      >
        <TextAreaAutoResize
          name="match_categories"
          label="Match categories"
          columns={6}
          placeholder="eg. *category*,category1"
          tooltip={
            <div>
              <p>Comma separated list of categories to match.</p>
              <DocsLink href="https://autobrr.com/filters/categories" />
            </div>
          }
        />
        <TextAreaAutoResize
          name="except_categories"
          label="Except categories"
          columns={6}
          placeholder="eg. *category*"
          tooltip={
            <div>
              <p>Comma separated list of categories to ignore (takes priority over Match releases).</p>
              <DocsLink href="https://autobrr.com/filters/categories" />
            </div>
          }
        />

        <TextAreaAutoResize
          name="tags"
          label="Match tags"
          columns={4}
          placeholder="eg. tag1,tag2"
          tooltip={
            <div>
              <p>Comma separated list of tags to match.</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
        <Select
          name="tags_match_logic"
          label="Tags logic"
          columns={2}
          options={tagsMatchLogicOptions}
          optionDefaultText="any"
          tooltip={
            <div>
              <p>Logic used to match filter tags.</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
        <TextAreaAutoResize
          name="except_tags"
          label="Except tags"
          columns={4}
          placeholder="eg. tag1,tag2"
          tooltip={
            <div>
              <p>Comma separated list of tags to ignore (takes priority over Match releases).</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
        <Select
          name="except_tags_match_logic"
          label="Except tags logic"
          columns={2}
          options={tagsMatchLogicOptions}
          optionDefaultText="any"
          tooltip={
            <div>
              <p>Logic used to match except tags.</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
      </CollapsableSection>

      <CollapsableSection
        defaultOpen={true}
        title="Uploaders"
        subtitle="Match or ignore uploaders."
      >
        <TextAreaAutoResize
          name="match_uploaders"
          label="Match uploaders"
          columns={6}
          placeholder="eg. uploader1,uploader2"
          tooltip={
            <div>
              <p>Comma separated list of uploaders to match.</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
        <TextAreaAutoResize
          name="except_uploaders"
          label="Except uploaders"
          columns={6}
          placeholder="eg. anonymous1,anonymous2"
          tooltip={
            <div>
              <p>Comma separated list of uploaders to ignore (takes priority over Match releases).
              </p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
            </div>
          }
        />
      </CollapsableSection>

      <CollapsableSection
        defaultOpen={true}
        title="Language"
        subtitle="Match or ignore languages."
      >
        <MultiSelect
          name="match_language"
          options={LANGUAGE_OPTIONS}
          label="Match Language"
          columns={6}
          creatable={true}
        />
        <MultiSelect
          name="except_language"
          options={LANGUAGE_OPTIONS}
          label="Except Language"
          columns={6}
          creatable={true}
        />
      </CollapsableSection>

      <CollapsableSection
        defaultOpen={true}
        title="Origins"
        subtitle="Match Internals, scene, p2p etc. if announced."
      >
        <MultiSelect
          name="origins"
          options={ORIGIN_OPTIONS}
          label="Match Origins"
          columns={6}
          creatable={true}
        />
        <MultiSelect
          name="except_origins"
          options={ORIGIN_OPTIONS}
          label="Except Origins"
          columns={6}
          creatable={true}
        />
      </CollapsableSection>

      <CollapsableSection
        defaultOpen={true}
        title="Release Tags"
        subtitle="This is the non-parsed releaseTags string from the announce."
      >
        <div className="grid col-span-12 gap-6">
          <WarningAlert text="These might not be what you think they are. For advanced users who know how things are parsed." />

          <RegexField
            name="match_release_tags"
            label="Match release tags"
            useRegex={values.use_regex_release_tags}
            columns={6}
            placeholder="eg. *mkv*,*foreign*"
          />
          <RegexField
            name="except_release_tags"
            label="Except release tags"
            useRegex={values.use_regex_release_tags}
            columns={6}
            placeholder="eg. *mkv*,*foreign*"
          />

          <div className="col-span-6">
            <SwitchGroup name="use_regex_release_tags" label="Use Regex" />
          </div>
        </div>
      </CollapsableSection>

      <CollapsableSection
        defaultOpen={true}
        title="Feeds"
        subtitle="These options are only for Feeds such as RSS, Torznab and Newznab"
      >
        {/*<div className="grid col-span-12 gap-6">*/}
        <RegexTextAreaField
          name="match_description"
          label="Match description"
          useRegex={values.use_regex_description}
          columns={6}
          placeholder="eg. *some?movie*,*some?show*s01*"
          tooltip={
            <div>
              <p>This field has full regex support (Golang flavour).</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
              <br />
              <br />
              <p>Remember to tick <b>Use Regex</b> below if using more than <code>*</code> and <code>?</code>.</p>
            </div>
          }
        />
        <RegexTextAreaField
          name="except_description"
          label="Except description"
          useRegex={values.use_regex_description}
          columns={6}
          placeholder="eg. *bad?movie*,*bad?show*s03*"
          tooltip={
            <div>
              <p>This field has full regex support (Golang flavour).</p>
              <DocsLink href="https://autobrr.com/filters#advanced" />
              <br />
              <br />
              <p>Remember to tick <b>Use Regex</b> below if using more than <code>*</code> and <code>?</code>.</p>
            </div>
          }
        />
        {/*</div>*/}

        <div className="col-span-6">
          <SwitchGroup name="use_regex_description" label="Use Regex" />
        </div>
      </CollapsableSection>

      <CollapsableSection
        defaultOpen={true}
        title="Freeleech"
        subtitle="Match only freeleech and freeleech percent."
      >
        <div className="col-span-6">
          <SwitchGroup
            name="freeleech"
            label="Freeleech"
            tooltip={
              <div>
                <p>
                  Freeleech may be announced as a binary true/false value or as
                  a percentage, depending on the indexer. Use either or both,
                  depending on the indexers you use.
                </p>
                <br />
                <p>
                  See who uses what in the documentation:{" "}
                  <DocsLink href="https://autobrr.com/filters/freeleech" />
                </p>
              </div>
            }
          />
        </div>

        <TextField
          name="freeleech_percent"
          label="Freeleech percent"
          disabled={values.freeleech}
          tooltip={
            <div>
              <p>
                Freeleech may be announced as a binary true/false value or as a
                percentage, depending on the indexer. Use either or both,
                depending on the indexers you use.
              </p>
              <br />
              <p>
                See who uses what in the documentation:{" "}
                <DocsLink href="https://autobrr.com/filters/freeleech" />
              </p>
            </div>
          }
          columns={6}
          placeholder="eg. 50,75-100"
        />
      </CollapsableSection>
    </div>
  );
}

interface WarningAlertProps {
  text: string | JSX.Element;
  alert?: string;
  colors?: string;
}

function WarningAlert({ text, alert, colors }: WarningAlertProps) {
  return (
    <div
      className={classNames(
        "col-span-12 flex items-center px-4 py-3 text-md font-medium rounded-lg",
        colors ?? "text-amber-800 bg-amber-100 border border-amber-700 dark:border-none dark:bg-amber-200 dark:text-amber-800"
      )}
      role="alert">
      <svg aria-hidden="true" className="flex-shrink-0 inline w-5 h-5 mr-3" fill="currentColor"
        viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
        <path fillRule="evenodd"
          d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
          clipRule="evenodd"></path>
      </svg>
      <span className="sr-only">Info</span>
      <div>
        <span className="font-extrabold">{alert ?? "Warning!"}</span>
        {" "}{text}
      </div>
    </div>
  );
}

interface CollapsableSectionProps {
  title: string;
  subtitle: string;
  children: ReactNode;
  defaultOpen?: boolean;
}

export function CollapsableSection({ title, subtitle, children, defaultOpen }: CollapsableSectionProps) {
  const [isOpen, toggleOpen] = useToggle(defaultOpen ?? false);

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
            {isOpen ? <ChevronDownIcon className="-mr-4 h-6 w-6 text-gray-500" aria-hidden="true" /> : <ChevronRightIcon className="-mr-4 h-6 w-6 text-gray-500" aria-hidden="true" />}
          </button>
        </div>
      </div>
      {isOpen && (
        <div className="mt-2 grid grid-cols-12 gap-6">
          {children}
        </div>
      )}
    </div>
  );
}

