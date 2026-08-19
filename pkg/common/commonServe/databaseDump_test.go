package commonServe

import (
	"testing"

	"github.com/sandstorm/synco/v2/pkg/common"
	"github.com/sandstorm/synco/v2/pkg/common/dto"
)

func TestFileSetTypeForDriver(t *testing.T) {
	tests := []struct {
		name   string
		driver common.DbDriver
		want   dto.FileSetType
	}{
		{name: "postgres", driver: common.DbDriverPostgres, want: dto.TYPE_POSTGRESDUMP},
		{name: "mysql", driver: common.DbDriverMysql, want: dto.TYPE_MYSQLDUMP},
		{name: "unset driver defaults to mysql", driver: "", want: dto.TYPE_MYSQLDUMP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fileSetTypeFor(tt.driver)
			if got != tt.want {
				t.Errorf("fileSetTypeFor(%q) = %q, want %q", tt.driver, got, tt.want)
			}
		})
	}
}
