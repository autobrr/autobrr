package domain

type Config struct {
	Host          string `toml:"host"`
	Port          int    `toml:"port"`
	LogLevel      string `toml:"logLevel"`
	LogPath       string `toml:"logPath"`
	BaseURL       string `toml:"baseUrl"`
	SessionSecret string `toml:"sessionSecret"`
}
