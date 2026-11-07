package database

import (
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		name       string
		args       args
		want       string
		wantConfig pq.Config
		wantErr    string
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
			want:       "host='localhost' port='5432' dbname='postgres' user='postgres' password='PASSWORD' sslmode='disable'",
			wantConfig: pq.Config{Host: "localhost", Port: 5432, Database: "postgres", User: "postgres", Password: "PASSWORD", SSLMode: pq.SSLModeDisable},
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
			want:       "host='localhost' port='5432' dbname='postgres' user='postgres' password='PASSWORD' sslmode='disable' connect_timeout='10'",
			wantConfig: pq.Config{Host: "localhost", Port: 5432, Database: "postgres", User: "postgres", Password: "PASSWORD", SSLMode: pq.SSLModeDisable},
		},
		{
			name: "socket",
			args: args{
				database: "postgres",
				socket:   "/path/to/socket",
			},
			want:       "host='/path/to/socket' dbname='postgres'",
			wantConfig: pq.Config{Host: "/path/to/socket", Database: "postgres"},
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
			want:       "host='/run/postgresql' dbname='postgres' user='postgres' password='PASSWORD' sslmode='disable'",
			wantConfig: pq.Config{Host: "/run/postgresql", Database: "postgres", User: "postgres", Password: "PASSWORD", SSLMode: pq.SSLModeDisable},
		},
		{
			name: "socket with port",
			args: args{
				port:     5433,
				database: "postgres",
				socket:   "/run/postgresql",
			},
			want:       "host='/run/postgresql' port='5433' dbname='postgres'",
			wantConfig: pq.Config{Host: "/run/postgresql", Port: 5433, Database: "postgres"},
		},
		{
			name: "host without port",
			args: args{
				host:     "db.internal",
				database: "postgres",
			},
			want:       "host='db.internal' dbname='postgres'",
			wantConfig: pq.Config{Host: "db.internal", Database: "postgres"},
		},
		{
			name: "port without host",
			args: args{
				port:     5432,
				database: "postgres",
			},
			want:       "port='5432' dbname='postgres'",
			wantConfig: pq.Config{Port: 5432, Database: "postgres"},
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
			want:       `host='localhost' port='5432' dbname='postgres' user='postgres' password='p@ss \'word\\'`,
			wantConfig: pq.Config{Host: "localhost", Port: 5432, Database: "postgres", User: "postgres", Password: `p@ss 'word\`},
		},
		{
			name: "extra params without sslmode",
			args: args{
				host:        "localhost",
				port:        5432,
				database:    "postgres",
				extraParams: "connect_timeout=10",
			},
			want:       "host='localhost' port='5432' dbname='postgres' connect_timeout='10'",
			wantConfig: pq.Config{Host: "localhost", Port: 5432, Database: "postgres"},
		},
		{
			name: "extra params filling an unset setting",
			args: args{
				host:        "localhost",
				port:        5432,
				database:    "postgres",
				extraParams: "user=autobrr",
			},
			want:       "host='localhost' port='5432' dbname='postgres' user='autobrr'",
			wantConfig: pq.Config{Host: "localhost", Port: 5432, Database: "postgres", User: "autobrr"},
		},
		{
			name: "extra params repeating a dedicated setting",
			args: args{
				host:        "localhost",
				port:        5432,
				database:    "postgres",
				sslMode:     "disable",
				extraParams: "sslmode=require",
			},
			wantErr: `postgres: extra param "sslmode" is already set by a dedicated setting`,
		},
		{
			name: "extra params using the lib/pq alias of a dedicated setting",
			args: args{
				host:        "localhost",
				port:        5432,
				database:    "postgres",
				extraParams: "database=other",
			},
			wantErr: `postgres: extra param "database" is already set by a dedicated setting`,
		},
		{
			name: "extra param name with whitespace",
			args: args{
				host:        "localhost",
				port:        5432,
				database:    "postgres",
				extraParams: "dbname%20=other",
			},
			wantErr: `postgres: invalid extra param name "dbname "`,
		},
		{
			name: "extra param name with a newline",
			args: args{
				host:        "localhost",
				port:        5432,
				database:    "postgres",
				extraParams: "dbname%0A=other",
			},
			wantErr: `postgres: invalid extra param name "dbname\n"`,
		},
		{
			name: "extra param name with an equals sign",
			args: args{
				host:        "localhost",
				port:        5432,
				database:    "postgres",
				extraParams: "dbname%3D=other",
			},
			wantErr: `postgres: invalid extra param name "dbname="`,
		},
		{
			name: "empty extra param name",
			args: args{
				host:        "localhost",
				port:        5432,
				database:    "postgres",
				extraParams: "=other",
			},
			wantErr: `postgres: invalid extra param name ""`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PostgresDSN(tt.args.host, tt.args.port, tt.args.user, tt.args.pass, tt.args.database, tt.args.socket, tt.args.sslMode, tt.args.extraParams)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equalf(t, tt.want, got, "PostgresDSN(%v, %v, %v, %v, %v, %v, %v, %v)", tt.args.host, tt.args.port, tt.args.user, tt.args.pass, tt.args.database, tt.args.socket, tt.args.sslMode, tt.args.extraParams)

			cfg, err := pq.NewConfig(got)
			require.NoError(t, err)
			assertDSNConfig(t, tt.wantConfig, cfg)
		})
	}
}

func assertDSNConfig(t *testing.T, want, got pq.Config) {
	t.Helper()
	if want.Host != "" {
		assert.Equal(t, want.Host, got.Host)
	}
	if want.Port != 0 {
		assert.Equal(t, want.Port, got.Port)
	}
	if want.Database != "" {
		assert.Equal(t, want.Database, got.Database)
	}
	if want.User != "" {
		assert.Equal(t, want.User, got.User)
	}
	if want.Password != "" {
		assert.Equal(t, want.Password, got.Password)
	}
	if want.SSLMode != "" {
		assert.Equal(t, want.SSLMode, got.SSLMode)
	}
}
