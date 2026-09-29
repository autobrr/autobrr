// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"github.com/autobrr/autobrr/internal/domain"
)

// arrDownloadClient returns the download client id and name to push to an arr app.
// An action override replaces the client level pair as a whole, since arr apps that validate
// pushes reject an id and a name that point at different download clients.
func arrDownloadClient(settings domain.DownloaderSettings, action *domain.Action) (int, string) {
	if action.ExternalDownloadClientID > 0 || action.ExternalDownloadClient != "" {
		return int(action.ExternalDownloadClientID), action.ExternalDownloadClient
	}

	return settings.ExternalDownloadClientId, settings.ExternalDownloadClient
}
