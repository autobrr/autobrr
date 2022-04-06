export const resolutions = [
    "2160p",
    "1080p",
    "1080i",
    "810p",
    "720p",
    "576p",
    "480p",
    "480i"
];

export const RESOLUTION_OPTIONS = resolutions.map(r => ({ value: r, label: r, key: r}));

export const codecs = [
    "AVC",
    "Remux",
    "h.264 Remux",
    "h.265 Remux",
    "HEVC",
    "VC-1",
    "VC-1 Remux",
    "h264",
    "h265",
    "x264",
    "x265",
    "h264 10-bit",
    "h265 10-bit",
    "x264 10-bit",
    "x265 10-bit",
    "XviD"
];

export const CODECS_OPTIONS = codecs.map(v => ({ value: v, label: v, key: v}));

export const sources = [
    "WEB-DL",
    "BluRay",
    "BD5",
    "BD9",
    "BDr",
    "BDRip",
    "BRRip",
    "CAM",
    "DVDR",
    "DVDRip",
    "DVDScr",
    "HDCAM",
    "HDDVD",
    "HDDVDRip",
    "HDTS",
    "HDTV",
    "Mixed",
    "SiteRip",
    "Webrip",
];

export const SOURCES_OPTIONS = sources.map(v => ({ value: v, label: v, key: v}));

export const containers = [
    "avi",
    "mp4",
    "mkv",
];

export const CONTAINER_OPTIONS = containers.map(v => ({ value: v, label: v, key: v}));

export const hdr = [
    "HDR",
    "HDR10",
    "HDR10+",
    "DV",
    "DV HDR",
    "DV HDR10",
    "DV HDR10+",
    "DoVi",
    "Dolby Vision",
];

export const HDR_OPTIONS = hdr.map(v => ({ value: v, label: v, key: v}));


export const formatMusic = [
    "MP3",
    "FLAC",
    "Ogg Vorbis",
    "Ogg",
    "AAC",
    "AC3",
    "DTS",
];

export const FORMATS_OPTIONS = formatMusic.map(r => ({ value: r, label: r, key: r}));

export const sourcesMusic = [
    "CD",
    "WEB",
    "DVD",
    "Vinyl",
    "Soundboard",
    "DAT",
    "Cassette",
    "Blu-Ray",
    "SACD",
];

export const SOURCES_MUSIC_OPTIONS = sourcesMusic.map(v => ({ value: v, label: v, key: v}));

export const qualityMusic = [
    "192",
    "256",
    "320",
    "APS (VBR)",
    "APX (VBR)",
    "V2 (VBR)",
    "V1 (VBR)",
    "V0 (VBR)",
    "Lossless",
    "24bit Lossless",
];

export const QUALITY_MUSIC_OPTIONS = qualityMusic.map(v => ({ value: v, label: v, key: v}));

export const releaseTypeMusic = [
    "Album",
    "Single",
    "EP",
    "Soundtrack",
    "Anthology",
    "Compilation",
    "Live album",
    "Remix",
    "Bootleg",
    "Interview",
    "Mixtape",
    "Demo",
    "Concert Recording",
    "DJ Mix",
    "Unkown",
];

export const RELEASE_TYPE_MUSIC_OPTIONS = releaseTypeMusic.map(v => ({ value: v, label: v, key: v}));

export interface RadioFieldsetOption {
    label: string;
    description: string;
    value: ActionType;
}

export const DownloadClientTypeOptions: RadioFieldsetOption[] = [
    {
        label: "qBittorrent",
        description: "Add torrents directly to qBittorrent",
        value: "QBITTORRENT"
    },
    {
        label: "Deluge",
        description: "Add torrents directly to Deluge",
        value: "DELUGE_V1"
    },
    {
        label: "Deluge 2",
        description: "Add torrents directly to Deluge 2",
        value: "DELUGE_V2"
    },
    {
        label: "Radarr",
        description: "Send to Radarr and let it decide",
        value: "RADARR"
    },
    {
        label: "Sonarr",
        description: "Send to Sonarr and let it decide",
        value: "SONARR"
    },
    {
        label: "Lidarr",
        description: "Send to Lidarr and let it decide",
        value: "LIDARR"
    },
    {
        label: "Whisparr",
        description: "Send to Whisparr and let it decide",
        value: "WHISPARR"
    },
];

export const DownloadClientTypeNameMap: Record<DownloadClientType | string, string> = {
    "DELUGE_V1": "Deluge v1",
    "DELUGE_V2": "Deluge v2",
    "QBITTORRENT": "qBittorrent",
    "RADARR": "Radarr",
    "SONARR": "Sonarr",
    "LIDARR": "Lidarr",
    "WHISPARR": "Whisparr",
};

export const ActionTypeOptions: RadioFieldsetOption[] = [
    {label: "Test", description: "A simple action to test a filter.", value: "TEST"},
    {label: "Watch dir", description: "Add filtered torrents to a watch directory", value: "WATCH_FOLDER"},
    {label: "Webhook", description: "Run webhook", value: "WEBHOOK"},
    {label: "Exec", description: "Run a custom command after a filter match", value: "EXEC"},
    {label: "qBittorrent", description: "Add torrents directly to qBittorrent", value: "QBITTORRENT"},
    {label: "Deluge", description: "Add torrents directly to Deluge", value: "DELUGE_V1"},
    {label: "Deluge v2", description: "Add torrents directly to Deluge 2", value: "DELUGE_V2"},
    {label: "Radarr", description: "Send to Radarr and let it decide", value: "RADARR"},
    {label: "Sonarr", description: "Send to Sonarr and let it decide", value: "SONARR"},
    {label: "Lidarr", description: "Send to Lidarr and let it decide", value: "LIDARR"},
    {label: "Whisparr", description: "Send to Whisparr and let it decide", value: "WHISPARR"},
];

export const ActionTypeNameMap = {
    "TEST": "Test",
    "WATCH_FOLDER": "Watch folder",
    "WEBHOOK": "Webhook",
    "EXEC": "Exec",
    "DELUGE_V1": "Deluge v1",
    "DELUGE_V2": "Deluge v2",
    "QBITTORRENT": "qBittorrent",
    "RADARR": "Radarr",
    "SONARR": "Sonarr",
    "LIDARR": "Lidarr",
    "WHISPARR": "Whisparr",
};

export const PushStatusOptions: any[] = [
    {
        label: "Rejected",
        value: "PUSH_REJECTED",
    },
    {
        label: "Approved",
        value: "PUSH_APPROVED"
    },
    {
        label: "Error",
        value: "PUSH_ERROR"
    },
];

export const NotificationTypeOptions: any[] = [
    {
        label: "Discord",
        value: "DISCORD",
    },
];

export interface SelectOption {
    label: string;
    description: string;
    value: any;
}

export const EventOptions: SelectOption[] = [
    {
        label: "Push Rejected",
        value: "PUSH_REJECTED",
        description: "On push rejected for the arrs or download client",
    },
    {
        label: "Push Approved",
        value: "PUSH_APPROVED",
        description: "On push approved for the arrs or download client",
    },
    {
        label: "Push Error",
        value: "PUSH_ERROR",
        description: "On push error for the arrs or download client",
    },
];
