package porla

type SysVersions struct {
	Porla SysVersionsPorla `json:"porla"`
}

type SysVersionsPorla struct {
	Commitish string `json:"commitish"`
	Version   string `json:"version"`
}

type TorrentsAddReq struct {
	DownloadLimit int64  `json:"download_limit,omitempty"`
	SavePath      string `json:"save_path,omitempty"`
	Ti            string `json:"ti"`
	UploadLimit   int64  `json:"upload_limit,omitempty"`
}

type TorrentsAddRes struct {
}
