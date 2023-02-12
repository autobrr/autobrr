type LogLevel = "DEBUG" | "INFO" | "WARN" | "ERROR" | "TRACE";

interface Config {
  host: string;
  port: number;
  log_level: LogLevel;
  log_path: string;
  log_max_size: number;
  log_max_backups: number;
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
  check_for_updates?: boolean;
}

interface LogFile {
  filename: string;
  size: string;
  updated_at: string;
}

interface LogFileResponse {
  files: LogFile[];
  count: number;
}