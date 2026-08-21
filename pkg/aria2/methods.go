// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package aria2

import (
	"context"
	"encoding/base64"

	"github.com/autobrr/autobrr/pkg/errors"
)

// activeStatusKeys limits what tellActive returns to the fields needed to tell
// downloading torrents from seeding ones.
var activeStatusKeys = []string{"gid", "status", "totalLength", "completedLength"}

// GetVersion returns the version and enabled features of the daemon.
func (c *Client) GetVersion(ctx context.Context) (*Version, error) {
	response, err := c.call(ctx, "aria2.getVersion")
	if err != nil {
		return nil, err
	}

	var version *Version
	if err := response.GetObject(&version); err != nil {
		return nil, errors.Wrap(err, "could not decode version response")
	}

	return version, nil
}

// AddURI queues a magnet link or http(s) uri and returns the download gid.
func (c *Client) AddURI(ctx context.Context, uris []string, options Options) (string, error) {
	response, err := c.call(ctx, "aria2.addUri", uris, orEmpty(options))
	if err != nil {
		return "", err
	}

	return decodeGid(response.Result)
}

// AddTorrent queues the contents of a torrent file and returns the download gid.
func (c *Client) AddTorrent(ctx context.Context, torrent []byte, options Options) (string, error) {
	// the empty slice is the optional list of web seeds, it has to be sent so
	// the options land on the right positional parameter
	response, err := c.call(ctx, "aria2.addTorrent", base64.StdEncoding.EncodeToString(torrent), []string{}, orEmpty(options))
	if err != nil {
		return "", err
	}

	return decodeGid(response.Result)
}

// TellActive returns the downloads the daemon currently considers active.
func (c *Client) TellActive(ctx context.Context) ([]Status, error) {
	response, err := c.call(ctx, "aria2.tellActive", activeStatusKeys)
	if err != nil {
		return nil, err
	}

	var statuses []Status
	if err := response.GetObject(&statuses); err != nil {
		return nil, errors.Wrap(err, "could not decode active downloads response")
	}

	return statuses, nil
}

func decodeGid(result any) (string, error) {
	gid, ok := result.(string)
	if !ok {
		return "", errors.New("unexpected gid in response: %v", result)
	}

	return gid, nil
}

func orEmpty(options Options) Options {
	if options == nil {
		return Options{}
	}

	return options
}
