package announce

//func Test_service_OnNewLine(t *testing.T) {
//	tfiles := tracker.NewService()
//	tfiles.ReadFiles()
//
//	type fields struct {
//		trackerSvc tracker.Service
//	}
//	type args struct {
//		msg string
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//		{
//			name: "parse announce",
//			fields: fields{
//				trackerSvc: tfiles,
//			},
//			args: args{
//				msg: "New Torrent Announcement: <PC :: Iso>  Name:'debian live 10 6 0 amd64 standard iso' uploaded by 'Anonymous' -  http://www.tracker01.test/torrent/263302",
//			},
//			// expect struct:  category, torrentName uploader freeleech baseurl torrentId
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &service{
//				trackerSvc: tt.fields.trackerSvc,
//			}
//			if err := s.OnNewLine(tt.args.msg); (err != nil) != tt.wantErr {
//				t.Errorf("OnNewLine() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}

//func Test_service_parse(t *testing.T) {
//	type fields struct {
//		trackerSvc tracker.Service
//	}
//	type args struct {
//		serverName  string
//		channelName string
//		announcer   string
//		line     string
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &service{
//				trackerSvc: tt.fields.trackerSvc,
//			}
//			if err := s.parse(tt.args.serverName, tt.args.channelName, tt.args.announcer, tt.args.line); (err != nil) != tt.wantErr {
//				t.Errorf("parse() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}

/*
var (
	tracker01 = domain.TrackerInstance{
		Name:     "T01",
		Enabled:  true,
		Settings: nil,
		Auth:     map[string]string{"rsskey": "000aaa111bbb222ccc333ddd"},
		//IRC:      nil,
		Info: &domain.TrackerInfo{
			Type:      "t01",
			ShortName: "T01",
			LongName:  "Tracker01",
			SiteName:  "www.tracker01.test",
			IRC: domain.TrackerIRCServer{
				Network:        "Tracker01.test",
				ServerNames:    []string{"irc.tracker01.test"},
				ChannelNames:   []string{"#tracker01", "#t01announces"},
				AnnouncerNames: []string{"_AnnounceBot_"},
			},
			ParseInfo: domain.ParseInfo{
				LinePatterns: []domain.TrackerExtractPattern{

					{
						PatternType: "linepattern",
						Optional:    false,
						Regex:       regexp.MustCompile("New Torrent Announcement:\\s*<([^>]*)>\\s*Name:'(.*)' uploaded by '([^']*)'\\s*(freeleech)*\\s*-\\s*https?\\:\\/\\/([^\\/]+\\/)torrent\\/(\\d+)"),
						Vars:        []string{"category", "torrentName", "uploader", "$freeleech", "$baseUrl", "$torrentId"},
					},
				},
				MultiLinePatterns: nil,
				LineMatched: domain.LineMatched{
					Vars: []domain.LineMatchVars{
						{
							Name: "freeleech",
							Vars: []domain.LineMatchVarElem{
								{Type: "string", Value: "false"},
							},
						},
						{
							Name: "torrentUrl",
							Vars: []domain.LineMatchVarElem{
								{Type: "string", Value: "https://"},
								{Type: "var", Value: "$baseUrl"},
								{Type: "string", Value: "rss/download/"},
								{Type: "var", Value: "$torrentId"},
								{Type: "string", Value: "/"},
								{Type: "var", Value: "rsskey"},
								{Type: "string", Value: "/"},
								{Type: "varenc", Value: "torrentName"},
								{Type: "string", Value: ".torrent"},
							},
						},
					},
					Extract:     nil,
					LineMatchIf: nil,
					VarReplace:  nil,
					SetRegex: &domain.SetRegex{
						SrcVar:   "$freeleech",
						Regex:    regexp.MustCompile("freeleech"),
						VarName:  "freeleech",
						NewValue: "true",
					},
					ExtractOne: domain.ExtractOne{Extract: nil},
					ExtractTags: domain.ExtractTags{
						Name:     "",
						SrcVar:   "",
						Split:    "",
						Regex:    nil,
						SetVarIf: nil,
					},
				},
				Ignore: []domain.TrackerIgnore{},
			},
		},
	}
	tracker05 = domain.TrackerInstance{
		Name:     "T05",
		Enabled:  true,
		Settings: nil,
		Auth:     map[string]string{"authkey": "000aaa111bbb222ccc333ddd", "torrent_pass": "eee444fff555ggg666hhh777"},
		//IRC:      nil,
		Info: &domain.TrackerInfo{
			Type:      "t05",
			ShortName: "T05",
			LongName:  "Tracker05",
			SiteName:  "tracker05.test",
			IRC: domain.TrackerIRCServer{
				Network:        "Tracker05.test",
				ServerNames:    []string{"irc.tracker05.test"},
				ChannelNames:   []string{"#t05-announce"},
				AnnouncerNames: []string{"Drone"},
			},
			ParseInfo: domain.ParseInfo{
				LinePatterns: []domain.TrackerExtractPattern{

					{
						PatternType: "linepattern",
						Optional:    false,
						Regex:       regexp.MustCompile("^(.*)\\s+-\\s+https?:.*[&amp;\\?]id=.*https?\\:\\/\\/([^\\/]+\\/).*[&amp;\\?]id=(\\d+)\\s*-\\s*(.*)"),
						Vars:        []string{"torrentName", "$baseUrl", "$torrentId", "tags"},
					},
				},
				MultiLinePatterns: nil,
				LineMatched: domain.LineMatched{
					Vars: []domain.LineMatchVars{
						{
							Name: "scene",
							Vars: []domain.LineMatchVarElem{
								{Type: "string", Value: "false"},
							},
						},
						{
							Name: "log",
							Vars: []domain.LineMatchVarElem{
								{Type: "string", Value: "false"},
							},
						},
						{
							Name: "cue",
							Vars: []domain.LineMatchVarElem{
								{Type: "string", Value: "false"},
							},
						},
						{
							Name: "freeleech",
							Vars: []domain.LineMatchVarElem{
								{Type: "string", Value: "false"},
							},
						},
						{
							Name: "torrentUrl",
							Vars: []domain.LineMatchVarElem{
								{Type: "string", Value: "https://"},
								{Type: "var", Value: "$baseUrl"},
								{Type: "string", Value: "torrents.php?action=download&id="},
								{Type: "var", Value: "$torrentId"},
								{Type: "string", Value: "&authkey="},
								{Type: "var", Value: "authkey"},
								{Type: "string", Value: "&torrent_pass="},
								{Type: "var", Value: "torrent_pass"},
							},
						},
					},
					Extract: []domain.Extract{
						{SrcVar: "torrentName", Optional: true, Regex: regexp.MustCompile("[(\\[]((?:19|20)\\d\\d)[)\\]]"), Vars: []string{"year"}},
						{SrcVar: "$releaseTags", Optional: true, Regex: regexp.MustCompile("([\\d.]+)%"), Vars: []string{"logScore"}},
					},
					LineMatchIf: nil,
					VarReplace: []domain.ParseVarReplace{
						{Name: "tags", SrcVar: "tags", Regex: regexp.MustCompile("[._]"), Replace: " "},
					},
					SetRegex: nil,
					ExtractOne: domain.ExtractOne{Extract: []domain.Extract{
						{SrcVar: "torrentName", Optional: false, Regex: regexp.MustCompile("^(.+?) - ([^\\[]+).*\\[(\\d{4})\\] \\[([^\\[]+)\\] - ([^\\-\\[\\]]+)"), Vars: []string{"name1", "name2", "year", "releaseType", "$releaseTags"}},
						{SrcVar: "torrentName", Optional: false, Regex: regexp.MustCompile("^([^\\-]+)\\s+-\\s+(.+)"), Vars: []string{"name1", "name2"}},
						{SrcVar: "torrentName", Optional: false, Regex: regexp.MustCompile("(.*)"), Vars: []string{"name1"}},
					}},
					ExtractTags: domain.ExtractTags{
						Name:   "",
						SrcVar: "$releaseTags",
						Split:  "/",
						Regex:  []*regexp.Regexp{regexp.MustCompile("^(?:5\\.1 Audio|\\.m4a|Various.*|~.*|&gt;.*)$")},
						SetVarIf: []domain.SetVarIf{
							{VarName: "format", Value: "", NewValue: "", Regex: regexp.MustCompile("^(?:MP3|FLAC|Ogg Vorbis|AAC|AC3|DTS)$")},
							{VarName: "bitrate", Value: "", NewValue: "", Regex: regexp.MustCompile("Lossless$")},
							{VarName: "bitrate", Value: "", NewValue: "", Regex: regexp.MustCompile("^(?:vbr|aps|apx|v\\d|\\d{2,4}|\\d+\\.\\d+|q\\d+\\.[\\dx]+|Other)?(?:\\s*kbps|\\s*kbits?|\\s*k)?(?:\\s*\\(?(?:vbr|cbr)\\)?)?$")},
							{VarName: "media", Value: "", NewValue: "", Regex: regexp.MustCompile("^(?:CD|DVD|Vinyl|Soundboard|SACD|DAT|Cassette|WEB|Blu-ray|Other)$")},
							{VarName: "scene", Value: "Scene", NewValue: "true", Regex: nil},
							{VarName: "log", Value: "Log", NewValue: "true", Regex: nil},
							{VarName: "cue", Value: "Cue", NewValue: "true", Regex: nil},
							{VarName: "freeleech", Value: "Freeleech!", NewValue: "true", Regex: nil},
						},
					},
				},
				Ignore: []domain.TrackerIgnore{},
			},
		},
	}
)
*/

//func Test_service_parse(t *testing.T) {
//	type fields struct {
//		name   string
//		trackerSvc     tracker.Service
//		queues map[string]chan string
//	}
//	type args struct {
//		ti      *domain.TrackerInstance
//		message string
//	}
//
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		want    *domain.Announce
//		wantErr bool
//	}{
//		{
//			name: "tracker01_no_freeleech",
//			fields: fields{
//				name:   "T01",
//				trackerSvc:     nil,
//				queues: make(map[string]chan string),
//			}, args: args{
//				ti:      &tracker01,
//				message: "New Torrent Announcement: <PC :: Iso>  Name:'debian live 10 6 0 amd64 standard iso' uploaded by 'Anonymous' - http://www.tracker01.test/torrent/263302",
//			},
//			want: &domain.Announce{
//				Freeleech:   false,
//				Category:    "PC :: Iso",
//				Name: "debian live 10 6 0 amd64 standard iso",
//				Uploader:    "Anonymous",
//				TorrentUrl:  "https://www.tracker01.test/rss/download/263302/000aaa111bbb222ccc333ddd/debian+live+10+6+0+amd64+standard+iso.torrent",
//				Site:        "T01",
//			},
//			wantErr: false,
//		},
//		{
//			name: "tracker01_freeleech",
//			fields: fields{
//				name:   "T01",
//				trackerSvc:     nil,
//				queues: make(map[string]chan string),
//			}, args: args{
//				ti:      &tracker01,
//				message: "New Torrent Announcement: <PC :: Iso>  Name:'debian live 10 6 0 amd64 standard iso' uploaded by 'Anonymous' freeleech - http://www.tracker01.test/torrent/263302",
//			},
//			want: &domain.Announce{
//				Freeleech:   true,
//				Category:    "PC :: Iso",
//				Name: "debian live 10 6 0 amd64 standard iso",
//				Uploader:    "Anonymous",
//				TorrentUrl:  "https://www.tracker01.test/rss/download/263302/000aaa111bbb222ccc333ddd/debian+live+10+6+0+amd64+standard+iso.torrent",
//				Site:        "T01",
//			},
//			wantErr: false,
//		},
//		{
//			name: "tracker05_01",
//			fields: fields{
//				name:   "T05",
//				trackerSvc:     nil,
//				queues: make(map[string]chan string),
//			}, args: args{
//				ti:      &tracker05,
//				message: "Roy Buchanan - Loading Zone [1977] - FLAC / Lossless / Log / 100% / Cue / CD - http://passtheheadphones.me/torrents.php?id=97614 / http://tracker05.test/torrents.php?action=download&id=1382972 - blues, rock, classic.rock,jazz,blues.rock,electric.blues",
//			},
//			want: &domain.Announce{
//				Name1:       "Roy Buchanan - Loading Zone [1977] - FLAC / Lossless / Log / 100% / Cue / CD",
//				Name2:       "Loading Zone [1977] - FLAC / Lossless / Log / 100% / Cue / CD",
//				Freeleech:   false,
//				Name: "Roy Buchanan - Loading Zone [1977] - FLAC / Lossless / Log / 100% / Cue / CD",
//				TorrentUrl:  "https://tracker05.test/torrents.php?action=download&id=1382972&authkey=000aaa111bbb222ccc333ddd&torrent_pass=eee444fff555ggg666hhh777",
//				Site:        "T05",
//				Tags:        "blues, rock, classic rock,jazz,blues rock,electric blues",
//				Log:         "true",
//				Cue:         true,
//				Format:      "FLAC",
//				Bitrate:     "Lossless",
//				Media:       "CD",
//				Scene:       false,
//				Year:        1977,
//			},
//			wantErr: false,
//		},
//		{
//			name: "tracker05_02",
//			fields: fields{
//				name:   "T05",
//				trackerSvc:     nil,
//				queues: make(map[string]chan string),
//			}, args: args{
//				ti:      &tracker05,
//				message: "Heirloom - Road to the Isles [1998] [Album] - FLAC / Lossless / Log / 100% / Cue / CD - http://tracker05.test/torrents.php?id=72158898 / http://tracker05.test/torrents.php?action=download&id=29910415 - 1990s, folk, world_music, celtic",
//			},
//			want: &domain.Announce{
//				ReleaseType: "Album",
//				Name1:       "Heirloom - Road to the Isles [1998] [Album] - FLAC / Lossless / Log / 100% / Cue / CD",
//				Name2:       "Road to the Isles [1998] [Album] - FLAC / Lossless / Log / 100% / Cue / CD",
//				Freeleech:   false,
//				Name: "Heirloom - Road to the Isles [1998] [Album] - FLAC / Lossless / Log / 100% / Cue / CD",
//				TorrentUrl:  "https://tracker05.test/torrents.php?action=download&id=29910415&authkey=000aaa111bbb222ccc333ddd&torrent_pass=eee444fff555ggg666hhh777",
//				Site:        "T05",
//				Tags:        "1990s, folk, world music, celtic",
//				Log:         "true",
//				Cue:         true,
//				Format:      "FLAC",
//				Bitrate:     "Lossless",
//				Media:       "CD",
//				Scene:       false,
//				Year:        1998,
//			},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &service{
//				name:   tt.fields.name,
//				trackerSvc:     tt.fields.trackerSvc,
//				queues: tt.fields.queues,
//			}
//			got, err := s.parse(tt.args.ti, tt.args.message)
//
//			if (err != nil) != tt.wantErr {
//				t.Errorf("parse() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			assert.Equal(t, tt.want, got)
//		})
//	}
//}

//func Test_service_parseSingleLine(t *testing.T) {
//	type fields struct {
//		name   string
//		ts     tracker.Service
//		queues map[string]chan string
//	}
//	type args struct {
//		ti   *domain.TrackerInstance
//		line string
//	}
//
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		want    *domain.Announce
//		wantErr bool
//	}{
//		{
//			name: "tracker01_no_freeleech",
//			fields: fields{
//				name:   "T01",
//				ts:     nil,
//				queues: make(map[string]chan string),
//			}, args: args{
//				ti:   &tracker01,
//				line: "New Torrent Announcement: <PC :: Iso>  Name:'debian live 10 6 0 amd64 standard iso' uploaded by 'Anonymous' - http://www.tracker01.test/torrent/263302",
//			},
//			want: &domain.Announce{
//				Freeleech:   false,
//				Category:    "PC :: Iso",
//				Name: "debian live 10 6 0 amd64 standard iso",
//				Uploader:    "Anonymous",
//				TorrentUrl:  "https://www.tracker01.test/rss/download/263302/000aaa111bbb222ccc333ddd/debian+live+10+6+0+amd64+standard+iso.torrent",
//				Site:        "T01",
//			},
//			wantErr: false,
//		},
//		{
//			name: "tracker01_freeleech",
//			fields: fields{
//				name:   "T01",
//				ts:     nil,
//				queues: make(map[string]chan string),
//			}, args: args{
//				ti:   &tracker01,
//				line: "New Torrent Announcement: <PC :: Iso>  Name:'debian live 10 6 0 amd64 standard iso' uploaded by 'Anonymous' freeleech - http://www.tracker01.test/torrent/263302",
//			},
//			want: &domain.Announce{
//				Freeleech:   true,
//				Category:    "PC :: Iso",
//				Name: "debian live 10 6 0 amd64 standard iso",
//				Uploader:    "Anonymous",
//				TorrentUrl:  "https://www.tracker01.test/rss/download/263302/000aaa111bbb222ccc333ddd/debian+live+10+6+0+amd64+standard+iso.torrent",
//				Site:        "T01",
//			},
//			wantErr: false,
//		},
//		{
//			name: "tracker05_01",
//			fields: fields{
//				name:   "T05",
//				ts:     nil,
//				queues: make(map[string]chan string),
//			}, args: args{
//				ti:   &tracker05,
//				line: "Roy Buchanan - Loading Zone [1977] - FLAC / Lossless / Log / 100% / Cue / CD - http://passtheheadphones.me/torrents.php?id=97614 / http://tracker05.test/torrents.php?action=download&id=1382972 - blues, rock, classic.rock,jazz,blues.rock,electric.blues",
//			},
//			want: &domain.Announce{
//				Name1:       "Roy Buchanan - Loading Zone [1977] - FLAC / Lossless / Log / 100% / Cue / CD",
//				Name2:       "Loading Zone [1977] - FLAC / Lossless / Log / 100% / Cue / CD",
//				Freeleech:   false,
//				Name: "Roy Buchanan - Loading Zone [1977] - FLAC / Lossless / Log / 100% / Cue / CD",
//				TorrentUrl:  "https://tracker05.test/torrents.php?action=download&id=1382972&authkey=000aaa111bbb222ccc333ddd&torrent_pass=eee444fff555ggg666hhh777",
//				Site:        "T05",
//				Tags:        "blues, rock, classic rock,jazz,blues rock,electric blues",
//				//Log:         "true",
//				//Cue:         true,
//				//Format:      "FLAC",
//				//Bitrate:     "Lossless",
//				//Media:       "CD",
//				Log:     "false",
//				Cue:     false,
//				Format:  "",
//				Bitrate: "",
//				Media:   "",
//				Scene:   false,
//				Year:    1977,
//			},
//			wantErr: false,
//		},
//		{
//			name: "tracker05_02",
//			fields: fields{
//				name:   "T05",
//				ts:     nil,
//				queues: make(map[string]chan string),
//			}, args: args{
//				ti:   &tracker05,
//				line: "Heirloom - Road to the Isles [1998] [Album] - FLAC / Lossless / Log / 100% / Cue / CD - http://tracker05.test/torrents.php?id=72158898 / http://tracker05.test/torrents.php?action=download&id=29910415 - 1990s, folk, world_music, celtic",
//			},
//			want: &domain.Announce{
//				ReleaseType: "Album",
//				Name1:       "Heirloom - Road to the Isles [1998] [Album] - FLAC / Lossless / Log / 100% / Cue / CD",
//				Name2:       "Road to the Isles [1998] [Album] - FLAC / Lossless / Log / 100% / Cue / CD",
//				Freeleech:   false,
//				Name: "Heirloom - Road to the Isles [1998] [Album] - FLAC / Lossless / Log / 100% / Cue / CD",
//				TorrentUrl:  "https://tracker05.test/torrents.php?action=download&id=29910415&authkey=000aaa111bbb222ccc333ddd&torrent_pass=eee444fff555ggg666hhh777",
//				Site:        "T05",
//				Tags:        "1990s, folk, world music, celtic",
//				Log:         "true",
//				Cue:         true,
//				Format:      "FLAC",
//				Bitrate:     "Lossless",
//				Media:       "CD",
//				Scene:       false,
//				Year:        1998,
//			},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &service{
//				name:       tt.fields.name,
//				trackerSvc: tt.fields.ts,
//				queues:     tt.fields.queues,
//			}
//
//			announce := domain.Announce{
//				Site: tt.fields.name,
//				//Line: msg,
//			}
//			got, err := s.parseSingleLine(tt.args.ti, tt.args.line, &announce)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("parseSingleLine() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//
//			assert.Equal(t, tt.want, got)
//		})
//	}
//}

//func Test_service_extractReleaseInfo(t *testing.T) {
//	type fields struct {
//		name   string
//		queues map[string]chan string
//	}
//	type args struct {
//		varMap      map[string]string
//		releaseName string
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		{
//			name: "test_01",
//			fields: fields{
//				name: "", queues: nil,
//			},
//			args: args{
//				varMap:      map[string]string{},
//				releaseName: "Heirloom - Road to the Isles [1998] [Album] - FLAC / Lossless / Log / 100% / Cue / CD",
//			},
//			wantErr: false,
//		},
//		{
//			name: "test_02",
//			fields: fields{
//				name: "", queues: nil,
//			},
//			args: args{
//				varMap:      map[string]string{},
//				releaseName: "Lost S06E07 720p WEB-DL DD 5.1 H.264 - LP",
//			},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &service{
//				queues: tt.fields.queues,
//			}
//			if err := s.extractReleaseInfo(tt.args.varMap, tt.args.releaseName); (err != nil) != tt.wantErr {
//				t.Errorf("extractReleaseInfo() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
