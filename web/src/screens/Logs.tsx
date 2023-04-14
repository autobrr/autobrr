import { Fragment, useEffect, useRef, useState } from "react";
import { ExclamationTriangleIcon } from "@heroicons/react/24/solid";
import format from "date-fns/format";
import { DebounceInput } from "react-debounce-input";
import { APIClient } from "../api/APIClient";
import { Checkbox } from "../components/Checkbox";
import { classNames, simplifyDate } from "../utils";
import { SettingsContext } from "../utils/Context";
import { EmptySimple } from "../components/emptystates";
import {
  Cog6ToothIcon,
  DocumentArrowDownIcon
} from "@heroicons/react/24/outline";
import { useQuery } from "react-query";
import { Menu, Transition } from "@headlessui/react";
import { baseUrl } from "../utils";


type LogEvent = {
  time: string;
  level: string;
  message: string;
};

type LogLevel = "TRACE" | "DEBUG" | "INFO" | "ERROR" | "WARN";

const LogColors: Record<LogLevel, string> = {
  "TRACE": "text-purple-300",
  "DEBUG": "text-yellow-500",
  "INFO": "text-green-500",
  "ERROR": "text-red-500",
  "WARN": "text-yellow-500"
};

export const Logs = () => {
  const [settings] = SettingsContext.use();

  const messagesEndRef = useRef<HTMLDivElement>(null);
  
  const [logs, setLogs] = useState<LogEvent[]>([]);
  const [searchFilter, setSearchFilter] = useState("");
  const [filteredLogs, setFilteredLogs] = useState<LogEvent[]>([]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth", block: "end", inline: "end" });
  };

  useEffect(() => {
    const es = APIClient.events.logs();

    es.onmessage = (event) => {
      const newData = JSON.parse(event.data) as LogEvent;
      setLogs((prevState) => [...prevState, newData]);

      if (settings.scrollOnNewLog)
        scrollToBottom();
    };

    return () => es.close();
  }, [setLogs, settings]);

  useEffect(() => {
    if (!searchFilter.length) {
      setFilteredLogs(logs);
      return;
    }
    
    const newLogs: LogEvent[] = [];
    logs.forEach((log) => {
      if (log.message.indexOf(searchFilter) !== -1)
        newLogs.push(log);
    });

    setFilteredLogs(newLogs);
  }, [logs, searchFilter]);

  return (
    <main>
      <header className="pt-10 pb-5">
        <div className="max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8">
          <h1 className="text-3xl font-bold text-black dark:text-white">Logs</h1>
        </div>
      </header>


      <div className="max-w-screen-xl mx-auto pb-12 px-2 sm:px-4 lg:px-8">
        <div className="flex justify-center py-4">
          <ExclamationTriangleIcon
            className="h-5 w-5 text-yellow-400"
            aria-hidden="true"
          />
          <p className="ml-2 text-sm text-black dark:text-gray-400">This page shows only new logs, i.e. no history.</p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg px-2 sm:px-4 pt-3 sm:pt-4 pb-3 sm:pb-4">
          <div className="flex relative mb-3">
            <DebounceInput
              minLength={2}
              debounceTimeout={200}
              onChange={(event) => setSearchFilter(event.target.value.toLowerCase().trim())}
              id="filter"
              type="text"
              autoComplete="off"
              className={classNames(
                "focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700",
                "block w-full dark:bg-gray-900 shadow-sm dark:text-gray-100 sm:text-sm rounded-md"
              )}
              placeholder="Enter a string to filter logs by..."
            />

            <LogsDropdown />
          </div>

          <div className="overflow-y-auto px-2 rounded-lg h-[60vh] min-w-full bg-gray-100 dark:bg-gray-900 overflow-auto">
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
                  className="font-mono text-gray-500 dark:text-gray-600 mr-2 h-full"
                >
                  {format(new Date(entry.time), "HH:mm:ss.SSS")}
                </span>
                {entry.level in LogColors ? (
                  <span
                    className={classNames(
                      LogColors[entry.level as LogLevel],
                      "font-mono font-semibold h-full"
                    )}
                  >
                    {entry.level}
                    {" "}
                  </span>
                ) : null}
                <span className="ml-2 text-black dark:text-gray-300">
                  {entry.message}
                </span>
              </div>
            ))}
            <div className="mt-6" ref={messagesEndRef} />
          </div>
        </div>
      </div>

      <div className="max-w-screen-xl mx-auto pb-10 px-2 sm:px-4 lg:px-8">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg px-2 sm:px-4 pt-3 sm:pt-4">
          <LogFiles />
        </div>
      </div>

    </main>
  );
};

export const LogFiles = () => {
  const { isLoading, data } = useQuery(
    ["log-files"],
    () => APIClient.logs.files(),
    {
      retry: false,
      refetchOnWindowFocus: false,
      onError: err => console.log(err)
    }
  );

  return (
    <div>
      <div className="mt-2">
        <h2 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Log files</h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Download old log files.
        </p>
      </div>

      {data && data.files.length > 0 ? (
        <section className="py-3 light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
          <ol className="min-w-full relative">
            <li className="hidden sm:grid grid-cols-12 mb-2 border-b border-gray-200 dark:border-gray-700">
              <div className="col-span-5 px-2 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Name
              </div>
              <div className="col-span-4 px-2 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Last modified
              </div>
              <div className="col-span-2 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Size
              </div>
            </li>

            {data && data.files.map((f, idx) => <LogFilesItem key={idx} file={f} />)}
          </ol>
        </section>
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

const Dots = () => {
  const [step, setStep] = useState(1);

  useEffect(() => {
    const interval = setInterval(() => {
      setStep((prevStep) => (prevStep % 3) + 1);
    }, 300);

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="flex">
      <div
        className={`h-2 w-2 bg-blue-500 rounded-full mx-1 ${
          step === 1 ? "opacity-100" : "opacity-30"
        }`}
      />
      <div
        className={`h-2 w-2 bg-blue-500 rounded-full mx-1 ${
          step === 2 ? "opacity-100" : "opacity-30"
        }`}
      />
      <div
        className={`h-2 w-2 bg-blue-500 rounded-full mx-1 ${
          step === 3 ? "opacity-100" : "opacity-30"
        }`}
      />
    </div>
  );
};

const LogFilesItem = ({ file }: LogFilesItemProps) => {
  const [isDownloading, setIsDownloading] = useState(false);

  const handleDownload = async () => {
    setIsDownloading(true);
    const response = await fetch(`${baseUrl()}api/logs/files/${file.filename}`);
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = file.filename;
    link.click();
    URL.revokeObjectURL(url);
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
              title="Download file"
              onClick={handleDownload}
            >
              {!isDownloading ? (
                <DocumentArrowDownIcon
                  className="text-blue-500 w-5 h-5 iconHeight"
                  aria-hidden="true"
                />
              ) : (
                <div className="h-5 flex items-center">
                  <span className="sanitizing-text">Sanitizing log</span>
                  <Dots />
                </div>
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
      <Menu.Button className="px-4 py-2">
        <Cog6ToothIcon
          className="w-5 h-5 text-gray-700 hover:text-gray-900 dark:text-gray-100 dark:hover:text-gray-400"
          aria-hidden="true"
        />
      </Menu.Button>
      <Transition
        as={Fragment}
        enter="transition ease-out duration-100"
        enterFrom="transform opacity-0 scale-95"
        enterTo="transform opacity-100 scale-100"
        leave="transition ease-in duration-75"
        leaveFrom="transform opacity-100 scale-100"
        leaveTo="transform opacity-0 scale-95"
      >
        <Menu.Items
          className="absolute right-0 mt-1 origin-top-right bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700 rounded-md shadow-lg ring-1 ring-black ring-opacity-10 focus:outline-none"
        >
          <div className="p-3">
            <Menu.Item>
              {({ active }) => (
                <Checkbox
                  label="Scroll to bottom on new message"
                  value={settings.scrollOnNewLog}
                  setValue={(newValue) => onSetValue("scrollOnNewLog", newValue)}
                />
              )}
            </Menu.Item>
            <Menu.Item>
              {({ active }) => (
                <Checkbox
                  label="Indent log lines"
                  description="Indent each log line according to their respective starting position."
                  value={settings.indentLogLines}
                  setValue={(newValue) => onSetValue("indentLogLines", newValue)}
                />
              )}
            </Menu.Item>
            <Menu.Item>
              {({ active }) => (
                <Checkbox
                  label="Hide wrapped text"
                  description="Hides text that is meant to be wrapped."
                  value={settings.hideWrappedText}
                  setValue={(newValue) => onSetValue("hideWrappedText", newValue)}
                />
              )}
            </Menu.Item>
          </div>
        </Menu.Items>
      </Transition>
    </Menu>
  );
};
