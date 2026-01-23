/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useRef } from "react";
import { useMutation, useSuspenseQuery } from "@tanstack/react-query";
import { getRouteApi, Link, Outlet, useNavigate } from "@tanstack/react-router";
import { Form, Formik, useFormikContext } from "formik";
import type { FormikErrors, FormikValues } from "formik";
import { z } from "zod";
import { toFormikValidationSchema } from "zod-formik-adapter";
import { ChevronRightIcon } from "@heroicons/react/24/solid";

import { APIClient } from "@api/APIClient";
import { FilterByIdQueryOptions } from "@api/queries";
import { FilterKeys } from "@api/query_keys";
import { useToggle } from "@hooks/hooks";
import { classNames } from "@utils";
import { DOWNLOAD_CLIENTS, ExternalFilterOnErrorValues } from "@domain/constants";

import { DEBUG } from "@components/debug";
import { toast } from "@components/hot-toast";
import Toast from "@components/notifications/Toast";
import { DeleteModal } from "@components/modals";


interface tabType {
  name: string;
  href: string;
  exact?: boolean;
  newFeature?: boolean;
}

const tabs: tabType[] = [
  { name: "General", href: "/filters/$filterId", exact: true },
  { name: "Movies and TV", href: "/filters/$filterId/movies-tv" },
  { name: "Music", href: "/filters/$filterId/music" },
  { name: "Advanced", href: "/filters/$filterId/advanced" },
  { name: "External", href: "/filters/$filterId/external" },
  { name: "Actions", href: "/filters/$filterId/actions" },
  { name: "Notifications", href: "/filters/$filterId/notifications", newFeature: true }
];

export interface NavLinkProps {
  item: tabType;
}

function TabNavLink({ item }: NavLinkProps) {
  // const location = useLocation();
  // const splitLocation = location.pathname.split("/");

  // we need to clean the / if it's a base root path
  return (
    <Link
      to={item.href}
      activeOptions={{ exact: item.exact }}
      search={{}}
      params={{}}
      // aria-current={splitLocation[2] === item.href ? "page" : undefined}
      // className="transition border-b-2 whitespace-nowrap py-4 duration-3000 px-1 font-medium text-sm first:rounded-tl-lg last:rounded-tr-lg"
    >
      {({ isActive }) => {
        return (
          <span
            className={
            classNames(
              "border-b-2 whitespace-nowrap py-4 px-1 first:rounded-tl-lg last:rounded-tr-lg",
              isActive
                ? "border-blue-600 dark:border-blue-500"
                : " border-transparent"
            )
          }>
            <span
              className={
                classNames(
                  "font-medium text-sm",
                  isActive
                    ? "text-blue-600 dark:text-white "
                    : "text-gray-550 hover:text-blue-500 dark:hover:text-white border-transparent"
                )
              }
            >
              {item.name}
            </span>
            {item.newFeature &&
              <span className="ml-2 inline-flex items-center rounded-md bg-green-100 px-1.5 py-0.5 text-xs font-medium text-green-700 dark:bg-green-400/10 dark:text-green-400">NEW</span>
            }
          </span>
        )
      }}
    </Link>
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
    <>
      <DeleteModal
        isOpen={deleteModalIsOpen}
        isLoading={isLoading}
        toggle={toggleDeleteModal}
        buttonRef={cancelModalButtonRef}
        deleteAction={deleteAction}
        title={`Remove filter: ${values.name}`}
        text="Are you sure you want to remove this filter? This action cannot be undone."
      />

      <div className="px-0.5 mt-8 flex flex-col-reverse sm:flex-row flex-wrap-reverse justify-between">
        <button
          type="button"
          className="flex items-center justify-center px-4 py-2 rounded-md sm:text-sm transition bg-red-700 dark:bg-red-900 dark:hover:bg-red-700 hover:bg-red-800 text-white focus:outline-hidden"
          onClick={toggleDeleteModal}
        >
          Delete Filter
        </button>

        <div className="flex justify-between mb-4 sm:mb-0">
          {/* {dirty && <span className="mr-4 text-sm text-gray-500">Unsaved changes..</span>} */}
          <button
            type="button"
            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 transition rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
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
            className="ml-1 sm:ml-4 flex items-center px-4 py-2 border border-transparent transition shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            Save
          </button>
        </div>
      </div>
    </>
  );
};

const ResolveKV = (obj: unknown, depth: string[] = []) => {
  if (obj === undefined || obj === null) {
    return [];
  }

  if (typeof (obj) !== "object") {
    return [`${depth.join("->")}: ${String(obj)}`];
  }

  const resolved: string[] = [];
  for (const [key, value] of Object.entries(obj)) {
    resolved.push(...ResolveKV(value, [...depth, key]));
  }

  return resolved;
};

const FormatFormikErrorObject = (obj: FormikErrors<unknown>) => "\n" + ResolveKV(obj).join("\n");

const FormErrorNotification = () => {
  const { isValid, isValidating, isSubmitting, errors } = useFormikContext();

  useEffect(() => {
    if (!isValid && !isValidating && isSubmitting) {
      console.log("Formik error object: ", errors);

      const formattedErrors = FormatFormikErrorObject(errors);
      console.log("--> Formatted Errors: ", formattedErrors);

      toast.custom((t) => (
        <Toast
          type="error"
          body={`Validation error${formattedErrors.length > 1 ? "s" : ""}: ${formattedErrors}`}
          t={t}
        />
      ));
    }
  }, [errors, isSubmitting, isValid, isValidating]);

  return null;
};

const actionSchema = z.object({
  enabled: z.boolean(),
  name: z.string(),
  type: z.enum(["TEST", "EXEC", "WATCH_FOLDER", "WEBHOOK", ...DOWNLOAD_CLIENTS]),
  client_id: z.number().optional(),
  exec_cmd: z.string().optional(),
  exec_args: z.string().optional(),
  watch_folder: z.string().optional(),
  category: z.string().optional(),
  tags: z.string().optional(),
  label: z.string().optional(),
  save_path: z.string().optional(),
  download_path: z.string().optional(),
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
  if (DOWNLOAD_CLIENTS.includes(value.type)) {
    if (!value.client_id) {
      ctx.addIssue({
        message: "Must select client",
        code: "custom",
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
  on_error: z.enum([...ExternalFilterOnErrorValues]),
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
}).superRefine((value, ctx) => {
  if (!value.name) {
    ctx.addIssue({
      message: "Must have a name",
      code: "custom",
      path: ["name"]
    });
  }

  if (value.type == "WEBHOOK") {
    if (!value.webhook_method) {
      ctx.addIssue({
        message: "Must select method",
        code: "custom",
        path: ["webhook_method"]
      });
    }
    if (!value.webhook_host) {
      ctx.addIssue({
        message: "Must have webhook host",
        code: "custom",
        path: ["webhook_host"]
      });
    }
    if (!value.webhook_expect_status) {
      ctx.addIssue({
        message: "Must have webhook expect status",
        code: "custom",
        path: ["webhook_expect_status"]
      });
    }
  }

  if (value.type == "EXEC") {
    if (!value.exec_cmd) {
      ctx.addIssue({
        message: "Must have exec cmd",
        code: "custom",
        path: ["exec_cmd"]
      });
    }
  }
});

const indexerSchema = z.object({
  id: z.number(),
  name: z.string().optional()
});

// Define the schema for the entire object
const schema = z.object({
  name: z.string(),
  max_downloads: z.number().optional(),
  max_downloads_unit: z.string().optional(),
  max_downloads_window_type: z.string().optional(),
  max_downloads_interval: z.number().optional(),
  indexers: z.array(indexerSchema).min(1, { message: "Must select at least one indexer" }),
  actions: z.array(actionSchema),
  external: z.array(externalFilterSchema)
}).superRefine((value, ctx) => {
  if (value.max_downloads && value.max_downloads > 0) {
    if (!value.max_downloads_unit) {
      ctx.addIssue({
        message: "Must select Max Downloads Per unit when Max Downloads is greater than 0",
        code: "custom",
        path: ["max_downloads_unit"]
      });
    }
  }
});

export const FilterDetails = () => {
  const navigate = useNavigate();

  const filterGetByIdRoute = getRouteApi("/auth/authenticated-routes/filters/$filterId");
  const { queryClient } =  filterGetByIdRoute.useRouteContext();

  const params = filterGetByIdRoute.useParams()
  const filterQuery = useSuspenseQuery(FilterByIdQueryOptions(params.filterId))
  const filter = filterQuery.data

  const updateMutation = useMutation({
    mutationFn: (filter: Filter) => APIClient.filters.update(filter),
    onSuccess: (newFilter, variables) => {
      queryClient.setQueryData(FilterKeys.detail(variables.id), newFilter);

      queryClient.setQueryData<Filter[]>(FilterKeys.lists(), (previous) => {
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
      queryClient.invalidateQueries({ queryKey: FilterKeys.lists() });
      queryClient.removeQueries({ queryKey: FilterKeys.detail(params.filterId) });

      toast.custom((t) => (
        <Toast type="success" body={`${filter?.name} was deleted`} t={t} />
      ));

      // redirect
      navigate({ to: "/filters" });
    }
  });

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
      <div className="my-6 max-w-(--breakpoint-xl) mx-auto px-4 sm:px-6 lg:px-8 flex items-center text-black dark:text-white">
        <h1 className="text-3xl font-bold">
          <Link to="/filters">
            Filters
          </Link>
        </h1>
        <ChevronRightIcon className="h-6 w-4 shrink-0 sm:shrink sm:h-6 sm:w-6 mx-1" aria-hidden="true" />
        <h1 className="text-3xl font-bold truncate" title={filter.name}>{filter.name}</h1>
      </div>
      <div className="max-w-(--breakpoint-xl) mx-auto pb-12 px-2 sm:px-6 lg:px-8">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-250 dark:border-gray-775">
          <div className="rounded-t-lg bg-gray-125 dark:bg-gray-850 border-b border-gray-200 dark:border-gray-750">
            <nav className="px-4 py-4 -mb-px flex space-x-6 sm:space-x-8 overflow-x-auto">
              {tabs.map((tab) => (
                <TabNavLink key={tab.href} item={tab}  />
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
              announce_types: filter.announce_types || [],
              delay: filter.delay,
              priority: filter.priority,
              max_downloads: filter.max_downloads,
              max_downloads_unit: filter.max_downloads_unit,
              max_downloads_window_type: filter.max_downloads_window_type || 'FIXED',
              max_downloads_interval: filter.max_downloads_interval || 1,
              use_regex: filter.use_regex || false,
              shows: filter.shows,
              years: filter.years,
              months: filter.months,
              days: filter.days,
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
              match_record_labels: filter.match_record_labels,
              except_record_labels: filter.except_record_labels,
              match_language: filter.match_language || [],
              except_language: filter.except_language || [],
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
              except_origins: filter.except_origins || [],
              min_seeders: filter.min_seeders,
              max_seeders: filter.max_seeders,
              min_leechers: filter.min_leechers,
              max_leechers: filter.max_leechers,
              indexers: filter.indexers || [],
              actions: filter.actions || [],
              external: filter.external || [],
              release_profile_duplicate_id: filter.release_profile_duplicate_id,
              notifications: filter.notifications || [],
            } as Filter}
            onSubmit={handleSubmit}
            enableReinitialize={true}
            validationSchema={toFormikValidationSchema(schema)}
          >
            {({ values, dirty, resetForm }) => (
              <Form className="pt-1 pb-4 px-5">
                <FormErrorNotification />
                <Outlet />
                <FormButtonsGroup
                  values={values}
                  deleteAction={deleteAction}
                  dirty={dirty}
                  reset={resetForm}
                  isLoading={false}
                />
                <DEBUG values={values} />
              </Form>
            )}
          </Formik>
        </div>
      </div>
    </main>
  );
};
