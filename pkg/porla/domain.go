package porla

type SysVersions struct {
	Porla SysVersionsPorla `json:"porla"`
}

type SysVersionsPorla struct {
	Commitish string `json:"commitish"`
	Version   string `json:"version"`
}
