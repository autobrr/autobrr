package database

import (
	"path"
)

func DataSourceName(configPath string, name string) string {
	if configPath != "" {
		return path.Join(configPath, name)
	}

	return name
}
