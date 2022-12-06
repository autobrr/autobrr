package porla

type SysVersions struct {
	Porla SysVersionsPorla `json:"porla"`
}

type SysVersionsPorla struct {
	Commitish string `json:"commitish"`
	Version   string `json:"version"`
}

type TorrentsAddReq struct {
	Ti       string `json:"ti"`
	SavePath string `json:"save_path,omitempty"`
}

type TorrentsAddRes struct {
}
