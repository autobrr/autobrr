import {DOWNLOAD_CLIENT_TYPES} from "./interfaces";

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
    "Webrip"
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

export interface radioFieldsetOption {
    label: string;
    description: string;
    value: string;
}

export const DownloadClientTypeOptions: radioFieldsetOption[] = [
    {label: "qBittorrent", description: "Add torrents directly to qBittorrent", value: DOWNLOAD_CLIENT_TYPES.qBittorrent},
    {label: "Deluge", description: "Add torrents directly to Deluge", value: DOWNLOAD_CLIENT_TYPES.DelugeV1},
    {label: "Deluge 2", description: "Add torrents directly to Deluge 2", value: DOWNLOAD_CLIENT_TYPES.DelugeV2},
    {label: "Radarr", description: "Send to Radarr and let it decide", value: DOWNLOAD_CLIENT_TYPES.Radarr},
    {label: "Sonarr", description: "Send to Sonarr and let it decide", value: DOWNLOAD_CLIENT_TYPES.Sonarr},
    {label: "Lidarr", description: "Send to Lidarr and let it decide", value: DOWNLOAD_CLIENT_TYPES.Lidarr},
];
export const DownloadClientTypeNameMap = {
    "DELUGE_V1": "Deluge v1",
    "DELUGE_V2": "Deluge v2",
    "QBITTORRENT": "qBittorrent",
    "RADARR": "Radarr",
    "SONARR": "Sonarr",
    "LIDARR": "Lidarr",
};

export const ActionTypeOptions: radioFieldsetOption[] = [
    {label: "Test", description: "A simple action to test a filter.", value: "TEST"},
    {label: "Watch dir", description: "Add filtered torrents to a watch directory", value: "WATCH_FOLDER"},
    {label: "Exec", description: "Run a custom command after a filter match", value: "EXEC"},
    {label: "qBittorrent", description: "Add torrents directly to qBittorrent", value: "QBITTORRENT"},
    {label: "Deluge", description: "Add torrents directly to Deluge", value: "DELUGE_V1"},
    {label: "Deluge v2", description: "Add torrents directly to Deluge 2", value: "DELUGE_V2"},
    {label: "Radarr", description: "Send to Radarr and let it decide", value: DOWNLOAD_CLIENT_TYPES.Radarr},
    {label: "Sonarr", description: "Send to Sonarr and let it decide", value: DOWNLOAD_CLIENT_TYPES.Sonarr},
    {label: "Lidarr", description: "Send to Lidarr and let it decide", value: DOWNLOAD_CLIENT_TYPES.Lidarr},
];

export const ActionTypeNameMap = {
    "TEST": "Test",
    "WATCH_FOLDER": "Watch folder",
    "EXEC": "Exec",
    "DELUGE_V1": "Deluge v1",
    "DELUGE_V2": "Deluge v2",
    "QBITTORRENT": "qBittorrent",
    "RADARR": "Radarr",
    "SONARR": "Sonarr",
    "LIDARR": "Lidarr",
};
