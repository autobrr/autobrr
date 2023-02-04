package domain

type Config struct {
	Version           string
	ConfigPath        string
	Host              string `toml:"host"`
	Port              int    `toml:"port"`
	LogLevel          string `toml:"logLevel"`
	LogPath           string `toml:"logPath"`
	LogMaxSize        int    `toml:"logMaxSize"`
	LogMaxBackups     int    `toml:"logMaxBackups"`
	BaseURL           string `toml:"baseUrl"`
	SessionSecret     string `toml:"sessionSecret"`
	CustomDefinitions string `toml:"customDefinitions"`
	CheckForUpdates   bool   `toml:"checkForUpdates"`
	DatabaseType      string `toml:"databaseType"`
	PostgresHost      string `toml:"postgresHost"`
	PostgresPort      int    `toml:"postgresPort"`
	PostgresDatabase  string `toml:"postgresDatabase"`
	PostgresUser      string `toml:"postgresUser"`
	PostgresPass      string `toml:"postgresPass"`
}
