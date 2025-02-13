// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_processTitle(t *testing.T) {
	type args struct {
		title        string
		matchRelease bool
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "test_01",
			args: args{
				title:        "The Quick Brown Fox (2022)",
				matchRelease: false,
			},
			want: []string{"The?Quick?Brown?Fox", "The?Quick?Brown?Fox*2022", "The?Quick?Brown?Fox*2022?"},
		},
		{
			name: "test_02",
			args: args{
				title:        "The Matrix     -        Reloaded (2929)",
				matchRelease: false,
			},
			want: []string{"The?Matrix*Reloaded", "The?Matrix*Reloaded*2929", "The?Matrix*Reloaded*2929?"},
		},
		{
			name: "test_03",
			args: args{
				title:        "The Matrix -(Test)- Reloaded (2929)",
				matchRelease: false,
			},
			want: []string{"The?Matrix*Reloaded", "The?Matrix*Test*Reloaded*2929?", "The?Matrix*Test*Reloaded*2929"},
		},
		{
			name: "test_04",
			args: args{
				title:        "The Marvelous Mrs. Maisel",
				matchRelease: false,
			},
			want: []string{"The?Marvelous?Mrs*Maisel"},
		},
		{
			name: "test_05",
			args: args{
				title:        "Arrr!! The Title (2020)",
				matchRelease: false,
			},
			want: []string{"Arrr*The?Title", "Arrr*The?Title*2020", "Arrr*The?Title*2020?"},
		},
		{
			name: "test_06",
			args: args{
				title:        "Whose Line Is It Anyway? (US)",
				matchRelease: false,
			},
			want: []string{"Whose?Line?Is?It?Anyway", "Whose?Line?Is?It?Anyway*US", "Whose?Line?Is?It?Anyway?", "Whose?Line?Is?It?Anyway*US?"},
		},
		{
			name: "test_07",
			args: args{
				title:        "MasterChef (US)",
				matchRelease: false,
			},
			want: []string{"MasterChef*US", "MasterChef", "MasterChef*US?"},
		},
		{
			name: "test_08",
			args: args{
				title:        "Brooklyn Nine-Nine",
				matchRelease: false,
			},
			want: []string{"Brooklyn?Nine?Nine"},
		},
		{
			name: "test_09",
			args: args{
				title:        "S.W.A.T.",
				matchRelease: false,
			},
			want: []string{"S?W?A?T?", "S?W?A?T"},
		},
		{
			name: "test_10",
			args: args{
				title:        "The Handmaid's Tale",
				matchRelease: false,
			},
			want: []string{"The?Handmaid?s?Tale", "The?Handmaids?Tale"},
		},
		{
			name: "test_11",
			args: args{
				title:        "The Handmaid's Tale (US)",
				matchRelease: false,
			},
			want: []string{"The?Handmaid?s?Tale*US", "The?Handmaids?Tale*US", "The?Handmaid?s?Tale", "The?Handmaids?Tale", "The?Handmaid?s?Tale*US?", "The?Handmaids?Tale*US?"},
		},
		{
			name: "test_12",
			args: args{
				title:        "Monsters, Inc.",
				matchRelease: false,
			},
			want: []string{"Monsters*Inc?", "Monsters*Inc"},
		},
		{
			name: "test_13",
			args: args{
				title:        "Hello Tomorrow!",
				matchRelease: false,
			},
			want: []string{"Hello?Tomorrow?", "Hello?Tomorrow"},
		},
		{
			name: "test_14",
			args: args{
				title:        "Be Cool, Scooby-Doo!",
				matchRelease: false,
			},
			want: []string{"Be?Cool*Scooby?Doo?", "Be?Cool*Scooby?Doo"},
		},
		{
			name: "test_15",
			args: args{
				title:        "Scooby-Doo! Mystery Incorporated",
				matchRelease: false,
			},
			want: []string{"Scooby?Doo*Mystery?Incorporated"},
		},
		{
			name: "test_16",
			args: args{
				title:        "Master.Chef (US)",
				matchRelease: false,
			},
			want: []string{"Master?Chef*US", "Master?Chef", "Master?Chef*US?"},
		},
		{
			name: "test_17",
			args: args{
				title:        "Whose Line Is It Anyway? (US)",
				matchRelease: false,
			},
			want: []string{"Whose?Line?Is?It?Anyway*US", "Whose?Line?Is?It?Anyway?", "Whose?Line?Is?It?Anyway", "Whose?Line?Is?It?Anyway*US?"},
		},
		{
			name: "test_18",
			args: args{
				title:        "90 Day Fiancé: Pillow Talk",
				matchRelease: false,
			},
			want: []string{"90?Day?Fianc*Pillow?Talk"},
		},
		{
			name: "test_19",
			args: args{
				title:        "進撃の巨人",
				matchRelease: false,
			},
			want: []string{"進撃の巨人"},
		},
		{
			name: "test_20",
			args: args{
				title:        "呪術廻戦 0: 東京都立呪術高等専門学校",
				matchRelease: false,
			},
			want: []string{"呪術廻戦?0*東京都立呪術高等専門学校"},
		},
		{
			name: "test_21",
			args: args{
				title:        "-!",
				matchRelease: false,
			},
			want: []string{"-!"},
		},
		{
			name: "test_22",
			args: args{
				title:        "A\u00a0Quiet\u00a0Place:\u00a0Day One",
				matchRelease: false,
			},
			want: []string{"A?Quiet?Place*Day?One"},
		},
		{
			name: "test_23",
			args: args{
				title:        "Whose Line Is It Anyway? (US) (1932)",
				matchRelease: false,
			},
			want: []string{"Whose?Line?Is?It?Anyway", "Whose?Line?Is?It?Anyway?", "Whose?Line?Is?It?Anyway*US*1932", "Whose?Line?Is?It?Anyway*US*1932?"},
		},
		{
			name: "test_24",
			args: args{
				title:        "What If…?",
				matchRelease: false,
			},
			want: []string{"What?If", "What?If*"},
		},
		{
			name: "test_25",
			args: args{
				title:        "Shōgun (2024)",
				matchRelease: false,
			},
			want: []string{"Sh?gun*2024?", "Sh?gun*2024", "Sh?gun"},
		},
		{
			name: "test_26",
			args: args{
				title:        "Pinball FX3 - Bethesda® Pinball",
				matchRelease: false,
			},
			want: []string{
				"Pinball?FX3*Bethesda*Pinball",
			},
		},
		{
			name: "test_27",
			args: args{
				title:        "Sally Goes to the Mall",
				matchRelease: true,
			},
			want: []string{"*Sally?Goes?to?the?Mall*"},
		},
		{
			name: "test_28",
			args: args{
				title:        "*****… (los asteriscos…)",
				matchRelease: false,
			},
			want: []string{"*los?asteriscos*", "*los?asteriscos"},
		},
		{
			name: "test_29",
			args: args{
				title:        "The Office (US)",
				matchRelease: false,
			},
			want: []string{"The?Office", "The?Office*US", "The?Office*US?"},
		},
		{
			name: "test_30",
			args: args{
				title:        "this is him (can’t be anyone else)",
				matchRelease: false,
			},
			want: []string{"this?is?him*can?t?be?anyone?else?", "this?is?him*can?t?be?anyone?else", "this?is?him*cant?be?anyone?else?", "this?is?him*cant?be?anyone?else"},
		},
		{
			name: "test_31",
			args: args{
				title:        "solo leveling 2ª temporada -ergam-se das sombras-",
				matchRelease: false,
			},
			want: []string{"solo?leveling?2*temporada*ergam?se?das?sombras", "solo?leveling?2*temporada*ergam?se?das?sombras?"},
		},
		{
			name: "test_32",
			args: args{
				title:        "pokémon",
				matchRelease: false,
			},
			want: []string{"pok?mon"},
		},
		{
			name: "test_33",
			args: args{
				title:        "What If…?",
				matchRelease: true,
			},
			want: []string{"*What?If*"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// order seem to be random so lets check if the elements are what we expect
			assert.ElementsMatch(t, tt.want, processTitle(tt.args.title, tt.args.matchRelease))
		})
	}
}
