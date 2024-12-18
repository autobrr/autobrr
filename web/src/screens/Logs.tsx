/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { Fragment, useEffect, useRef, useState } from "react";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Menu, MenuButton, MenuItem, MenuItems, Transition } from "@headlessui/react";
import { DebounceInput } from "react-debounce-input";
import {
  Cog6ToothIcon,
  DocumentArrowDownIcon
} from "@heroicons/react/24/outline";
import { ExclamationCircleIcon } from "@heroicons/react/24/solid";
import { format } from "date-fns/format";
import { toast } from "react-hot-toast";

import { APIClient } from "@api/APIClient";
import { Checkbox } from "@components/Checkbox";
import { baseUrl, classNames, simplifyDate } from "@utils";
import { SettingsContext } from "@utils/Context";
import { EmptySimple } from "@components/emptystates";
import { RingResizeSpinner } from "@components/Icons";
import Toast from "@components/notifications/Toast";

type LogEvent = {
  time: string;
  level: string;
  message: string;
};

type LogLevel = "TRC" | "DBG" | "INF" | "ERR" | "WRN" | "FTL" | "PNC";

const LogColors: Record<LogLevel, string> = {
  "TRC": "text-purple-300",
  "DBG": "text-yellow-500",
  "INF": "text-green-500",
  "ERR": "text-red-500",
  "WRN": "text-yellow-500",
  "FTL": "text-red-500",
  "PNC": "text-red-600"
};

export const Logs = () => {
  const [settings] = SettingsContext.use();

  const messagesEndRef = useRef<HTMLDivElement>(null);

  const [logs, setLogs] = useState<LogEvent[]>([]);
  const [searchFilter, setSearchFilter] = useState("");
  const [, setRegexPattern] = useState<RegExp | null>(null);
  const [filteredLogs, setFilteredLogs] = useState<LogEvent[]>([]);
  const [isInvalidRegex, setIsInvalidRegex] = useState(false);

  useEffect(() => {
    const scrollToBottom = () => {
      if (messagesEndRef.current) {
        messagesEndRef.current.scrollTop = messagesEndRef.current.scrollHeight;
      }
    };
    if (settings.scrollOnNewLog)
      scrollToBottom();
  }, [filteredLogs, settings.scrollOnNewLog]);

  // Add a useEffect to clear logs div when settings.scrollOnNewLog changes to prevent duplicate entries.
  useEffect(() => {
    setLogs([]);
  }, [settings.scrollOnNewLog]);

  useEffect(() => {
    const es = APIClient.events.logs();

    es.onmessage = (event) => {
      const newData = JSON.parse(event.data) as LogEvent;
      setLogs((prevState) => [...prevState, newData]);
    };

    return () => es.close();
  }, [setLogs, settings]);

  useEffect(() => {
    if (!searchFilter.length) {
      setFilteredLogs(logs);
      setIsInvalidRegex(false);
      return;
    }

    try {
      const pattern = new RegExp(searchFilter, "i");
      setRegexPattern(pattern);
      const newLogs = logs.filter(log => pattern.test(log.message));
      setFilteredLogs(newLogs);
      setIsInvalidRegex(false);
    } catch (error) {
      // Handle regex errors by showing nothing when the regex pattern is invalid
      setFilteredLogs([]);
      setIsInvalidRegex(true);
    }
  }, [logs, searchFilter]);

  return (
    <main>
      <div className="my-6 max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold text-black dark:text-white">Logs</h1>
      </div>

      <div className="max-w-screen-xl mx-auto pb-12 px-2 sm:px-4 lg:px-8">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-250 dark:border-gray-775 px-2 sm:px-4 pt-3 sm:pt-4 pb-3 sm:pb-4">
          <div className="flex relative mb-3">
            <DebounceInput
              minLength={2}
              debounceTimeout={200}
              onChange={(event) => {
                const inputValue = event.target.value.toLowerCase().trim();
                setSearchFilter(inputValue);
              }}
              className={classNames(
                "focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700",
                "block w-full dark:bg-gray-900 shadow-sm dark:text-gray-100 sm:text-sm rounded-md"
              )}
              placeholder="Enter a regex pattern to filter logs by..."
            />
            {isInvalidRegex && (
              <div className="absolute mt-1.5 right-14 items-center text-xs text-red-500">
                <ExclamationCircleIcon className="h-6 w-6 inline mr-1" />
              </div>
            )}
            <LogsDropdown />
          </div>

          <div className="overflow-y-auto px-2 rounded-lg h-[60vh] min-w-full bg-gray-100 dark:bg-gray-900 overflow-auto" ref={messagesEndRef}>
            {filteredLogs.map((entry, idx) => (
              <div
                key={idx}
                className={classNames(
                  settings.indentLogLines ? "grid justify-start grid-flow-col" : "",
                  settings.hideWrappedText ? "truncate hover:text-ellipsis hover:whitespace-normal" : ""
                )}
              >
                <span
                  title={entry.time}
                  className="font-mono text-gray-500 dark:text-gray-600 h-full"
                >
                  {format(new Date(entry.time), "HH:mm:ss")}
                </span>
                {entry.level in LogColors ? (
                  <span
                    className={classNames(
                      LogColors[entry.level as LogLevel],
                      "font-mono font-semibold h-full"
                    )}
                  >
                    {` ${entry.level} `}
                  </span>
                ) : null}
                <span className="text-black dark:text-gray-300">
                  {entry.message}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="max-w-screen-xl mx-auto pb-10 px-2 sm:px-4 lg:px-8">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-250 dark:border-gray-775 px-4 sm:px-6 pt-3 sm:pt-4">
          <LogFiles />
        </div>
      </div>

    </main>
  );
};

export const LogFiles = () => {
  const { isError, error, data } = useSuspenseQuery({
    queryKey: ["log-files"],
    queryFn: () => APIClient.logs.files(),
    retry: false,
    refetchOnWindowFocus: false
  });

  if (isError) {
    console.log("could not load log files", error);
  }

  return (
    <div>
      <div className="mt-2">
        <h2 className="text-lg leading-4 font-bold text-gray-900 dark:text-white">Log files</h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Download old log files.
        </p>
      </div>

      {data && data.files && data.files.length > 0 ? (
        <ul className="py-3 min-w-full relative">
          <li className="grid grid-cols-12 mb-2 border-b border-gray-200 dark:border-gray-700">
            <div className="hidden sm:block col-span-5 px-2 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Name
            </div>
            <div className="col-span-8 sm:col-span-4 px-1 sm:px-2 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Last modified
            </div>
            <div className="col-span-3 sm:col-span-2 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Size
            </div>
          </li>

          {data.files.map((f, idx) => <LogFilesItem key={idx} file={f} />)}
        </ul>
      ) : (
        <EmptySimple
          title="No old log files"
          subtitle=""
        />
      )}
    </div>
  );
};

interface LogFilesItemProps {
  file: LogFile;
}

const LogFilesItem = ({ file }: LogFilesItemProps) => {
  const [isDownloading, setIsDownloading] = useState(false);

  const handleDownload = async () => {
    setIsDownloading(true);

    // Add a custom toast before the download starts
    const toastId = toast.custom((t) => (
      <Toast type="info" body="Log file is being sanitized. Please wait..." t={t} />
    ));

    const response = await fetch(`${baseUrl()}api/logs/files/${file.filename}`);
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = file.filename;
    link.click();
    URL.revokeObjectURL(url);

    // Dismiss the custom toast after the download is complete
    toast.dismiss(toastId);

    setIsDownloading(false);
  };

  return (
    <li className="text-gray-500 dark:text-gray-400">
      <div className="grid grid-cols-12 items-center py-2">
        <div className="col-span-4 sm:col-span-5 px-2 py-0 truncate hidden sm:block sm:text-sm text-md font-medium text-gray-900 dark:text-gray-200">
          <div className="block truncate justify-between">
            {file.filename}
          </div>
        </div>
        <div className="col-span-8 sm:col-span-4 block truncate px-1 sm:px-2 items-center text-sm font-medium text-gray-900 dark:text-gray-200" title={file.updated_at}>
          {simplifyDate(file.updated_at)}
        </div>
        <div className="col-span-3 sm:col-span-2 flex items-center text-sm font-small sm:font-medium text-gray-900 dark:text-gray-200">
          {file.size}
        </div>
        <div className="col-span-1 sm:col-span-1 pl-0 flex items-center justify-center text-sm font-medium text-gray-900 dark:text-white">
          <div className="logFilesItem">
            <button
              className={classNames(
                "text-gray-900 dark:text-gray-300",
                "font-medium group flex rounded-md items-center px-2 py-2 text-sm"
              )}
              title={!isDownloading ? "Download file" : "Sanitizing log..."}
              onClick={handleDownload}
            >
              {!isDownloading ? (
                <DocumentArrowDownIcon
                  className="text-blue-500 w-5 h-5 iconHeight"
                  aria-hidden="true"
                />
              ) : (
                <RingResizeSpinner
                  className="text-blue-500 w-5 h-5 iconHeight"
                  aria-hidden="true"
                />
              )}
            </button>
          </div>
        </div>
      </div>
    </li>
  );
};

// interface LogsDropdownProps {}

const LogsDropdown = () => {
  const [settings, setSettings] = SettingsContext.use();

  const onSetValue = (
    key: "scrollOnNewLog" | "indentLogLines" | "hideWrappedText",
    newValue: boolean
  ) => setSettings((prevState) => ({
    ...prevState,
    [key]: newValue
  }));

  return (
    <Menu as="div">
      <MenuButton className="px-4 py-2">
        <Cog6ToothIcon
          className="w-5 h-5 text-gray-700 hover:text-gray-900 dark:text-gray-100 dark:hover:text-gray-400"
          aria-hidden="true"
        />
      </MenuButton>
      <Transition
        as={Fragment}
        enter="transition ease-out duration-100"
        enterFrom="transform opacity-0 scale-95"
        enterTo="transform opacity-100 scale-100"
        leave="transition ease-in duration-75"
        leaveFrom="transform opacity-100 scale-100"
        leaveTo="transform opacity-0 scale-95"
      >
        <MenuItems
          className="absolute right-0 mt-1 origin-top-right bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700 rounded-md shadow-lg ring-1 ring-black ring-opacity-10 focus:outline-none"
        >
          <div className="p-3">
            <MenuItem>
              {() => (
                <Checkbox
                  label="Scroll to bottom on new message"
                  value={settings.scrollOnNewLog}
                  setValue={(newValue) => onSetValue("scrollOnNewLog", newValue)}
                />
              )}
            </MenuItem>
            <MenuItem>
              {() => (
                <Checkbox
                  label="Indent log lines"
                  description="Indent each log line according to their respective starting position."
                  value={settings.indentLogLines}
                  setValue={(newValue) => onSetValue("indentLogLines", newValue)}
                />
              )}
            </MenuItem>
            <MenuItem>
              {() => (
                <Checkbox
                  label="Hide wrapped text"
                  description="Hides text that is meant to be wrapped."
                  value={settings.hideWrappedText}
                  setValue={(newValue) => onSetValue("hideWrappedText", newValue)}
                />
              )}
            </MenuItem>
          </div>
        </MenuItems>
      </Transition>
    </Menu>
  );
};
