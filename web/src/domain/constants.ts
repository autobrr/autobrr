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
    "XviD"
];

export const CODECS_OPTIONS = codecs.map(v => ({ value: v, label: v, key: v}));

export const sources = [
    "BD5",
    "BD9",
    "BDr",
    "BDRip",
    "BluRay",
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
    "WEB-DL",
    "Webrip"
];

export const SOURCES_OPTIONS = sources.map(v => ({ value: v, label: v, key: v}));

export const containers = [
    "avi",
    "mp4",
    "mkv",
];

export const CONTAINER_OPTIONS = containers.map(v => ({ value: v, label: v, key: v}));

export interface radioFieldsetOption {
    label: string;
    description: string;
    value: string;
}

export const DownloadClientTypeOptions: radioFieldsetOption[] = [
    {
        label: "qBittorrent",
        description: "Add torrents directly to qBittorrent",
        value: DOWNLOAD_CLIENT_TYPES.qBittorrent
    },
    {label: "Deluge", description: "Add torrents directly to Deluge", value: DOWNLOAD_CLIENT_TYPES.Deluge},
];
