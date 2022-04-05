type DownloadClientType =
    'QBITTORRENT' |
    'DELUGE_V1' |
    'DELUGE_V2' |
    'RADARR' |
    'SONARR' |
    'LIDARR' |
    'WHISPARR';

interface DownloadClientRules {
  enabled: boolean;
  max_active_downloads: number;
  ignore_slow_torrents: boolean;
  download_speed_threshold: number;
}

interface DownloadClientBasicAuth {
  auth: boolean;
  username: string;
  password: string;
}

interface DownloadClientSettings {
  apikey?: string;
  basic?: DownloadClientBasicAuth;
  rules?: DownloadClientRules;
}

interface DownloadClient {
  id?: number;
  id: number;
  name: string;
  type: DownloadClientType;
  enabled: boolean;
  host: string;
  port: number;
  tls: boolean;
  tls_skip_verify: boolean;
  username: string;
  password: string;
  settings?: DownloadClientSettings;
}