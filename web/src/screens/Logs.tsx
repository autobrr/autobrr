import { useEffect, useRef, useState } from "react";
import APIClient from "../api/APIClient";

type LogEvent = {
    time: string;
    level: string;
    message: string;
};

export default function Logs() {
    const [logs, setLogs] = useState<LogEvent[]>([])

    const messagesEndRef: any = useRef(null)

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
        <main className="-mt-48">
            <header className="py-10">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <h1 className="text-3xl font-bold text-white capitalize">Logs</h1>
                </div>
            </header>
            <div className="max-w-7xl mx-auto pb-12 px-2 sm:px-4 lg:px-8">
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow px-2 sm:px-4 py-3 sm:py-4">
                    <div className=" overflow-y-auto p-2 rounded-lg h-96 bg-gray-900">
                        {logs.map((a, idx) => (
                            <p key={idx}>
                                <span className="font-mono text-gray-600 mr-2">{a.time}</span>
                                {a.level === "TRACE" && <span className="font-mono font-semibold text-purple-300">{a.level}</span>}
                                {a.level === "DEBUG" && <span className="font-mono font-semibold text-yellow-500">{a.level}</span>}
                                {a.level === "INFO" && <span className="font-mono font-semibold text-green-500">{a.level} </span>}
                                {a.level === "ERROR" && <span className="font-mono font-semibold text-red-500">{a.level}</span>}
                                <span className="ml-2 text-gray-300">{a.message}</span>
                            </p>
                        ))}
                        <div ref={messagesEndRef} />
                    </div>
                </div>
            </div>
        </main>
    )
}