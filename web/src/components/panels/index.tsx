/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useRef, ReactNode, ReactElement, RefObject } from "react";
import { XMarkIcon } from "@heroicons/react/24/solid";
import { Drawer } from "@base-ui/react/drawer";
import { useSelector } from "@tanstack/react-form";
import { useTranslation } from "react-i18next";

import { DEBUG } from "@components/debug";
import { useToggle } from "@hooks/hooks";
import { useAppForm, fieldErrors } from "@hooks/form";
import type { FormFieldErrors } from "@hooks/form";
import { DeleteModal } from "@components/modals";
import { classNames } from "@utils";

interface SlideOverShellProps {
  isOpen: boolean;
  toggle: () => void;
  initialFocus?: RefObject<HTMLElement | null>;
  zIndexClass?: string;
  children: ReactNode;
}

// Deliberately non-modal: a modal drawer locks page scroll and closes on any
// outside pointer press, which fights the overlays password managers inject
// and freezes the panel. See https://github.com/autobrr/autobrr/issues/2536.
export function SlideOverShell({ isOpen, toggle, initialFocus, zIndexClass, children }: SlideOverShellProps): ReactElement {
  return (
    <Drawer.Root
      open={isOpen}
      onOpenChange={(open) => {
        if (!open) {
          toggle();
        }
      }}
      modal={false}
      disablePointerDismissal
      swipeDirection="right"
    >
      <Drawer.Portal>
        <Drawer.Backdrop
          className={classNames("fixed inset-0", zIndexClass ?? "")}
          onClick={(e) => {
            e.stopPropagation();
            toggle();
          }}
        />
        {/* Clicks bubble through the React portal to ancestors (e.g. the expandable
            network row an update form is mounted in), so stop them at the boundary. */}
        <Drawer.Viewport
          className={classNames("fixed inset-0 flex justify-end pointer-events-none", zIndexClass ?? "")}
          onClick={(e) => e.stopPropagation()}
        >
          <Drawer.Popup
            initialFocus={initialFocus}
            className="pointer-events-auto h-full w-screen max-w-2xl shadow-xl [transform:translateX(var(--drawer-swipe-movement-x))] transition-transform ease-in-out duration-500 sm:duration-700 data-starting-style:[transform:translateX(100%)] data-ending-style:[transform:translateX(100%)]"
          >
            {/* Swipe-to-dismiss would discard form input on stray touch drags. */}
            <div className="h-full min-h-0" data-base-ui-swipe-ignore>
              {children}
            </div>
          </Drawer.Popup>
        </Drawer.Viewport>
      </Drawer.Portal>
    </Drawer.Root>
  );
}

interface SlideOverTitleProps {
  children: ReactNode;
  className?: string;
}

// Accessible drawer title; use once per drawer. For section headings inside
// the drawer body, use a plain heading element instead.
export function SlideOverTitle({ children, className }: SlideOverTitleProps): ReactElement {
  return (
    <Drawer.Title className={className ?? "text-lg font-medium text-gray-900 dark:text-white"}>
      {children}
    </Drawer.Title>
  );
}

interface SlideOverProps<DataType> {
  title: string;
  initialValues: DataType;
  validate?: (values: DataType) => FormFieldErrors;
  onSubmit: (values: DataType) => void;
  isOpen: boolean;
  toggle: () => void;
  children?: (values: DataType) => ReactNode;
  deleteAction?: () => void;
  type: "CREATE" | "UPDATE";
  testFn?: (data: unknown) => void;
  isTesting?: boolean;
  isTestSuccessful?: boolean;
  isTestError?: boolean;
  extraButtons?: (values: DataType) => ReactNode;
}

function SlideOver<DataType>({
  isOpen,
  toggle,
  ...props
}: SlideOverProps<DataType>): ReactElement {
  return (
    <SlideOverShell isOpen={isOpen} toggle={toggle}>
      <SlideOverForm toggle={toggle} {...props} />
    </SlideOverShell>
  );
}

// Lives inside the shell so the form state starts fresh every time the drawer opens
function SlideOverForm<DataType>({
  title,
  initialValues,
  validate,
  onSubmit,
  deleteAction,
  toggle,
  type,
  children,
  testFn,
  isTesting,
  isTestSuccessful,
  isTestError,
  extraButtons
}: Omit<SlideOverProps<DataType>, "isOpen">): ReactElement {
  const { t } = useTranslation("settings");
  const cancelModalButtonRef = useRef<HTMLInputElement | null>(null);

  const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);

  const form = useAppForm({
    defaultValues: initialValues,
    validators: {
      onChange: ({ value }) => validate ? fieldErrors(validate(value)) : undefined
    },
    onSubmit: ({ value }) => onSubmit(value)
  });

  const values = useSelector(form.store, (state) => state.values);

  return (
    <>
      {deleteAction && (
        <DeleteModal
          isOpen={deleteModalIsOpen}
          isLoading={isTesting || false}
          toggle={toggleDeleteModal}
          buttonRef={cancelModalButtonRef}
          deleteAction={deleteAction}
          title={t("panel.removeTitle", { title })}
          text={t("panel.removeText", { title })}
        />
      )}

      <form.AppForm>
        <form
          className="h-full min-h-0 flex flex-col bg-white dark:bg-gray-800"
          onSubmit={(e) => {
            e.preventDefault();
            e.stopPropagation();
            form.handleSubmit();
          }}
        >
          <div className="min-h-0 flex-1 overflow-y-auto">
            <div className="px-4 py-6 bg-gray-50 dark:bg-gray-900 sm:px-6">
              <div className="flex items-start justify-between space-x-3">
                <div className="space-y-1">
                  <SlideOverTitle>
                    {type === "CREATE"
                      ? t("panel.createTitle", { title })
                      : t("panel.updateTitle", { title })}
                  </SlideOverTitle>
                  <p className="text-sm text-gray-500 dark:text-gray-400">
                    {type === "CREATE"
                      ? t("panel.createDescription", { title })
                      : t("panel.updateDescription", { title })}
                  </p>
                </div>
                <div className="h-7 flex items-center">
                  <button
                    type="button"
                    className="bg-white dark:bg-gray-900 rounded-md text-gray-400 hover:text-gray-500 cursor-pointer focus:outline-hidden focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                    onClick={toggle}
                  >
                    <span className="sr-only">{t("panel.closePanel")}</span>
                    <XMarkIcon className="h-6 w-6" aria-hidden="true" />
                  </button>
                </div>
              </div>
            </div>

            {!!values && children !== undefined ? (
              children(values)
            ) : null}

            <DEBUG values={values} />
          </div>

          <div className="shrink-0 px-4 border-t border-gray-200 dark:border-gray-700 py-5 sm:px-6">
            <div className={classNames(type === "CREATE" ? "justify-end" : "justify-between", "space-x-3 flex")}>
              {type === "UPDATE" && (
                <button
                  type="button"
                  className="inline-flex items-center justify-center px-4 py-2 border border-transparent cursor-pointer font-medium rounded-md text-red-700 dark:text-white bg-red-100 dark:bg-red-700 hover:bg-red-200 dark:hover:bg-red-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
                  onClick={toggleDeleteModal}
                >
                  {t("panel.remove")}
                </button>
              )}
              <div className="flex">
                {!!values && extraButtons !== undefined && (
                  extraButtons(values)
                )}

                {testFn && (
                  <button
                    type="button"
                    className={classNames(
                      isTestSuccessful
                        ? "text-green-500 border-green-500 bg-green-50"
                        : isTestError
                          ? "text-red-500 border-red-500 bg-red-50"
                          : "border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 focus:border-rose-700 active:bg-rose-700",
                      isTesting ? "cursor-not-allowed" : "",
                      "mr-2 inline-flex items-center px-4 py-2 border font-medium rounded-md shadow-xs text-sm transition ease-in-out duration-150 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                    )}
                    disabled={isTesting}
                    onClick={(e) => {
                      e.preventDefault();
                      testFn(values);
                    }}
                  >
                    {isTesting ? (
                      <svg
                        className="animate-spin h-5 w-5 text-green-500"
                        xmlns="http://www.w3.org/2000/svg"
                        fill="none"
                        viewBox="0 0 24 24"
                      >
                        <circle
                          className="opacity-25"
                          cx="12"
                          cy="12"
                          r="10"
                          stroke="currentColor"
                          strokeWidth="4"
                        ></circle>
                        <path
                          className="opacity-75"
                          fill="currentColor"
                          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                        ></path>
                      </svg>
                    ) : isTestSuccessful ? (
                      t("panel.ok")
                    ) : isTestError ? (
                      t("panel.error")
                    ) : (
                      t("panel.test")
                    )}
                  </button>
                )}

                <button
                  type="button"
                  className="bg-white dark:bg-gray-700 py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md shadow-xs text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                  onClick={(e) => {
                    e.preventDefault();
                    toggle();
                  }}
                >
                  {t("panel.cancel")}
                </button>
                <button
                  type="button"
                  className="ml-4 inline-flex justify-center py-2 px-4 border border-transparent shadow-xs text-sm font-medium rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                  onClick={(e) => {
                    e.preventDefault();
                    form.handleSubmit();
                  }}
                >
                  {type === "CREATE" ? t("panel.create") : t("panel.save")}
                </button>
              </div>
            </div>
          </div>
        </form>
      </form.AppForm>
    </>
  );
}

export { SlideOver };
