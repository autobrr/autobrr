package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIRCParserGazelleGames_Parse(t *testing.T) {
	t.Parallel()
	type args struct {
		rls  *Release
		vars map[string]string
	}
	type want struct {
		title   string
		release string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "Trouble.in.Paradise-GROUP in Trouble in Paradise",
				},
			},
			want: want{
				title:   "Trouble in Paradise",
				release: "Trouble.in.Paradise-GROUP",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "F.I.L.F. Game Walkthrough v.0.18 in F.I.L.F.",
				},
			},
			want: want{
				title:   "F.I.L.F.",
				release: "F.I.L.F. Game Walkthrough v.0.18",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "Ni no Kuni: Dominion of the Dark Djinn in Ni no Kuni: Dominion of the Dark Djinn",
				},
			},
			want: want{
				title:   "Ni no Kuni: Dominion of the Dark Djinn",
				release: "Ni no Kuni: Dominion of the Dark Djinn",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "Year 2 Remastered by Insaneintherainmusic",
					"category":    "OST",
				},
			},
			want: want{
				title:   "Year 2 Remastered by Insaneintherainmusic",
				release: "Year 2 Remastered by Insaneintherainmusic",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "Lanota v2.23.1 in Lanota",
					"category":    "iOS",
				},
			},
			want: want{
				title:   "Lanota",
				release: "Lanota",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "Korean_Drone_Flying_Tour_Han_River_NSW-SUXXORS in Korean Drone Flying Tour Han River",
					"category":    "Switch",
				},
			},
			want: want{
				title:   "Korean Drone Flying Tour Han River",
				release: "Korean_Drone_Flying_Tour_Han_River_NSW-SUXXORS",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "Carmen_Sandiego_Update_v1.4.0_NSW-VENOM - Update - Version 1.4.0 in Carmen Sandiego",
					"category":    "Switch",
				},
			},
			want: want{
				title:   "Carmen Sandiego",
				release: "Carmen_Sandiego_Update_v1.4.0_NSW-VENOM",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "Colin McRae Rally 3 - Version 1.1 in Colin McRae Rally 3",
					"category":    "Windows",
				},
			},
			want: want{
				title:   "Colin McRae Rally 3",
				release: "Colin McRae Rally 3",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "Soulstone Survivors - Version 1.1.5 (83772) in Soulstone Survivors",
					"category":    "Windows",
				},
			},
			want: want{
				title:   "Soulstone Survivors",
				release: "Soulstone Survivors",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "Digger: Galactic Treasures - Version 1.07",
					"category":    "Windows",
				},
			},
			want: want{
				title:   "Digger: Galactic Treasures",
				release: "Digger: Galactic Treasures",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "Bee.Simulator.The.Hive-RUNE - Version Unknown in Bee Simulator: The Hive",
					"category":    "Windows",
				},
			},
			want: want{
				title:   "Bee Simulator: The Hive",
				release: "Bee.Simulator.The.Hive-RUNE",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := IRCParserGazelleGames{}
			p.Parse(tt.args.rls, tt.args.vars)
			assert.Equal(t, tt.want.release, tt.args.rls.TorrentName)
			assert.Equal(t, tt.want.title, tt.args.rls.Title)
		})
	}
}

func TestIRCParserOrpheus_Parse(t *testing.T) {
	t.Parallel()
	type args struct {
		rls  *Release
		vars map[string]string
	}
	type want struct {
		title   string
		release string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "Orpheus", "ops", "Orpheus"}),
				vars: map[string]string{
					"torrentName": "Busta Rhymes – BEACH BALL (feat. BIA) – [2023] [Single] WEB/FLAC/24bit Lossless",
					"title":       "Busta Rhymes – BEACH BALL (feat. BIA)",
					"year":        "2023",
					"releaseTags": "WEB/FLAC/24bit Lossless",
				},
			},
			want: want{
				title:   "BEACH BALL",
				release: "Busta Rhymes - BEACH BALL (feat. BIA) [2023] (WEB FLAC 24BIT Lossless)",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "Orpheus", "ops", "Orpheus"}),
				vars: map[string]string{
					"torrentName": "Busta Rhymes – BEACH BALL (feat. BIA) – [2023] [Single] CD/FLAC/Lossless",
					"title":       "Busta Rhymes – BEACH BALL (feat. BIA)",
					"year":        "2023",
					"releaseTags": "CD/FLAC/Lossless",
				},
			},
			want: want{
				title:   "BEACH BALL",
				release: "Busta Rhymes - BEACH BALL (feat. BIA) [2023] (CD FLAC Lossless)",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := IRCParserOrpheus{}
			p.Parse(tt.args.rls, tt.args.vars)
			assert.Equal(t, tt.want.release, tt.args.rls.TorrentName)
			assert.Equal(t, tt.want.title, tt.args.rls.Title)
		})
	}
}

func Test_splitInMiddle(t *testing.T) {
	type args struct {
		s   string
		sep string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 string
	}{
		{name: "", args: args{s: "Trouble.in.Paradise-GROUP in Trouble in Paradise", sep: " in "}, want: "Trouble.in.Paradise-GROUP", want1: "Trouble in Paradise"},
		{name: "", args: args{s: "Trouble.in.Paradise-GROUP", sep: " in "}, want: "Trouble.in.Paradise-GROUP", want1: ""},
		{name: "", args: args{s: "Best.Game.Ever-GROUP in Best Game Ever", sep: " in "}, want: "Best.Game.Ever-GROUP", want1: "Best Game Ever"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := splitInMiddle(tt.args.s, tt.args.sep)
			assert.Equalf(t, tt.want, got, "splitInMiddle(%v, %v)", tt.args.s, tt.args.sep)
			assert.Equalf(t, tt.want1, got1, "splitInMiddle(%v, %v)", tt.args.s, tt.args.sep)
		})
	}
}
