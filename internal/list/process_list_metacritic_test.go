package list

import (
	"fmt"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/list/provider/metacritic"
	"github.com/stretchr/testify/assert"
)

func TestMetacriticProcessor_process(t *testing.T) {
	type fields struct {
		client *metacritic.Client
		list   *domain.List
	}
	type args struct {
		data *metacritic.ListResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domain.FilterUpdate
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "",
			fields: fields{
				list: &domain.List{
					MatchRelease: true,
				},
			},
			args: args{
				data: &metacritic.ListResponse{
					Title: "",
					Albums: []metacritic.Album{
						{
							Artist: "Dinosaur Jr.",
							Title:  "There Near",
						},

						{
							Artist: "Little Big Town",
							Title:  "It's A Dying Art",
						},
						{
							Artist: "Mike D 5D",
							Title:  "Thank You",
						},
						{
							Artist: "Pulp",
							Title:  "Live!",
						},
						{
							Artist: "Angus & Julia Stone",
							Title:  "Karaoke Bar",
						},
						{
							Artist: "Arab Strap",
							Title:  "Half-Told Tales",
						},
					},
				},
			},
			want: &domain.FilterUpdate{
				Artists:       new(""),
				Albums:        new(""),
				MatchReleases: new("*Angus?Julia?Stone*Karaoke?Bar*,*Arab?Strap*Half?Told?Tales*,*Dinosaur?Jr*There?Near*,*Little?Big?Town*It?s?A?Dying?Art*,*Little?Big?Town*Its?A?Dying?Art*,*Mike?D?5D*Thank?You*,*Pulp*Live*"),
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &MetacriticProcessor{
				client: tt.fields.client,
				list:   tt.fields.list,
			}
			got, err := p.process(tt.args.data)
			if !tt.wantErr(t, err, fmt.Sprintf("process(%v)", tt.args.data)) {
				return
			}
			assert.Equalf(t, tt.want, got, "process(%v)", tt.args.data)
		})
	}
}
