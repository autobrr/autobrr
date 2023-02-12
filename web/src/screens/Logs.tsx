import { useEffect, useRef, useState } from "react";
import { ExclamationTriangleIcon } from "@heroicons/react/24/solid";
import format from "date-fns/format";
import { DebounceInput } from "react-debounce-input";
import { APIClient } from "../api/APIClient";
import { Checkbox } from "../components/Checkbox";
import { baseUrl, classNames, simplifyDate } from "../utils";
import { SettingsContext } from "../utils/Context";
import { EmptySimple } from "../components/emptystates";
import { DocumentArrowDownIcon } from "@heroicons/react/24/outline";
import { useQuery } from "react-query";
import { Link } from "react-router-dom";

type LogEvent = {
  time: string;
  level: string;
  message: string;
};

type LogLevel = "TRACE" | "DEBUG" | "INFO" | "ERROR";

const LogColors: Record<LogLevel, string> = {
  "TRACE": "text-purple-300",
  "DEBUG": "text-yellow-500",
  "INFO": "text-green-500",
  "ERROR": "text-red-500"
};

export const Logs = () => {
  const [settings, setSettings] = SettingsContext.use();
  
  const messagesEndRef = useRef<HTMLDivElement>(null);
  
  const [logs, setLogs] = useState<LogEvent[]>([]);
  const [searchFilter, setSearchFilter] = useState("");
  const [filteredLogs, setFilteredLogs] = useState<LogEvent[]>([]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "auto" });
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

  const onSetValue = (
    key: "scrollOnNewLog" | "indentLogLines" | "hideWrappedText",
    newValue: boolean
  ) => setSettings((prevState) => ({
    ...prevState,
    [key]: newValue
  }));

  return (
    <main>
      <header className="pt-10 pb-5">
        <div className="max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8">
          <h1 className="text-3xl font-bold text-black dark:text-white">Logs</h1>
          <div className="flex justify-center mt-1">
            <ExclamationTriangleIcon
              className="h-5 w-5 text-yellow-400"
              aria-hidden="true"
            />
            <p className="ml-2 text-sm text-black dark:text-gray-400">This page shows only new logs, i.e. no history.</p>
          </div>
        </div>
      </header>
      <div className="max-w-screen-xl mx-auto pb-12 px-2 sm:px-4 lg:px-8">
        <div
          className="bg-white dark:bg-gray-800 rounded-lg shadow-lg px-2 sm:px-4 pt-3 sm:pt-4"
        >
          <DebounceInput
            minLength={2}
            debounceTimeout={200}
            onChange={(event) => setSearchFilter(event.target.value.toLowerCase().trim())}
            id="filter"
            type="text"
            autoComplete="off"
            className={classNames(
              "focus:ring-blue-500 dark:focus:ring-blue-500 focus:border-blue-500 dark:focus:border-blue-500 border-gray-300 dark:border-gray-700",
              "block w-full dark:bg-gray-800 shadow-sm dark:text-gray-100 sm:text-sm rounded-md"
            )}
            placeholder="Enter a string to filter logs by..."
          />
          <div
            className="mt-2 overflow-y-auto p-2 rounded-lg h-[60vh] min-w-full bg-gray-100 dark:bg-gray-900 overflow-auto"
          >
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
            <div ref={messagesEndRef} />
          </div>
          <Checkbox
            label="Scroll to bottom on new message"
            value={settings.scrollOnNewLog}
            setValue={(newValue) => onSetValue("scrollOnNewLog", newValue)}
          />
          <Checkbox
            label="Indent log lines"
            description="Indent each log line according to their respective starting position."
            value={settings.indentLogLines}
            setValue={(newValue) => onSetValue("indentLogLines", newValue)}
          />
          <Checkbox
            label="Hide wrapped text"
            description="Hides text that is meant to be wrapped."
            value={settings.hideWrappedText}
            setValue={(newValue) => onSetValue("hideWrappedText", newValue)}
          />
        </div>
      </div>

      <LogFiles />
    </main>
  );
};

const LogFiles = () => {
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
    <div className="max-w-screen-xl mx-auto pb-12 px-2 sm:px-4 lg:px-8">
      <div
        className="bg-white dark:bg-gray-800 rounded-lg shadow-lg px-2 sm:px-4 pt-3 sm:pt-4"
      >
        <div className="mt-2">
          <h2 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Log files</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Download old log files.
          </p>
        </div>

        {data && data.files.length > 0 ? (
          <section className="py-6 light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
            <ol className="min-w-full relative">
              <li className="hidden sm:grid grid-cols-12 gap-4 mb-2 border-b border-gray-200 dark:border-gray-700">
                <div className="col-span-5 px-2 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Name
                </div>
                <div className="col-span-2 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Size
                </div>
                <div className="col-span-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Last modified
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
    </div>
  );
};

interface LogFilesItemProps {
  file: LogFile;
}

const LogFilesItem = ({ file }: LogFilesItemProps) => {
  return (

    <li className="text-gray-500 dark:text-gray-400">
      <div className="sm:grid grid-cols-12 gap-4 items-center py-2">
        <div className="col-span-5 px-2 py-2 sm:py-0 truncate block sm:text-sm text-md font-medium text-gray-900 dark:text-gray-200">
          <div className="flex justify-between">
            {file.filename}
          </div>
        </div>
        <div className="col-span-2 flex items-center text-sm font-medium text-gray-900 dark:text-gray-200">
          {file.size}
        </div>

        <div className="col-span-4 flex items-center text-sm font-medium text-gray-900 dark:text-gray-200" title={file.updated_at}>
          {simplifyDate(file.updated_at)}
        </div>

        <div className="col-span-1 hidden sm:flex items-center text-sm font-medium text-gray-900 dark:text-white">
          <Link
            className={classNames(
              "text-gray-900 dark:text-gray-300",
              "font-medium group flex rounded-md items-center px-2 py-2 text-sm"
            )}
            title="Download file"
            to={`${baseUrl()}api/logs/files/${file.filename}`}
            target="_blank"
            download={true}
          >
            <DocumentArrowDownIcon className="text-blue-500 w-5 h-5" aria-hidden="true" />
          </Link>
        </div>
      </div>
    </li>
  );
};