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
