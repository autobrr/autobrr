package qbittorrent

import "testing"

func Test_buildUrl(t *testing.T) {
	type args struct {
		settings Settings
		endpoint string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "build_url_1",
			args: args{
				settings: Settings{
					Hostname:      "https://qbit.domain.ltd",
					Port:          0,
					Username:      "",
					Password:      "",
					TLS:           true,
					TLSSkipVerify: false,
					protocol:      "",
				},
				endpoint: "auth/login",
			},
			want: "https://qbit.domain.ltd/api/v2/auth/login",
		},
		{
			name: "build_url_2",
			args: args{
				settings: Settings{
					Hostname:      "http://qbit.domain.ltd",
					Port:          0,
					Username:      "",
					Password:      "",
					TLS:           false,
					TLSSkipVerify: false,
					protocol:      "",
				},
				endpoint: "/auth/login",
			},
			want: "http://qbit.domain.ltd/api/v2/auth/login",
		},
		{
			name: "build_url_3",
			args: args{
				settings: Settings{
					Hostname:      "https://qbit.domain.ltd:8080",
					Port:          0,
					Username:      "",
					Password:      "",
					TLS:           true,
					TLSSkipVerify: false,
					protocol:      "",
				},
				endpoint: "/auth/login",
			},
			want: "https://qbit.domain.ltd:8080/api/v2/auth/login",
		},
		{
			name: "build_url_4",
			args: args{
				settings: Settings{
					Hostname:      "qbit.domain.ltd:8080",
					Port:          0,
					Username:      "",
					Password:      "",
					TLS:           false,
					TLSSkipVerify: false,
					protocol:      "",
				},
				endpoint: "/auth/login",
			},
			want: "http://qbit.domain.ltd:8080/api/v2/auth/login",
		},
		{
			name: "build_url_5",
			args: args{
				settings: Settings{
					Hostname:      "qbit.domain.ltd",
					Port:          8080,
					Username:      "",
					Password:      "",
					TLS:           false,
					TLSSkipVerify: false,
					protocol:      "",
				},
				endpoint: "/auth/login",
			},
			want: "http://qbit.domain.ltd:8080/api/v2/auth/login",
		},
		{
			name: "build_url_6",
			args: args{
				settings: Settings{
					Hostname:      "qbit.domain.ltd",
					Port:          443,
					Username:      "",
					Password:      "",
					TLS:           true,
					TLSSkipVerify: false,
					protocol:      "",
				},
				endpoint: "/auth/login",
			},
			want: "https://qbit.domain.ltd/api/v2/auth/login",
		},
		{
			name: "build_url_6",
			args: args{
				settings: Settings{
					Hostname:      "qbit.domain.ltd",
					Port:          10200,
					Username:      "",
					Password:      "",
					TLS:           false,
					TLSSkipVerify: false,
					protocol:      "",
				},
				endpoint: "/auth/login",
			},
			want: "http://qbit.domain.ltd:10200/api/v2/auth/login",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildUrl(tt.args.settings, tt.args.endpoint); got != tt.want {
				t.Errorf("buildUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}
