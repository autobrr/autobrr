/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Suspense, useEffect, useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Form, Formik, useFormikContext } from "formik";
import type { FormikErrors, FormikValues } from "formik";
import { z } from "zod";
import { toast } from "react-hot-toast";
import { toFormikValidationSchema } from "zod-formik-adapter";
import { ChevronRightIcon } from "@heroicons/react/24/solid";
import { NavLink, Route, Routes, useLocation, useNavigate, useParams } from "react-router-dom";

import { APIClient } from "@api/APIClient";
import { useToggle } from "@hooks/hooks";
import { classNames } from "@utils";
import { DOWNLOAD_CLIENTS } from "@domain/constants";

import { DEBUG } from "@components/debug";
import Toast from "@components/notifications/Toast";
import { DeleteModal } from "@components/modals";
import { SectionLoader } from "@components/SectionLoader";

import { filterKeys } from "./List";
import * as Section from "./sections";

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
        "transition border-b-2 whitespace-nowrap py-4 duration-3000 px-1 font-medium text-sm first:rounded-tl-lg last:rounded-tr-lg",
        isActive
          ? "text-blue-600 dark:text-white border-blue-600 dark:border-blue-500"
          : "text-gray-550 hover:text-blue-500 dark:hover:text-white border-transparent"
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
          className="flex items-center justify-center px-4 py-2 rounded-md sm:text-sm transition bg-red-700 dark:bg-red-900 hover:dark:bg-red-700 hover:bg-red-800 text-white focus:outline-none"
          onClick={toggleDeleteModal}
        >
          Delete Filter
        </button>

        <div className="flex justify-between mb-4 sm:mb-0">
          {/* {dirty && <span className="mr-4 text-sm text-gray-500">Unsaved changes..</span>} */}
          <button
            type="button"
            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 transition rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
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
            className="ml-1 sm:ml-4 flex items-center px-4 py-2 border border-transparent transition shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
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
  }, [isSubmitting, isValid, isValidating]);

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
}).superRefine((value, ctx) => {
  if (!value.name) {
    ctx.addIssue({
      message: "Must have a name",
      code: z.ZodIssueCode.custom,
      path: ["name"]
    });
  }

  if (value.type == "WEBHOOK") {
    if (!value.webhook_method) {
      ctx.addIssue({
        message: "Must select method",
        code: z.ZodIssueCode.custom,
        path: ["webhook_method"]
      });
    }
    if (!value.webhook_host) {
      ctx.addIssue({
        message: "Must have webhook host",
        code: z.ZodIssueCode.custom,
        path: ["webhook_host"]
      });
    }
    if (!value.webhook_expect_status) {
      ctx.addIssue({
        message: "Must have webhook expect status",
        code: z.ZodIssueCode.custom,
        path: ["webhook_expect_status"]
      });
    }
  }

  if (value.type == "EXEC") {
    if (!value.exec_cmd) {
      ctx.addIssue({
        message: "Must have exec cmd",
        code: z.ZodIssueCode.custom,
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
  indexers: z.array(indexerSchema).min(1, { message: "Must select at least one indexer" }),
  actions: z.array(actionSchema),
  external: z.array(externalFilterSchema)
});

export const FilterDetails = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { filterId } = useParams<{ filterId: string }>();

  if (filterId === "0" || filterId === undefined) {
    navigate("/filters");
  }

   
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
      <div className="my-6 max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center text-black dark:text-white">
        <h1 className="text-3xl font-bold">
          <NavLink to="/filters">
            Filters
          </NavLink>
        </h1>
        <ChevronRightIcon className="h-6 w-4 shrink-0 sm:shrink sm:h-6 sm:w-6 mx-1" aria-hidden="true" />
        <h1 className="text-3xl font-bold truncate" title={filter.name}>{filter.name}</h1>
      </div>
      <div className="max-w-screen-xl mx-auto pb-12 px-2 sm:px-6 lg:px-8">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-250 dark:border-gray-775">
          <div className="rounded-t-lg bg-gray-125 dark:bg-gray-850 border-b border-gray-200 dark:border-gray-750">
            <nav className="px-4 -mb-px flex space-x-6 sm:space-x-8 overflow-x-auto">
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
              <Form className="pt-1 pb-4 px-5">
                <FormErrorNotification />
                <Suspense fallback={<SectionLoader $size="large" />}>
                  <Routes>
                    <Route index element={<Section.General />} />
                    <Route path="movies-tv" element={<Section.MoviesTv />} />
                    <Route path="music" element={<Section.Music values={values} />} />
                    <Route path="advanced" element={<Section.Advanced values={values} />} />
                    <Route path="external" element={<Section.External />} />
                    <Route path="actions" element={<Section.Actions filter={filter} values={values} />} />
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
    </main>
  );
};
