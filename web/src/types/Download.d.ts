/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

type DownloaderType =
  "QBITTORRENT" |
  "DELUGE_V1" |
  "DELUGE_V2" |
  "RTORRENT" |
  "TRANSMISSION" |
  "PORLA" |
  "ARIA2" |
  "RADARR" |
  "SONARR" |
  "LIDARR" |
  "WHISPARR" |
  "WHISPARR_V3" |
  "READARR" |
  "SPORTARR" |
  "SABNZBD" |
  "NZBGET";

// export enum DownloaderTypeEnum {
//     QBITTORRENT = "QBITTORRENT",
//     DELUGE_V1 = "DELUGE_V1",
//     DELUGE_V2 = "DELUGE_V2",
//     RADARR = "RADARR",
//     SONARR = "SONARR",
//     LIDARR = "LIDARR",
//     WHISPARR = "WHISPARR"
// }

interface DownloaderRules {
  enabled: boolean;
  max_active_downloads: number;
  ignore_slow_torrents: boolean;
  ignore_slow_torrents_condition: IgnoreTorrentsCondition;
  download_speed_threshold: number;
  upload_speed_threshold: number;
}

type IgnoreTorrentsCondition = "ALWAYS" | "MAX_DOWNLOADS_REACHED";

interface DownloaderBasicAuth {
  auth: boolean;
  username: string;
  password: string;
}

interface DownloaderSettings {
  apikey?: string;
  basic?: DownloaderBasicAuth;
  rules?: DownloaderRules;
  external_download_client_id?: number;
  external_download_client?: string;
}

interface Downloader {
  id: number;
  name: string;
  type: DownloaderType;
  enabled: boolean;
  host: string;
  port: number;
  tls: boolean;
  tls_skip_verify: boolean;
  username: string;
  password: string;
  settings?: DownloaderSettings;
}

interface ArrTag {
  id: number;
  label: string;
}