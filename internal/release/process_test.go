package release

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/autobrr/autobrr/internal/domain"
)

func Test_actionIsArr(t *testing.T) {
	type args struct {
		actions []domain.Action
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "pass_qbit", args: args{actions: []domain.Action{{Name: "qbit", Type: domain.ActionTypeQbittorrent}}}, want: false},
		{name: "pass_deluge_v1", args: args{actions: []domain.Action{{Name: "deluge_v1", Type: domain.ActionTypeDelugeV1}}}, want: false},
		{name: "pass_deluge_v2", args: args{actions: []domain.Action{{Name: "deluge_v2", Type: domain.ActionTypeDelugeV2}}}, want: false},
		{name: "pass_test", args: args{actions: []domain.Action{{Name: "test", Type: domain.ActionTypeTest}}}, want: false},
		{name: "pass_watch_folder", args: args{actions: []domain.Action{{Name: "watch_folder", Type: domain.ActionTypeWatchFolder}}}, want: false},
		{name: "pass_exec", args: args{actions: []domain.Action{{Name: "exec", Type: domain.ActionTypeExec}}}, want: false},
		{name: "match_radarr", args: args{actions: []domain.Action{{Name: "radarr", Type: domain.ActionTypeRadarr}}}, want: true},
		{name: "match_sonarr", args: args{actions: []domain.Action{{Name: "sonarr", Type: domain.ActionTypeSonarr}}}, want: true},
		{name: "match_lidarr", args: args{actions: []domain.Action{{Name: "lidarr", Type: domain.ActionTypeLidarr}}}, want: true},
		{name: "match_mixed", args: args{actions: []domain.Action{{Name: "lidarr", Type: domain.ActionTypeLidarr}, {Name: "deluge", Type: domain.ActionTypeDelugeV2}}}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := actionIsArr(tt.args.actions)
			assert.Equal(t, tt.want, got)
		})
	}
}
