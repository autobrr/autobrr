package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostgresDSN(t *testing.T) {
	type args struct {
		host        string
		port        int
		user        string
		pass        string
		database    string
		socket      string
		sslMode     string
		extraParams string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "default",
			args: args{
				host:     "localhost",
				port:     5432,
				user:     "postgres",
				pass:     "PASSWORD",
				database: "postgres",
				sslMode:  "disable",
				socket:   "",
			},
			want: "postgres://postgres:PASSWORD@localhost:5432/postgres?sslmode=disable",
		},
		{
			name: "default",
			args: args{
				host:        "localhost",
				port:        5432,
				user:        "postgres",
				pass:        "PASSWORD",
				database:    "postgres",
				sslMode:     "disable",
				extraParams: "connect_timeout=10",
				socket:      "",
			},
			want: "postgres://postgres:PASSWORD@localhost:5432/postgres?sslmode=disable&connect_timeout=10",
		},
		{
			name: "default",
			args: args{
				database: "postgres",
				socket:   "/path/to/socket",
			},
			want: "postgres://postgres?host=%2Fpath%2Fto%2Fsocket",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := PostgresDSN(tt.args.host, tt.args.port, tt.args.user, tt.args.pass, tt.args.database, tt.args.socket, tt.args.sslMode, tt.args.extraParams)
			assert.Equalf(t, tt.want, got, "PostgresDSN(%v, %v, %v, %v, %v, %v, %v, %v)", tt.args.host, tt.args.port, tt.args.user, tt.args.pass, tt.args.database, tt.args.socket, tt.args.sslMode, tt.args.extraParams)
		})
	}
}
