/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

interface UpdateAvailableResponse {
    id: number;
    name: string;
}

export interface GithubRelease {
    id:               number;
    node_id:          string;
    url:              string;
    html_url:         string;
    tag_name:         string;
    target_commitish: string;
    name:             string;
    body:             string;
    created_at:       Date;
    published_at:     Date;
    author:           GithubAuthor;
    assets:           GitHubReleaseAsset[];
}

export interface GitHubReleaseAsset {
    url:                  string;
    id:                   number;
    node_id:              string;
    name:                 string;
    label:                string;
    uploader:             GithubAuthor;
    content_type:         string;
    state:                string;
    size:                 number;
    download_count:       number;
    created_at:           Date;
    updated_at:           Date;
    browser_download_url: string;
}

export interface GithubAuthor {
    login:       string;
    id:          number;
    node_id:     string;
    avatar_url:  string;
    gravatar_id: string;
    url:         string;
    html_url:    string;
    type:        string;
}