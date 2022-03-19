import { useEffect, useRef, useState } from "react";
import { APIClient } from "../api/APIClient";
import {ExclamationIcon} from "@heroicons/react/solid";

type LogEvent = {
    time: string;
    level: string;
    message: string;
};

export const Logs = () => {
    const messagesEndRef = useRef<HTMLDivElement>(null);
    const [logs, setLogs] = useState<LogEvent[]>([]);

    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: "auto" })
    }

    useEffect(() => {
        const es = APIClient.events.logs()

        es.onmessage = (event) => {
            const d = JSON.parse(event.data) as LogEvent;
            setLogs(prevState => ([...prevState, d]));
            scrollToBottom();
        }
        return () => {
            es.close();
        }
    }, [setLogs]);

    return (
        <main>
            <header className="py-10">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                 <h1 className="text-3xl font-bold text-black dark:text-white capitalize">Logs</h1>
                    <div className="flex mt-4 justify-center">
                        <ExclamationIcon
                            className="h-5 w-5 text-yellow-400"
                            aria-hidden="true"
                        />
                        <p className="ml-2 text-sm text-gray-800 dark:text-gray-400">This only shows new logs, no history</p>
                    </div>
                </div>
            </header>
            <div className="max-w-7xl mx-auto pb-12 px-2 sm:px-4 lg:px-8">
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg px-2 sm:px-4 py-3 sm:py-4">
                    <div className="overflow-y-auto p-2 rounded-lg min-h-[32rem] lg:min-h-[48rem] min-w-full bg-gray-100 dark:bg-gray-900">
                        {logs.map((a, idx) => (
                            <p key={idx}>
                                <span className="font-mono text-gray-500 dark:text-gray-600 mr-2">{a.time}</span>
                                {a.level === "TRACE" && <span className="font-mono font-semibold text-purple-300">{a.level}</span>}
                                {a.level === "DEBUG" && <span className="font-mono font-semibold text-yellow-500">{a.level}</span>}
                                {a.level === "INFO" && <span className="font-mono font-semibold text-green-500">{a.level} </span>}
                                {a.level === "ERROR" && <span className="font-mono font-semibold text-red-500">{a.level}</span>}
                                <span className="ml-2 text-black dark:text-gray-300">{a.message}</span>
                            </p>
                        ))}
                        <div ref={messagesEndRef} />
                    </div>
                </div>
            </div>
        </main>
    )
}
