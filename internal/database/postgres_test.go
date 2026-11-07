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
			name: "tcp",
			args: args{
				host:     "localhost",
				port:     5432,
				user:     "postgres",
				pass:     "PASSWORD",
				database: "postgres",
				sslMode:  "disable",
				socket:   "",
			},
			want: "host='localhost' port='5432' dbname='postgres' user='postgres' password='PASSWORD' sslmode='disable'",
		},
		{
			name: "tcp with extra params",
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
			want: "host='localhost' port='5432' dbname='postgres' user='postgres' password='PASSWORD' sslmode='disable' connect_timeout='10'",
		},
		{
			name: "socket",
			args: args{
				database: "postgres",
				socket:   "/path/to/socket",
			},
			want: "host='/path/to/socket' dbname='postgres'",
		},
		{
			name: "socket with credentials",
			args: args{
				user:     "postgres",
				pass:     "PASSWORD",
				database: "postgres",
				socket:   "/run/postgresql",
				sslMode:  "disable",
			},
			want: "host='/run/postgresql' dbname='postgres' user='postgres' password='PASSWORD' sslmode='disable'",
		},
		{
			name: "socket with port",
			args: args{
				port:     5433,
				database: "postgres",
				socket:   "/run/postgresql",
			},
			want: "host='/run/postgresql' port='5433' dbname='postgres'",
		},
		{
			name: "host without port",
			args: args{
				host:     "db.internal",
				database: "postgres",
			},
			want: "host='db.internal' dbname='postgres'",
		},
		{
			name: "port without host",
			args: args{
				port:     5432,
				database: "postgres",
			},
			want: "port='5432' dbname='postgres'",
		},
		{
			name: "password needing quoting",
			args: args{
				host:     "localhost",
				port:     5432,
				user:     "postgres",
				pass:     `p@ss 'word\`,
				database: "postgres",
			},
			want: `host='localhost' port='5432' dbname='postgres' user='postgres' password='p@ss \'word\\'`,
		},
		{
			name: "extra params without sslmode",
			args: args{
				host:        "localhost",
				port:        5432,
				database:    "postgres",
				extraParams: "connect_timeout=10",
			},
			want: "host='localhost' port='5432' dbname='postgres' connect_timeout='10'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := PostgresDSN(tt.args.host, tt.args.port, tt.args.user, tt.args.pass, tt.args.database, tt.args.socket, tt.args.sslMode, tt.args.extraParams)
			assert.Equalf(t, tt.want, got, "PostgresDSN(%v, %v, %v, %v, %v, %v, %v, %v)", tt.args.host, tt.args.port, tt.args.user, tt.args.pass, tt.args.database, tt.args.socket, tt.args.sslMode, tt.args.extraParams)
		})
	}
}
