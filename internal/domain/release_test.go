package domain

import (
	"testing"
	"time"
)

func TestReleaseInfo_Parse(t *testing.T) {
	type fields struct {
		ID               int64
		Status           ReleaseStatus
		Rejections       []string
		Indexer          string
		FilterName       string
		Protocol         ReleaseProtocol
		Implementation   ReleaseImplementation
		Timestamp        time.Time
		TorrentID        string
		GroupID          string
		TorrentName      string
		Raw              string
		Title            string
		Category         string
		Season           int
		Episode          int
		Year             int
		Resolution       string
		Source           string
		Codec            string
		Container        string
		HDR              string
		Audio            string
		Group            string
		Region           string
		Edition          string
		Hardcoded        bool
		Proper           bool
		Repack           bool
		Website          string
		Language         string
		Unrated          bool
		Hybrid           bool
		Size             uint64
		ThreeD           bool
		Artists          []string
		Type             string
		Format           string
		Bitrate          string
		LogScore         int
		HasLog           bool
		HasCue           bool
		IsScene          bool
		Origin           string
		Tags             []string
		Freeleech        bool
		FreeleechPercent int
		Uploader         string
		PreTime          string
		TorrentURL       string
		Filter           *Filter
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "parse_1", fields: fields{
			ID:               0,
			Status:           "",
			Rejections:       nil,
			Indexer:          "",
			FilterName:       "",
			Protocol:         "",
			Implementation:   "",
			Timestamp:        time.Time{},
			TorrentID:        "",
			GroupID:          "",
			TorrentName:      "Servant S01 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-FLUX",
			Raw:              "",
			Title:            "",
			Category:         "",
			Season:           0,
			Episode:          0,
			Year:             0,
			Resolution:       "",
			Source:           "",
			Codec:            "",
			Container:        "",
			HDR:              "",
			Audio:            "",
			Group:            "",
			Region:           "",
			Edition:          "",
			Hardcoded:        false,
			Proper:           false,
			Repack:           false,
			Website:          "",
			Language:         "",
			Unrated:          false,
			Hybrid:           false,
			Size:             0,
			ThreeD:           false,
			Artists:          nil,
			Type:             "",
			Format:           "",
			Bitrate:          "",
			LogScore:         0,
			HasLog:           false,
			HasCue:           false,
			IsScene:          false,
			Origin:           "",
			Tags:             nil,
			Freeleech:        false,
			FreeleechPercent: 0,
			Uploader:         "",
			PreTime:          "",
			TorrentURL:       "",
			Filter:           nil,
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Release{
				ID:               tt.fields.ID,
				Status:           tt.fields.Status,
				Rejections:       tt.fields.Rejections,
				Indexer:          tt.fields.Indexer,
				FilterName:       tt.fields.FilterName,
				Protocol:         tt.fields.Protocol,
				Implementation:   tt.fields.Implementation,
				Timestamp:        tt.fields.Timestamp,
				TorrentID:        tt.fields.TorrentID,
				GroupID:          tt.fields.GroupID,
				Name:             tt.fields.TorrentName,
				Raw:              tt.fields.Raw,
				Title:            tt.fields.Title,
				Category:         tt.fields.Category,
				Season:           tt.fields.Season,
				Episode:          tt.fields.Episode,
				Year:             tt.fields.Year,
				Resolution:       tt.fields.Resolution,
				Source:           tt.fields.Source,
				Codec:            tt.fields.Codec,
				Container:        tt.fields.Container,
				HDR:              tt.fields.HDR,
				Audio:            tt.fields.Audio,
				Group:            tt.fields.Group,
				Region:           tt.fields.Region,
				Edition:          tt.fields.Edition,
				Hardcoded:        tt.fields.Hardcoded,
				Proper:           tt.fields.Proper,
				Repack:           tt.fields.Repack,
				Website:          tt.fields.Website,
				Language:         tt.fields.Language,
				Unrated:          tt.fields.Unrated,
				Hybrid:           tt.fields.Hybrid,
				Size:             tt.fields.Size,
				ThreeD:           tt.fields.ThreeD,
				Artists:          tt.fields.Artists,
				Type:             tt.fields.Type,
				Format:           tt.fields.Format,
				Bitrate:          tt.fields.Bitrate,
				LogScore:         tt.fields.LogScore,
				HasLog:           tt.fields.HasLog,
				HasCue:           tt.fields.HasCue,
				IsScene:          tt.fields.IsScene,
				Origin:           tt.fields.Origin,
				Tags:             tt.fields.Tags,
				Freeleech:        tt.fields.Freeleech,
				FreeleechPercent: tt.fields.FreeleechPercent,
				Uploader:         tt.fields.Uploader,
				PreTime:          tt.fields.PreTime,
				TorrentURL:       tt.fields.TorrentURL,
				Filter:           tt.fields.Filter,
			}
			if err := r.Parse(); (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReleaseInfo_CheckFilter(t *testing.T) {
	type args struct {
		filter Filter
	}
	tests := []struct {
		name   string
		fields *Release
		args   args
		want   bool
	}{
		{
			name:   "test_1",
			fields: &Release{Size: uint64(10000000001)},
			args: args{
				filter: Filter{
					Enabled:       true,
					FilterGeneral: FilterGeneral{MinSize: "10 GB", MaxSize: "20GB"},
				},
			},
			want: true,
		},
		{
			name:   "test_2",
			fields: &Release{Size: uint64(30000000001)},
			args: args{
				filter: Filter{
					Enabled:       true,
					FilterGeneral: FilterGeneral{MinSize: "10 GB", MaxSize: "20GB"},
				},
			},
			want: false,
		},
		{
			name:   "test_no_size",
			fields: &Release{Size: uint64(0)},
			args: args{
				filter: Filter{
					Enabled:       true,
					FilterGeneral: FilterGeneral{MinSize: "10 GB", MaxSize: "20GB"},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Release{
				ID:                          tt.fields.ID,
				Status:                      tt.fields.Status,
				Rejections:                  tt.fields.Rejections,
				Indexer:                     tt.fields.Indexer,
				FilterName:                  tt.fields.FilterName,
				Protocol:                    tt.fields.Protocol,
				Implementation:              tt.fields.Implementation,
				Timestamp:                   tt.fields.Timestamp,
				TorrentID:                   tt.fields.TorrentID,
				GroupID:                     tt.fields.GroupID,
				Name:                        tt.fields.Name,
				Raw:                         tt.fields.Raw,
				Title:                       tt.fields.Title,
				Category:                    tt.fields.Category,
				Season:                      tt.fields.Season,
				Episode:                     tt.fields.Episode,
				Year:                        tt.fields.Year,
				Resolution:                  tt.fields.Resolution,
				Source:                      tt.fields.Source,
				Codec:                       tt.fields.Codec,
				Container:                   tt.fields.Container,
				HDR:                         tt.fields.HDR,
				Audio:                       tt.fields.Audio,
				Group:                       tt.fields.Group,
				Region:                      tt.fields.Region,
				Edition:                     tt.fields.Edition,
				Hardcoded:                   tt.fields.Hardcoded,
				Proper:                      tt.fields.Proper,
				Repack:                      tt.fields.Repack,
				Website:                     tt.fields.Website,
				Language:                    tt.fields.Language,
				Unrated:                     tt.fields.Unrated,
				Hybrid:                      tt.fields.Hybrid,
				Size:                        tt.fields.Size,
				ThreeD:                      tt.fields.ThreeD,
				Artists:                     tt.fields.Artists,
				Type:                        tt.fields.Type,
				Format:                      tt.fields.Format,
				Bitrate:                     tt.fields.Bitrate,
				LogScore:                    tt.fields.LogScore,
				HasLog:                      tt.fields.HasLog,
				HasCue:                      tt.fields.HasCue,
				IsScene:                     tt.fields.IsScene,
				Origin:                      tt.fields.Origin,
				Tags:                        tt.fields.Tags,
				Freeleech:                   tt.fields.Freeleech,
				FreeleechPercent:            tt.fields.FreeleechPercent,
				Uploader:                    tt.fields.Uploader,
				PreTime:                     tt.fields.PreTime,
				TorrentURL:                  tt.fields.TorrentURL,
				AdditionalSizeCheckRequired: tt.fields.AdditionalSizeCheckRequired,
				FilterID:                    tt.fields.FilterID,
				Filter:                      tt.fields.Filter,
			}
			if got := r.CheckFilter(tt.args.filter); got != tt.want {
				t.Errorf("CheckFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}
