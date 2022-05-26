package domain

import (
	"os"
	"path"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Version           string `toml:"-"`
	ConfigPath        string `toml:"-"`
	Host              string `toml:"host"`
	Port              int    `toml:"port"`
	LogLevel          string `toml:"logLevel"`
	LogPath           string `toml:"logPath"`
	BaseURL           string `toml:"baseUrl"`
	SessionSecret     string `toml:"sessionSecret"`
	CustomDefinitions string `toml:"customDefinitions"`
	DatabaseType      string `toml:"databaseType"`
	PostgresHost      string `toml:"postgresHost"`
	PostgresPort      int    `toml:"postgresPort"`
	PostgresDatabase  string `toml:"postgresDatabase"`
	PostgresUser      string `toml:"postgresUser"`
	PostgresPass      string `toml:"postgresPass"`
}

func (c Config) GetPreferredLogDir() (string, []string) {
	// 0. Check if ~/.config/autobrr/ is accessible to the current user
	// 1. Check if ~/.config/autobrr/log/ is accessible to the current user
	// 2. Check if golang can find the temp directory and use that.
	// 3. If neither 1 nor 2 were successful, bail with an error message.
	// NOTE: If neither $XDG_CONFIG_HOME nor $HOME are defined, UserConfigDir will return an error.
	configDir, err := os.UserConfigDir()

	// Keep track of errors, if any. Might help diagnose misconfiguration problems and such.
	var discoveredErrors []string
	if err == nil {
		// If we managed to find the user config directory,
		// then return ~/.config/autobrr/logs as the preferred log dir
		logDir := path.Join(configDir, "autobrr", "logs")
		return logDir, discoveredErrors
	} else {
		discoveredErrors = append(discoveredErrors, err.Error())
	}

	for _, dir := range [3]string{"/var/log/", "/opt/", os.TempDir()} {
		if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
			return path.Join(dir, "autobrr", "logs"), discoveredErrors
		} else {
			discoveredErrors = append(discoveredErrors, err.Error())
		}
	}

	return "", discoveredErrors
}

func (c *Config) UpdateConfig(file string) error {
	c.LogPath = file

	//// create log dir before
	//// (in case problems arise)
	//// empty LogPath indicates that no log file is needed
	//if strings.TrimSpace(user.LogPath) != "" {
	//	err = os.MkdirAll(user.LogPath, os.ModePerm)
	//	if err != nil {
	//		return errors.New("failed to create log dir: " + err.Error())
	//	}
	//}

	cfgPath := path.Join(c.ConfigPath, "config.toml")

	f, err := os.OpenFile(cfgPath, os.O_CREATE|os.O_RDWR, os.ModePerm)
	//f, err := os.Open("./config.toml")
	if err != nil {
		// failed to create/open the file
		//log.Fatal(err)
		return err
	}
	//defer f.Close()

	if err := toml.NewEncoder(f).Encode(c); err != nil {
		// failed to encode
		//log.Fatal(err)
		return err
	}
	if err := f.Close(); err != nil {
		// failed to close the file
		//log.Fatal(err)
		return err

	}

	return nil
}
