import { useEffect, useRef, useState } from "react";
import { ExclamationIcon } from "@heroicons/react/solid";

import { APIClient } from "../api/APIClient";
import { Checkbox } from "../components/Checkbox";
import { classNames } from "../utils";
import { SettingsContext } from "../utils/Context";

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
  "ERROR": "text-red-500",
};

export const Logs = () => {
  const [settings, setSettings] = SettingsContext.use();

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const [logs, setLogs] = useState<LogEvent[]>([]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "auto" });
  }

  useEffect(() => {
    const es = APIClient.events.logs();

    es.onmessage = (event) => {
      const newData = JSON.parse(event.data) as LogEvent;
      setLogs((prevState) => [...prevState, newData]);

      if (settings.scrollOnNewLog)
        scrollToBottom();
    }

    return () => es.close();
  }, [setLogs, settings]);

  const onSetValue = (
    key: "scrollOnNewLog" | "indentLogLines" | "hideWrappedText",
    newValue: boolean
  ) => setSettings((prevState) => ({
    ...prevState,
    [key]: newValue
  }));

  return (
    <main>
      <header className="py-10">
        <div className="max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8">
          <h1 className="text-3xl font-bold text-black dark:text-white">Logs</h1>
          <div className="flex mt-4 justify-center">
            <ExclamationIcon
              className="h-5 w-5 text-yellow-400"
              aria-hidden="true"
            />
            <p className="ml-2 text-sm text-gray-800 dark:text-gray-400">This only shows new logs, no history.</p>
          </div>
        </div>
      </header>
      <div className="max-w-screen-xl mx-auto pb-12 px-2 sm:px-4 lg:px-8">
        <div
          className="bg-white dark:bg-gray-800 rounded-lg shadow-lg px-2 sm:px-4 pb-3 sm:pb-4"
        >
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
          <div
            className="overflow-y-auto p-2 rounded-lg min-h-[32rem] lg:min-h-[48rem] min-w-full bg-gray-100 dark:bg-gray-900"
          >
            {logs.map((a, idx) => (
              <div
                key={idx}
                className={classNames(
                  settings.indentLogLines ? "grid justify-start grid-flow-col" : "",
                  settings.hideWrappedText ? "truncate hover:text-ellipsis hover:whitespace-normal" : "",
                )}
              >
                <span
                  className="font-mono text-gray-500 dark:text-gray-600 mr-2 h-full"
                >
                  {a.time}
                </span>
                {a.level in LogColors ? (
                  <span
                    className={classNames(
                      LogColors[a.level as LogLevel],
                      "font-mono font-semibold h-full"
                    )}
                  >
                    {a.level}
                    {' '}
                  </span>
                ) : null}
                <span className="ml-2 text-black dark:text-gray-300">
                  {a.message}
                </span>
              </div>
            ))}
            <div ref={messagesEndRef} />
          </div>
        </div>
      </div>
    </main>
  )
}
