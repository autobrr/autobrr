import { MultiSelectOption } from "../components/inputs/select";

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

export const RESOLUTION_OPTIONS: MultiSelectOption[] = resolutions.map(r => ({ value: r, label: r, key: r }));

export const codecs = [
  "HEVC",
  "H.264",
  "H.265",
  "x264",
  "x265",
  "AVC",
  "VC-1",
  "AV1",
  "XviD"
];

export const CODECS_OPTIONS: MultiSelectOption[] = codecs.map(v => ({ value: v, label: v, key: v }));

export const sources = [
  "BluRay",
  "UHD.BluRay",
  "WEB-DL",
  "WEB",
  "WEBRip",
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
  "SiteRip"
];

export const SOURCES_OPTIONS: MultiSelectOption[] = sources.map(v => ({ value: v, label: v, key: v }));

export const containers = [
  "avi",
  "mp4",
  "mkv"
];

export const CONTAINER_OPTIONS: MultiSelectOption[] = containers.map(v => ({ value: v, label: v, key: v }));

export const hdr = [
  "HDR",
  "HDR10",
  "HDR10+",
  "HLG",
  "DV",
  "DV HDR",
  "DV HDR10",
  "DV HDR10+",
  "DoVi",
  "Dolby Vision"
];

export const HDR_OPTIONS: MultiSelectOption[] = hdr.map(v => ({ value: v, label: v, key: v }));

export const quality_other = [
  "REMUX",
  "HYBRID",
  "REPACK"
];

export const OTHER_OPTIONS = quality_other.map(v => ({ value: v, label: v, key: v }));

export const formatMusic = [
  "MP3",
  "FLAC",
  "Ogg Vorbis",
  "Ogg",
  "AAC",
  "AC3",
  "DTS"
];

export const FORMATS_OPTIONS: MultiSelectOption[] = formatMusic.map(r => ({ value: r, label: r, key: r }));

export const sourcesMusic = [
  "CD",
  "WEB",
  "DVD",
  "Vinyl",
  "Soundboard",
  "DAT",
  "Cassette",
  "Blu-Ray",
  "SACD"
];

export const SOURCES_MUSIC_OPTIONS: MultiSelectOption[] = sourcesMusic.map(v => ({ value: v, label: v, key: v }));

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
  "24bit Lossless"
];

export const QUALITY_MUSIC_OPTIONS: MultiSelectOption[] = qualityMusic.map(v => ({ value: v, label: v, key: v }));

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
  "Unknown"
];

export const RELEASE_TYPE_MUSIC_OPTIONS: MultiSelectOption[] = releaseTypeMusic.map(v => ({ value: v, label: v, key: v }));

export const originOptions = [
  "P2P",
  "Internal",
  "SCENE",
  "O-SCENE"
];

export const ORIGIN_OPTIONS = originOptions.map(v => ({ value: v, label: v, key: v }));

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
  }
];

export const DownloadClientTypeNameMap: Record<DownloadClientType | string, string> = {
  "DELUGE_V1": "Deluge v1",
  "DELUGE_V2": "Deluge v2",
  "QBITTORRENT": "qBittorrent",
  "RADARR": "Radarr",
  "SONARR": "Sonarr",
  "LIDARR": "Lidarr",
  "WHISPARR": "Whisparr"
};

export const ActionTypeOptions: RadioFieldsetOption[] = [
  { label: "Test", description: "A simple action to test a filter.", value: "TEST" },
  { label: "Watch dir", description: "Add filtered torrents to a watch directory", value: "WATCH_FOLDER" },
  { label: "Webhook", description: "Run webhook", value: "WEBHOOK" },
  { label: "Exec", description: "Run a custom command after a filter match", value: "EXEC" },
  { label: "qBittorrent", description: "Add torrents directly to qBittorrent", value: "QBITTORRENT" },
  { label: "Deluge", description: "Add torrents directly to Deluge", value: "DELUGE_V1" },
  { label: "Deluge v2", description: "Add torrents directly to Deluge 2", value: "DELUGE_V2" },
  { label: "Radarr", description: "Send to Radarr and let it decide", value: "RADARR" },
  { label: "Sonarr", description: "Send to Sonarr and let it decide", value: "SONARR" },
  { label: "Lidarr", description: "Send to Lidarr and let it decide", value: "LIDARR" },
  { label: "Whisparr", description: "Send to Whisparr and let it decide", value: "WHISPARR" }
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
  "WHISPARR": "Whisparr"
};

export interface OptionBasic {
    label: string;
    value: string;
}

export const PushStatusOptions: OptionBasic[] = [
  {
    label: "Rejected",
    value: "PUSH_REJECTED"
  },
  {
    label: "Approved",
    value: "PUSH_APPROVED"
  },
  {
    label: "Error",
    value: "PUSH_ERROR"
  }
];

export const NotificationTypeOptions: OptionBasic[] = [
  {
    label: "Discord",
    value: "DISCORD"
  }
];

export interface SelectOption {
    label: string;
    description: string;
    value: string;
}

export const EventOptions: SelectOption[] = [
  {
    label: "Push Rejected",
    value: "PUSH_REJECTED",
    description: "On push rejected for the arrs or download client"
  },
  {
    label: "Push Approved",
    value: "PUSH_APPROVED",
    description: "On push approved for the arrs or download client"
  },
  {
    label: "Push Error",
    value: "PUSH_ERROR",
    description: "On push error for the arrs or download client"
  }
];
