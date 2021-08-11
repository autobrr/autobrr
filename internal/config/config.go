package config

import (
	"errors"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/viper"
)

type Cfg struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	LogLevel string `toml:"logLevel"`
	LogPath  string `toml:"logPath"`
	BaseURL  string `toml:"baseUrl"`
}

var Config Cfg

func Defaults() Cfg {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	return Cfg{
		Host:     hostname,
		Port:     8989,
		LogLevel: "DEBUG",
		LogPath:  "",
		BaseURL:  "/",
	}
}

func writeConfig(configPath string, configFile string) error {
	path := filepath.Join(configPath, configFile)

	// check if configPath exists, if not create it
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(configPath, os.ModePerm)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	// check if config exists, if not create it
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		f, err := os.Create(path)
		if err != nil { // perm 0666
			// handle failed create
			log.Printf("error creating file: %q", err)
			return err
		}

		defer f.Close()

		_, err = f.WriteString(`# config.toml

# Hostname / IP
#
# Default: "localhost"
#
host = "127.0.0.1"

# Port
#
# Default: 8989
#
port = 8989

# Base url
# Set custom baseUrl eg /autobrr/ to serve in subdirectory.
# Not needed for subdomain, or by accessing with the :port directly.
#
# Optional
#
#baseUrl = "/autobrr/"

# autobrr logs file
# If not defined, logs to stdout
#
# Optional
#
#logPath = "log/autobrr.log"

# Log level
#
# Default: "DEBUG"
#
# Options: "ERROR", "DEBUG", "INFO", "WARN"
#
logLevel = "DEBUG"`)
		if err != nil {
			log.Printf("error writing contents to file: %v %q", configPath, err)
			return err
		}

		return f.Sync()

	}

	return nil
}

func Read(configPath string) Cfg {
	config := Defaults()

	// or use viper.SetDefault(val, def)
	//viper.SetDefault("host", config.Host)
	//viper.SetDefault("port", config.Port)
	//viper.SetDefault("logLevel", config.LogLevel)
	//viper.SetDefault("logPath", config.LogPath)

	viper.SetConfigType("toml")

	// clean trailing slash from configPath
	configPath = path.Clean(configPath)

	if configPath != "" {
		//viper.SetConfigName("config")

		// check if path and file exists
		// if not, create path and file
		err := writeConfig(configPath, "config.toml")
		if err != nil {
			log.Printf("write error: %q", err)
		}

		viper.SetConfigFile(path.Join(configPath, "config.toml"))
	} else {
		viper.SetConfigName("config")

		// Search config in directories
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.config/autobrr")
		viper.AddConfigPath("$HOME/.autobrr")
	}

	// read config
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("config read error: %q", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Could not unmarshal config file: %v", viper.ConfigFileUsed())
	}

	Config = config

	return config
}
