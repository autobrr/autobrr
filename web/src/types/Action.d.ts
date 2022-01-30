interface Action {
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
}

interface Filter {
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
    match_release_types: string[];
    quality: string[];
    formats: string[];
    media: string[];
    match_hdr: string[];
    except_hdr: string[];
    log_score: number;
    log: boolean;
    cue: boolean;
    perfect_flac: boolean;
    artists: string;
    albums: string;
    seasons: string;
    episodes: string;
    match_releases: string;
    except_releases: string;
    match_release_groups: string;
    except_release_groups: string;
    match_categories: string;
    except_categories: string;
    tags: string;
    except_tags: string;
    match_uploaders: string;
    except_uploaders: string;
    freeleech: boolean;
    freeleech_percent: string;
    actions: Action[];
    indexers: Indexer[];
}

type ActionType = 'TEST' | 'EXEC' | 'WATCH_FOLDER' | DownloadClientType;
