package laravelServe

import (
	"testing"

	"github.com/sandstorm/synco/v2/pkg/common"
)

func TestToDbCredentialsDetectsDriverAndPortDefault(t *testing.T) {
	tests := []struct {
		name       string
		driver     string
		port       string
		wantDriver common.DbDriver
		wantPort   int
	}{
		{name: "pgsql defaults to port 5432", driver: "pgsql", port: "", wantDriver: common.DbDriverPostgres, wantPort: 5432},
		{name: "mysql defaults to port 3306", driver: "mysql", port: "", wantDriver: common.DbDriverMysql, wantPort: 3306},
		{name: "unrecognized driver falls back to mysql", driver: "sqlite", port: "", wantDriver: common.DbDriverMysql, wantPort: 3306},
		{name: "explicit port overrides the postgres default", driver: "pgsql", port: "6543", wantDriver: common.DbDriverPostgres, wantPort: 6543},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := laravelDatabaseOptions{
				Default: "default",
				Connections: map[string]laravelDatabaseConnectionOptions{
					"default": {Driver: tt.driver, Port: tt.port},
				},
			}
			creds := opts.ToDbCredentials()
			if creds.Driver != tt.wantDriver {
				t.Errorf("Driver = %q, want %q", creds.Driver, tt.wantDriver)
			}
			if creds.Port != tt.wantPort {
				t.Errorf("Port = %d, want %d", creds.Port, tt.wantPort)
			}
		})
	}
}
