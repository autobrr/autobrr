/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect, useRef } from "react";
import { useMutation, useSuspenseQuery } from "@tanstack/react-query";
import { getRouteApi, Link, Outlet, useNavigate } from "@tanstack/react-router";
import { useSelector } from "@tanstack/react-form";
import type { AnyFormApi, StandardSchemaV1 } from "@tanstack/react-form";
import { z } from "zod";
import { ChevronRightIcon } from "@heroicons/react/24/solid";
import { useTranslation } from "react-i18next";
import i18n from "../../i18n";

import { APIClient } from "@api/APIClient";
import { FilterByIdQueryOptions } from "@api/queries";
import { FilterKeys } from "@api/query_keys";
import { useToggle } from "@hooks/hooks";
import { useAppForm, useFormContext, useFormValues, errorMessages, touchInvalidFields } from "@hooks/form";
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
  { name: "details.tabs.general", href: "/filters/$filterId", exact: true },
  { name: "details.tabs.moviesTv", href: "/filters/$filterId/movies-tv" },
  { name: "details.tabs.music", href: "/filters/$filterId/music" },
  { name: "details.tabs.books", href: "/filters/$filterId/books" },
  { name: "details.tabs.advanced", href: "/filters/$filterId/advanced" },
  { name: "details.tabs.external", href: "/filters/$filterId/external" },
  { name: "details.tabs.actions", href: "/filters/$filterId/actions" },
  { name: "details.tabs.notifications", href: "/filters/$filterId/notifications", newFeature: true }
];

export interface NavLinkProps {
  item: tabType;
}

function TabNavLink({ item }: NavLinkProps) {
  const { t } = useTranslation("filters");

  return (
    <Link
      to={item.href}
      activeOptions={{ exact: item.exact }}
      search={{}}
      params={{}}
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
              {t(item.name)}
            </span>
            {item.newFeature &&
              <span className="ml-2 inline-flex items-center rounded-md bg-green-100 px-1.5 py-0.5 text-xs font-medium text-green-700 dark:bg-green-400/10 dark:text-green-400">{t("details.tabs.new")}</span>
            }
          </span>
        )
      }}
    </Link>
  );
}

interface FormButtonsGroupProps {
  deleteAction: () => void;
  reset: () => void;
  isLoading: boolean;
}

const FormButtonsGroup = ({ deleteAction, reset, isLoading }: FormButtonsGroupProps) => {
  const { t } = useTranslation("filters");
  const form = useFormContext();
  const name = useSelector(form.store, (state) => state.values.name);
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
        title={t("details.removeTitle", { name })}
        text={t("details.removeText")}
      />

      <div className="px-0.5 mt-8 flex flex-col-reverse sm:flex-row flex-wrap-reverse justify-between">
        <button
          type="button"
          className="flex items-center justify-center px-4 py-2 rounded-md sm:text-sm transition bg-red-700 dark:bg-red-900 dark:hover:bg-red-700 hover:bg-red-800 text-white focus:outline-hidden cursor-pointer"
          onClick={toggleDeleteModal}
        >
          {t("details.delete")}
        </button>

        <div className="flex justify-between mb-4 sm:mb-0">
          <button
            type="button"
            className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 transition rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500 cursor-pointer"
            onClick={(e) => {
              e.preventDefault();
              reset();

              toast.custom((toastInstance) => <Toast type="success" body={t("details.resetSuccess")} t={toastInstance} />);
            }}
          >
            {t("details.reset")}
          </button>
          <button
            type="submit"
            className="ml-1 sm:ml-4 flex items-center px-4 py-2 border border-transparent transition shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 cursor-pointer"
          >
            {t("details.save")}
          </button>
        </div>
      </div>
    </>
  );
};

const FormDebug = () => {
  const values = useFormValues<Filter>();

  return <DEBUG values={values} />;
};

// Renders "actions->0->client_id: message" lines for the validation toast
const collectFieldErrors = (form: AnyFormApi) => {
  const lines: string[] = [];
  for (const [field, meta] of Object.entries(form.state.fieldMeta)) {
    const path = field.replace(/\[(\d+)\]/g, ".$1").split(".").join("->");
    for (const message of errorMessages(meta?.errors ?? [])) {
      lines.push(`${path}: ${message}`);
    }
  }

  return lines;
};

const actionSchema = z.object({
  enabled: z.boolean(),
  name: z.string().min(1, { message: "Required" }),
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
        message: i18n.t("filters:details.mustSelectClient"),
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

const schema = z.object({
  name: z.string().min(1, { message: "Required" }),
  max_downloads: z.number().optional(),
  max_downloads_unit: z.string().optional(),
  max_downloads_period: z.number().min(1).optional(),
  max_downloads_window_type: z.string().optional(),
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

const filterFormValues = (filter: Filter): Filter => ({
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
  max_downloads_period: filter.max_downloads_period || 1,
  max_downloads_window_type: filter.max_downloads_window_type || "FIXED",
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
} as Filter);

export const FilterDetails = () => {
  const { t } = useTranslation("filters");
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

      toast.custom((tst) => (
        <Toast type="success" body={t("list.updated", {name: newFilter.name})} t={tst} />
      ));
    }
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => APIClient.filters.delete(id),
    onSuccess: () => {
      // Invalidate filters just in case, most likely not necessary but can't hurt.
      queryClient.invalidateQueries({ queryKey: FilterKeys.lists() });
      queryClient.removeQueries({ queryKey: FilterKeys.detail(params.filterId) });

      toast.custom((tst) => (
        <Toast type="success" body={t("list.deleted", {name: filter?.name})} t={tst} />
      ));

      // redirect
      navigate({ to: "/filters" });
    }
  });

  const onSubmit = (data: Filter) => {
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

  const form = useAppForm({
    defaultValues: filterFormValues(filter),
    validators: {
      // The schema covers a subset of the filter, which Standard Schema's covariant input type rejects
      onChange: schema as unknown as StandardSchemaV1<Filter>
    },
    onSubmit: ({ value }) => onSubmit(value),
    onSubmitInvalid: ({ formApi }) => {
      touchInvalidFields(formApi);
      const errors = collectFieldErrors(formApi);

      toast.custom((tst) => (
        <Toast
          type="error"
          body={`${errors.length > 1 ? t("details.validationErrors") : t("details.validationError")}: \n${errors.join("\n")}`}
          t={tst}
        />
      ));
    }
  });

  // Follow the query cache after a save, refetch or navigation to another filter.
  // Skipped on mount so tab sections can seed values in their own mount effects.
  const resetForm = form.reset;
  const loadedFilter = useRef(filter);
  useEffect(() => {
    if (loadedFilter.current === filter) {
      return;
    }
    loadedFilter.current = filter;
    resetForm(filterFormValues(filter));
  }, [filter, resetForm]);

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
          <form.AppForm>
            <form
              className="pt-1 pb-4 px-5"
              onSubmit={(e) => {
                e.preventDefault();
                e.stopPropagation();
                form.handleSubmit();
              }}
            >
              <Outlet />
              <FormButtonsGroup
                deleteAction={deleteAction}
                reset={() => form.reset()}
                isLoading={false}
              />
              <FormDebug />
            </form>
          </form.AppForm>
        </div>
      </div>
    </main>
  );
};
