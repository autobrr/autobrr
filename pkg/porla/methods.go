package porla

func (c *Client) Version() (*SysVersionsPorla, error) {
	response, err := c.rpcClient.Call("sys.versions")

	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, response.Error
	}

	var versions *SysVersions
	err = response.GetObject(&versions)

	if err != nil {
		return nil, err
	}

	return &versions.Porla, nil
}

func (c *Client) TorrentsAdd(req *TorrentsAddReq) error {
	response, err := c.rpcClient.Call("torrents.add", req)

	if err != nil {
		return err
	}

	if response.Error != nil {
		return response.Error
	}

	var res *TorrentsAddRes
	err = response.GetObject(&res)

	if err != nil {
		return err
	}

	return nil
}
