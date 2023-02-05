interface Config {
    host: string;
    port: number;
    log_level: string;
    log_path: string;
    base_url: string;
    check_for_updates: boolean;
    version: string;
    commit: string;
    date: string;
}

interface ConfigUpdate {
    host?: string;
    port?: number;
    log_level?: string;
    log_path?: string;
    base_url?: string;
    check_for_updates: boolean;
}
