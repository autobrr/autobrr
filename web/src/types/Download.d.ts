/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

type DownloadClientType =
  "QBITTORRENT" |
  "DELUGE_V1" |
  "DELUGE_V2" |
  "RTORRENT" |
  "TRANSMISSION" |
  "PORLA" |
  "RADARR" |
  "SONARR" |
  "LIDARR" |
  "WHISPARR" |
  "READARR" |
  "SABNZBD";

// export enum DownloadClientTypeEnum {
//     QBITTORRENT = "QBITTORRENT",
//     DELUGE_V1 = "DELUGE_V1",
//     DELUGE_V2 = "DELUGE_V2",
//     RADARR = "RADARR",
//     SONARR = "SONARR",
//     LIDARR = "LIDARR",
//     WHISPARR = "WHISPARR"
// }

interface DownloadClientRules {
  enabled: boolean;
  max_active_downloads: number;
  ignore_slow_torrents: boolean;
  ignore_slow_torrents_condition: IgnoreTorrentsCondition;
  download_speed_threshold: number;
  upload_speed_threshold: number;
}

type IgnoreTorrentsCondition = "ALWAYS" | "MAX_DOWNLOADS_REACHED";

interface DownloadClientBasicAuth {
  auth: boolean;
  username: string;
  password: string;
}

interface DownloadClientSettings {
  apikey?: string;
  basic?: DownloadClientBasicAuth;
  rules?: DownloadClientRules;
  external_download_client_id?: number;
  external_download_client?: string;
}

interface DownloadClient {
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

interface ArrTag {
  id: number;
  label: string;
}