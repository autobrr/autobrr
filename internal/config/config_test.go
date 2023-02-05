package config

import (
	"reflect"
	"sync"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
)

func TestAppConfig_processLines(t *testing.T) {
	type fields struct {
		Config *domain.Config
		m      sync.Mutex
	}
	type args struct {
		lines []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			name: "append missing",
			fields: fields{
				Config: &domain.Config{CheckForUpdates: true},
				m:      sync.Mutex{},
			},
			args: args{[]string{}},
			want: []string{"# Check for updates", "#", "checkForUpdates = true"},
		},
		{
			name: "update existing",
			fields: fields{
				Config: &domain.Config{CheckForUpdates: true},
				m:      sync.Mutex{},
			},
			args: args{[]string{"# Check for updates", "#", "#checkForUpdates = false"}},
			want: []string{"# Check for updates", "#", "checkForUpdates = true"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &AppConfig{
				Config: tt.fields.Config,
				m:      tt.fields.m,
			}
			if got := c.processLines(tt.args.lines); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processLines() = %v, want %v", got, tt.want)
			}
		})
	}
}
