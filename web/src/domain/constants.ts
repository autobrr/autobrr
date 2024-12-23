/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { MultiSelectOption } from "@components/inputs/select";

export const AnnounceTypeOptions: MultiSelectOption[] = [
  {
    label: "New",
    value: "NEW"
  },
  {
    label: "Checked",
    value: "CHECKED"
  },
  {
    label: "Promo",
    value: "PROMO"
  },
  {
    label: "Promo GP",
    value: "PROMO_GP"
  },
  {
    label: "Resurrected",
    value: "RESURRECTED"
  }
];

export const resolutions = [
  "2160p",
  "1080p",
  "810p",
  "720p",
  "576p",
  "480p",
];

export const RESOLUTION_OPTIONS: MultiSelectOption[] = resolutions.map(r => ({ value: r, label: r, key: r }));

export const codecs = [
  "AV1",
  "AVC",
  "H.264",
  "H.265",
  "HEVC",
  "MPEG-2",
  "VC-1",
  "XviD",
  "x264",
  "x265"
];

export const CODECS_OPTIONS: MultiSelectOption[] = codecs.map(v => ({ value: v, label: v, key: v }));

export const sources = [
  "AHDTV",
  "BD5",
  "BD9",
  "BDRip",
  "BDr",
  "BRRip",
  "BluRay",
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
  "UHD.BluRay",
  "WEB",
  "WEB-DL",
  "WEBRip"
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
  "Anthology",
  "Bootleg",
  "Compilation",
  "Concert Recording",
  "Demo",
  "DJ Mix",
  "EP",
  "Interview",
  "Live album",
  "Mixtape",
  "Remix",
  "Sampler",
  "Single",
  "Soundtrack",
  "Unknown"
];

export const RELEASE_TYPE_MUSIC_OPTIONS: MultiSelectOption[] = releaseTypeMusic.map(v => ({
  value: v,
  label: v,
  key: v
}));

export const originOptions = [
  "P2P",
  "Internal",
  "SCENE",
  "O-SCENE"
];

export const ORIGIN_OPTIONS = originOptions.map(v => ({ value: v, label: v, key: v }));

export const languageOptions = [
  "BALTIC",
  "BRAZiLiAN",
  "BULGARiAN",
  "CHiNESE",
  "CHS",
  "CHT",
  "CZECH",
  "DANiSH",
  "DUBBED",
  "DKSUBS",
  "DUTCH",
  "ENGLiSH",
  "ESTONiAN",
  "FLEMiSH",
  "FiNNiSH",
  "FRENCH",
  "GERMAN",
  "GREEK",
  "HAiTiAN",
  "HARDSUB",
  "Hardcoded",
  "HEBREW",
  "HebSub",
  "HiNDi",
  "HUNGARiAN",
  "iCELANDiC",
  "iTALiAN",
  "JAPANESE",
  "KOREAN",
  "LATiN",
  "MANDARiN",
  "MULTi",
  "MULTILANG",
  "MULTiSUB",
  "MULTiSUBS",
  "NORDiC",
  "NORWEGiAN",
  "POLiSH",
  "PORTUGUESE",
  "ROMANiAN",
  "RUSSiAN",
  "SLOVAK",
  "SPANiSH",
  "SUBBED",
  "SUBFORCED",
  "SUBPACK",
  "SWEDiSH",
  "SYNCED",
  "TURKiSH",
  "UKRAiNiAN",
  "UNSUBBED"
];

export const LANGUAGE_OPTIONS = languageOptions.map(v => ({ value: v, label: v, key: v }));

export interface RadioFieldsetOption {
  label: string;
  description: string;
  value: ActionType;
  type?: string;
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
    label: "rTorrent",
    description: "Add torrents directly to rTorrent",
    value: "RTORRENT"
  },
  {
    label: "Transmission",
    description: "Add torrents directly to Transmission",
    value: "TRANSMISSION"
  },
  {
    label: "Porla",
    description: "Add torrents directly to Porla",
    value: "PORLA"
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
  {
    label: "Readarr",
    description: "Send to Readarr and let it decide",
    value: "READARR"
  },
  {
    label: "SABnzbd",
    description: "Add nzbs directly to SABnzbd",
    value: "SABNZBD",
    type: "nzb"
  }
];

export const ActionTypeOptions: RadioFieldsetOption[] = [
  { label: "Test", description: "A simple action to test a filter.", value: "TEST" },
  { label: "Watch dir", description: "Add filtered torrents to a watch directory", value: "WATCH_FOLDER" },
  { label: "Webhook", description: "Run webhook", value: "WEBHOOK" },
  { label: "Exec", description: "Run a custom command after a filter match", value: "EXEC" },
  { label: "qBittorrent", description: "Add torrents directly to qBittorrent", value: "QBITTORRENT" },
  { label: "Deluge", description: "Add torrents directly to Deluge", value: "DELUGE_V1" },
  { label: "Deluge v2", description: "Add torrents directly to Deluge 2", value: "DELUGE_V2" },
  { label: "rTorrent", description: "Add torrents directly to rTorrent", value: "RTORRENT" },
  { label: "Transmission", description: "Add torrents directly to Transmission", value: "TRANSMISSION" },
  { label: "Porla", description: "Add torrents directly to Porla", value: "PORLA" },
  { label: "Radarr", description: "Send to Radarr and let it decide", value: "RADARR" },
  { label: "Sonarr", description: "Send to Sonarr and let it decide", value: "SONARR" },
  { label: "Lidarr", description: "Send to Lidarr and let it decide", value: "LIDARR" },
  { label: "Whisparr", description: "Send to Whisparr and let it decide", value: "WHISPARR" },
  { label: "Readarr", description: "Send to Readarr and let it decide", value: "READARR" },
  { label: "SABnzbd", description: "Add to SABnzbd", value: "SABNZBD" }
];

export const ActionTypeNameMap: Record<ActionType, string> = {
  "TEST": "Test",
  "WATCH_FOLDER": "Watch folder",
  "WEBHOOK": "Webhook",
  "EXEC": "Exec",
  "DELUGE_V1": "Deluge v1",
  "DELUGE_V2": "Deluge v2",
  "QBITTORRENT": "qBittorrent",
  "RTORRENT": "rTorrent",
  "TRANSMISSION": "Transmission",
  "PORLA": "Porla",
  "RADARR": "Radarr",
  "SONARR": "Sonarr",
  "LIDARR": "Lidarr",
  "WHISPARR": "Whisparr",
  "READARR": "Readarr",
  "SABNZBD": "SABnzbd"
} as const;

export const DOWNLOAD_CLIENTS = [
  "QBITTORRENT",
  "DELUGE_V1",
  "DELUGE_V2",
  "RTORRENT",
  "TRANSMISSION",
  "PORLA",
  "RADARR",
  "SONARR",
  "LIDARR",
  "WHISPARR",
  "READARR",
  "SABNZBD"
];

export const ActionContentLayoutOptions: SelectGenericOption<ActionContentLayout>[] = [
  { label: "Original", description: "Original", value: "ORIGINAL" },
  { label: "Create subfolder", description: "Create subfolder", value: "SUBFOLDER_CREATE" },
  { label: "Don't create subfolder", description: "Don't create subfolder", value: "SUBFOLDER_NONE" }
];

export const ActionPriorityOptions: SelectGenericOption<ActionPriorityLayout>[] = [
  { label: "Top of queue", description: "Top of queue", value: "MAX" },
  { label: "Bottom of queue", description: "Bottom of queue", value: "MIN" },
  { label: "Disabled", description: "Disabled", value: "" }
];

export const ActionRtorrentRenameOptions: SelectGenericOption<ActionContentLayout>[] = [
  { label: "No", description: "No", value: "ORIGINAL" },
  { label: "Yes", description: "Yes", value: "SUBFOLDER_NONE" }
];

export interface OptionBasic {
  label: string;
  value: string;
}

export interface OptionBasicTyped<T> {
  label: string;
  value: T;
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

export const ListTypeOptions: OptionBasicTyped<ListType>[] = [
  {
    label: "Sonarr",
    value: "SONARR"
  },
  {
    label: "Radarr",
    value: "RADARR"
  },
  {
    label: "Lidarr",
    value: "LIDARR"
  },
  {
    label: "Readarr",
    value: "READARR"
  },
  {
    label: "Whisparr",
    value: "WHISPARR"
  },
  {
    label: "MDBList",
    value: "MDBLIST"
  },
  {
    label: "Trakt",
    value: "TRAKT"
  },
  {
    label: "Plaintext",
    value: "PLAINTEXT"
  },
  {
    label: "Steam",
    value: "STEAM"
  },
  {
    label: "Metacritic",
    value: "METACRITIC"
  },
];

export const NotificationTypeOptions: OptionBasicTyped<NotificationType>[] = [
  {
    label: "Discord",
    value: "DISCORD"
  },
  {
    label: "Gotify",
    value: "GOTIFY"
  },
  {
    label: "LunaSea",
    value: "LUNASEA"
  },
  {
    label: "Notifiarr",
    value: "NOTIFIARR"
  },
  {
    label: "Ntfy",
    value: "NTFY"
  },
  {
    label: "Pushover",
    value: "PUSHOVER"
  },
  {
    label: "Shoutrrr",
    value: "SHOUTRRR"
  },
  {
    label: "Telegram",
    value: "TELEGRAM"
  },
];

export const IrcAuthMechanismTypeOptions: OptionBasicTyped<IrcAuthMechanism>[] = [
  {
    label: "None",
    value: "NONE"
  },
  {
    label: "SASL (plain)",
    value: "SASL_PLAIN"
  },
  {
    label: "NickServ",
    value: "NICKSERV"
  }
];

export const downloadsPerUnitOptions: OptionBasic[] = [
  {
    label: "Select",
    value: ""
  },
  {
    label: "HOUR",
    value: "HOUR"
  },
  {
    label: "DAY",
    value: "DAY"
  },
  {
    label: "WEEK",
    value: "WEEK"
  },
  {
    label: "MONTH",
    value: "MONTH"
  },
  {
    label: "EVER",
    value: "EVER"
  }
];

export const DownloadRuleConditionOptions: OptionBasic[] = [
  {
    label: "Always",
    value: "ALWAYS"
  },
  {
    label: "Max downloads reached",
    value: "MAX_DOWNLOADS_REACHED"
  }
];

export const DownloadClientAuthType: OptionBasic[] = [
  {
    label: "None",
    value: "NONE"
  },
  {
    label: "Basic Auth",
    value: "BASIC_AUTH"
  },
  {
    label: "Digest Auth",
    value: "DIGEST_AUTH"
  }
];

const logLevel = ["DEBUG", "INFO", "WARN", "ERROR", "TRACE"] as const;

export const LogLevelOptions = logLevel.map(v => ({ value: v, label: v, key: v }));

export interface SelectOption {
  label: string;
  description: string;
  value: NotificationEvent;
}

export interface SelectGenericOption<T> {
  label: string;
  description: string;
  value: T;
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
  },
  {
    label: "IRC Disconnected",
    value: "IRC_DISCONNECTED",
    description: "Unexpectedly disconnected from irc network"
  },
  {
    label: "IRC Reconnected",
    value: "IRC_RECONNECTED",
    description: "Reconnected to irc network after error"
  },
  {
    label: "New update",
    value: "APP_UPDATE_AVAILABLE",
    description: "Get notified on updates"
  }
];

export const FeedDownloadTypeOptions: OptionBasicTyped<FeedDownloadType>[] = [
  {
    label: "Magnet",
    value: "MAGNET"
  },
  {
    label: "Torrent",
    value: "TORRENT"
  }
];

export const tagsMatchLogicOptions: OptionBasic[] = [
  {
    label: "any",
    value: "ANY"
  },
  {
    label: "all",
    value: "ALL"
  }
];

export const ExternalFilterTypeOptions: RadioFieldsetOption[] = [
  { label: "Exec", description: "Run a custom command", value: "EXEC" },
  { label: "Webhook", description: "Run webhook", value: "WEBHOOK" }
];

export const ExternalFilterTypeNameMap = {
  "EXEC": "Exec",
  "WEBHOOK": "Webhook"
};

export const ExternalFilterWebhookMethodOptions: OptionBasicTyped<WebhookMethod>[] = [
  { label: "GET", value: "GET" },
  { label: "POST", value: "POST" },
  { label: "PUT", value: "PUT" },
  { label: "PATCH", value: "PATCH" },
  { label: "DELETE", value: "DELETE" }
];

export const ProxyTypeOptions: OptionBasicTyped<ProxyType>[] = [
  {
    label: "SOCKS5",
    value: "SOCKS5"
  },
];

export const ListsTraktOptions: OptionBasic[] = [
  {
    label: "Anticipated TV",
    value: "https://api.autobrr.com/lists/trakt/anticipated-tv"
  },
  {
    label: "Popular TV",
    value: "https://api.autobrr.com/lists/trakt/popular-tv"
  },
  {
    label: "Upcoming Movies",
    value: "https://api.autobrr.com/lists/trakt/upcoming-movies"
  },
  {
    label: "Upcoming BluRay",
    value: "https://api.autobrr.com/lists/trakt/upcoming-bluray"
  },
  {
    label: "Popular TV",
    value: "https://api.autobrr.com/lists/trakt/popular-tv"
  },
  {
    label: "Steven Lu",
    value: "https://api.autobrr.com/lists/stevenlu"
  },
];

export const ListsMetacriticOptions: OptionBasic[] = [
  {
    label: "Upcoming Albums",
    value: "https://api.autobrr.com/lists/metacritic/upcoming-albums"
  },
  {
    label: "New Albums",
    value: "https://api.autobrr.com/lists/metacritic/new-albums"
  }
];

export const ListsMDBListOptions: OptionBasic[] = [
  {
    label: "Latest TV Shows",
    value: "https://mdblist.com/lists/garycrawfordgc/latest-tv-shows/json"
  },
];
