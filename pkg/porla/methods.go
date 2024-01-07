// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package porla

import "context"

func (c *Client) Version() (*SysVersionsPorla, error) {
	response, err := c.rpcClient.Call("sys.versions")
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, response.Error
	}

	var versions *SysVersions
	if err = response.GetObject(&versions); err != nil {
		return nil, err
	}

	return &versions.Porla, nil
}

func (c *Client) TorrentsAdd(ctx context.Context, req *TorrentsAddReq) error {
	response, err := c.rpcClient.CallCtx(ctx, "torrents.add", req)
	if err != nil {
		return err
	}

	if response.Error != nil {
		return response.Error
	}

	var res *TorrentsAddRes
	if err = response.GetObject(&res); err != nil {
		return err
	}

	return nil
}

func (c *Client) TorrentsList(ctx context.Context, filters *TorrentsListFilters) (*TorrentsListRes, error) {
	response, err := c.rpcClient.CallCtx(ctx, "torrents.list", TorrentsListReq{Filters: filters})
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, response.Error
	}

	var res *TorrentsListRes
	if err = response.GetObject(&res); err != nil {
		return nil, err
	}

	return res, nil
}
