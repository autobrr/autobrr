type DownloadClientType =
    'QBITTORRENT' |
    'DELUGE_V1' |
    'DELUGE_V2' |
    'RADARR' |
    'SONARR' |
    'LIDARR';

interface DownloadClient {
    id?: number;
    name: string;
    enabled: boolean;
    host: string;
    port: number;
    ssl: boolean;
    username: string;
    password: string;
    type: DownloadClientType;
    settings: object;
}