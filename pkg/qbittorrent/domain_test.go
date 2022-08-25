package qbittorrent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func PtrBool(b bool) *bool {
	return &b
}

func PtrStr(s string) *string {
	return &s
}

func PtrInt64(i int64) *int64 {
	return &i
}
func PtrFloat64(f float64) *float64 {
	return &f
}

func TestTorrentAddOptions_Prepare(t *testing.T) {
	layoutNone := ContentLayoutSubfolderNone
	layoutCreate := ContentLayoutSubfolderCreate
	layoutOriginal := ContentLayoutOriginal
	type fields struct {
		Paused             *bool
		SkipHashCheck      *bool
		ContentLayout      *ContentLayout
		SavePath           *string
		AutoTMM            *bool
		Category           *string
		Tags               *string
		LimitUploadSpeed   *int64
		LimitDownloadSpeed *int64
		LimitRatio         *float64
		LimitSeedTime      *int64
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]string
	}{
		{
			name: "test_01",
			fields: fields{
				Paused:             nil,
				SkipHashCheck:      PtrBool(true),
				ContentLayout:      nil,
				SavePath:           PtrStr("/home/test/torrents"),
				AutoTMM:            nil,
				Category:           PtrStr("test"),
				Tags:               PtrStr("limited,slow"),
				LimitUploadSpeed:   PtrInt64(100000),
				LimitDownloadSpeed: PtrInt64(100000),
				LimitRatio:         PtrFloat64(2.0),
				LimitSeedTime:      PtrInt64(100),
			},
			want: map[string]string{
				"skip_checking":    "true",
				"autoTMM":          "false",
				"ratioLimit":       "2.00",
				"savepath":         "/home/test/torrents",
				"seedingTimeLimit": "100",
				"category":         "test",
				"tags":             "limited,slow",
				"upLimit":          "100000000",
				"dlLimit":          "100000000",
			},
		},
		{
			name: "test_02",
			fields: fields{
				Paused:             nil,
				SkipHashCheck:      PtrBool(true),
				ContentLayout:      &layoutCreate,
				SavePath:           PtrStr("/home/test/torrents"),
				AutoTMM:            nil,
				Category:           PtrStr("test"),
				Tags:               PtrStr("limited,slow"),
				LimitUploadSpeed:   PtrInt64(100000),
				LimitDownloadSpeed: PtrInt64(100000),
				LimitRatio:         PtrFloat64(2.0),
				LimitSeedTime:      PtrInt64(100),
			},
			want: map[string]string{
				"skip_checking":    "true",
				"root_folder":      "true",
				"autoTMM":          "false",
				"ratioLimit":       "2.00",
				"savepath":         "/home/test/torrents",
				"seedingTimeLimit": "100",
				"category":         "test",
				"tags":             "limited,slow",
				"upLimit":          "100000000",
				"dlLimit":          "100000000",
			},
		},
		{
			name: "test_03",
			fields: fields{
				Paused:             nil,
				SkipHashCheck:      PtrBool(true),
				ContentLayout:      &layoutNone,
				SavePath:           PtrStr("/home/test/torrents"),
				AutoTMM:            nil,
				Category:           PtrStr("test"),
				Tags:               PtrStr("limited,slow"),
				LimitUploadSpeed:   PtrInt64(100000),
				LimitDownloadSpeed: PtrInt64(100000),
				LimitRatio:         PtrFloat64(2.0),
				LimitSeedTime:      PtrInt64(100),
			},
			want: map[string]string{
				"skip_checking":    "true",
				"root_folder":      "false",
				"autoTMM":          "false",
				"ratioLimit":       "2.00",
				"savepath":         "/home/test/torrents",
				"seedingTimeLimit": "100",
				"category":         "test",
				"tags":             "limited,slow",
				"upLimit":          "100000000",
				"dlLimit":          "100000000",
			},
		},
		{
			name: "test_04",
			fields: fields{
				Paused:             nil,
				SkipHashCheck:      PtrBool(true),
				ContentLayout:      &layoutOriginal,
				SavePath:           PtrStr("/home/test/torrents"),
				AutoTMM:            nil,
				Category:           PtrStr("test"),
				Tags:               PtrStr("limited,slow"),
				LimitUploadSpeed:   PtrInt64(100000),
				LimitDownloadSpeed: PtrInt64(100000),
				LimitRatio:         PtrFloat64(2.0),
				LimitSeedTime:      PtrInt64(100),
			},
			want: map[string]string{
				"skip_checking":    "true",
				"autoTMM":          "false",
				"ratioLimit":       "2.00",
				"savepath":         "/home/test/torrents",
				"seedingTimeLimit": "100",
				"category":         "test",
				"tags":             "limited,slow",
				"upLimit":          "100000000",
				"dlLimit":          "100000000",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &TorrentAddOptions{
				Paused:             tt.fields.Paused,
				SkipHashCheck:      tt.fields.SkipHashCheck,
				ContentLayout:      tt.fields.ContentLayout,
				SavePath:           tt.fields.SavePath,
				AutoTMM:            tt.fields.AutoTMM,
				Category:           tt.fields.Category,
				Tags:               tt.fields.Tags,
				LimitUploadSpeed:   tt.fields.LimitUploadSpeed,
				LimitDownloadSpeed: tt.fields.LimitDownloadSpeed,
				LimitRatio:         tt.fields.LimitRatio,
				LimitSeedTime:      tt.fields.LimitSeedTime,
			}

			got := o.Prepare()
			assert.Equal(t, tt.want, got)
		})
	}
}
