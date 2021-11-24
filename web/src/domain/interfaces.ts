export interface APP {
    baseUrl: string;
}

export interface Action {
    id: number;
    name: string;
    enabled: boolean;
    type: ActionType;
    exec_cmd: string;
    exec_args: string;
    watch_folder: string;
    category: string;
    tags: string;
    label: string;
    save_path: string;
    paused: boolean;
    ignore_rules: boolean;
    limit_upload_speed: number;
    limit_download_speed: number;
    client_id: number;
    filter_id: number;
    // settings: object;
}

export interface Indexer {
    id: number;
    name: string;
    identifier: string;
    enabled: boolean;
    settings: object | any;
}

export interface IndexerSchema {
    // id: number;
    name: string;
    identifier: string;
    description: string;
    language: string;
    privacy: string;
    protocol: string;
    urls: string[];
    settings: IndexerSchemaSettings[];
    irc: IndexerSchemaIRC;
}


export interface IndexerSchemaSettings {
    name: string;
    type: string;
    required: boolean;
    label: string;
    help: string;
    description: string;
    default: string;
}

export interface IndexerSchemaIRC {
    network: string;
    server: string;
    port: number;
    tls: boolean;
    nickserv: boolean;
    announcers: string[];
    channels: string[];
    invite: string[];
    invite_command: string;
    settings: IndexerSchemaSettings[];
}

export interface Filter {
    id: number;
    name: string;
    enabled: boolean;
    shows: string;
    min_size: string;
    max_size: string;
    match_sites: string[];
    except_sites: string[];
    delay: number;
    years: string;
    resolutions: string[];
    sources: string[];
    codecs: string[];
    containers: string[];
    seasons: string;
    episodes: string;
    match_releases: string;
    except_releases: string;
    match_release_groups: string;
    except_release_groups: string;
    match_categories: string;
    except_categories: string;
    match_tags: string;
    except_tags: string;
    match_uploaders: string;
    except_uploaders: string;
    freeleech: boolean;
    freeleech_percent: string;
    actions: Action[];
    indexers: Indexer[];
}

export type ActionType = 'TEST' | 'EXEC' | 'WATCH_FOLDER' | 'QBITTORRENT' | 'DELUGE_V1' | 'DELUGE_V2' | 'RADARR' | 'SONARR' | 'LIDARR';
export const ACTIONTYPES: ActionType[] = ['TEST', 'EXEC' , 'WATCH_FOLDER' , 'QBITTORRENT' , 'DELUGE_V1', 'DELUGE_V2', 'RADARR', 'SONARR', 'LIDARR'];


export type DownloadClientType = 'QBITTORRENT' | 'DELUGE_V1' | 'DELUGE_V2' | 'RADARR' | 'SONARR' | 'LIDARR';

export enum DOWNLOAD_CLIENT_TYPES {
    qBittorrent = 'QBITTORRENT',
    DelugeV1 = 'DELUGE_V1',
    DelugeV2 = 'DELUGE_V2',
    Radarr = 'RADARR',
    Sonarr = 'SONARR',
    Lidarr = 'LIDARR'
}

export interface DownloadClient {
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

export interface NickServ {
    account: string;
    password: string;
}

export interface Network {
    id?: number;
    name: string;
    enabled: boolean;
    server: string;
    port: number;
    tls: boolean;
    invite_command: string;
    nickserv: {
        account: string;
        password: string;
    }
    channels: Channel[];
    settings: object;
}

export interface Channel {
    name: string;
}

export interface SASL {
    mechanism: string;
    plain: {
        username: string;
        password: string;
    }
}

export interface Config {
    host: string;
    port: number;
    log_level: string;
    log_path: string;
    base_url: string;
}

export interface Release {
    id: number;
    filter_status: string;
    push_status: string;
    rejections: string[];
    indexer: string;
    filter: string;
    protocol: string;
    title: string;
    size: number;
    raw: string;
    timestamp: Date
}

export interface ReleaseFindResponse {
    data: Release[];
    next_cursor: number;
}

export interface ReleaseStats {
    total_count: number;
    filtered_count: number;
    filter_rejected_count: number;
    push_approved_count: number;
    push_rejected_count: number;
}