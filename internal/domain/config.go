package domain

type Settings struct {
	Host  string `toml:"host"`
	Debug bool
}

//type AppConfig struct {
//	Settings `toml:"settings"`
//	Trackers []Tracker `mapstructure:"tracker"`
//}
