package flowServe

import (
	"testing"

	"github.com/sandstorm/synco/v2/pkg/common"
)

type generateS3ResourcesPathTest struct {
	filename            string
	resourceSha1        string
	targetConfiguration flowResourceTarget
	wantedPublicUri     string
}

var generateS3ResourcesPathTests = []generateS3ResourcesPathTest{
	{
		filename:     "sample-image.jpg",
		resourceSha1: "3233621371f429bfc0e36b47c12b116b688055bf",
		targetConfiguration: flowResourceTarget{
			Target: "Flownative\\Aws\\S3\\S3Target",
			TargetOptions: struct {
				Path                   string `yaml:"path"`
				BaseUri                string `yaml:"baseUri"`
				Bucket                 string `yaml:"bucket"`
				KeyPrefix              string `yaml:"keyPrefix"`
				PersistentResourceUris struct {
					Pattern string `yaml:"pattern"`
				} `yaml:"persistentResourceUris"`
			}{Path: "", BaseUri: "https://cdn.vendor.com/r/", Bucket: "project-prod-cdn-export", KeyPrefix: "r/"},
		},
		wantedPublicUri: "https://cdn.vendor.com/r/3233621371f429bfc0e36b47c12b116b688055bf/sample-image.jpg",
	},
	{
		filename:     "sample-image.jpg",
		resourceSha1: "2b5a802db2bc2f3e5eb7f7d9720201abc5cc511a",
		targetConfiguration: flowResourceTarget{
			Target: "Flownative\\Aws\\S3\\S3Target",
			TargetOptions: struct {
				Path                   string `yaml:"path"`
				BaseUri                string `yaml:"baseUri"`
				Bucket                 string `yaml:"bucket"`
				KeyPrefix              string `yaml:"keyPrefix"`
				PersistentResourceUris struct {
					Pattern string `yaml:"pattern"`
				} `yaml:"persistentResourceUris"`
			}{
				Path:      "",
				BaseUri:   "https://fsn1.your-objectstorage.com",
				Bucket:    "project-prod-web-assets",
				KeyPrefix: "project/site",
				PersistentResourceUris: struct {
					Pattern string `yaml:"pattern"`
				}{Pattern: "/web-assets/{keyPrefix}/{sha1}/{filename}"},
			},
		},
		wantedPublicUri: "/web-assets/project/site/2b5a802db2bc2f3e5eb7f7d9720201abc5cc511a/sample-image.jpg",
	},
}

func TestToDbCredentialsDetectsDriverAndPortDefault(t *testing.T) {
	tests := []struct {
		name       string
		driver     string
		port       string
		wantDriver common.DbDriver
		wantPort   int
	}{
		{name: "pdo_pgsql defaults to port 5432", driver: "pdo_pgsql", port: "", wantDriver: common.DbDriverPostgres, wantPort: 5432},
		{name: "pdo_mysql defaults to port 3306", driver: "pdo_mysql", port: "", wantDriver: common.DbDriverMysql, wantPort: 3306},
		{name: "unrecognized driver falls back to mysql", driver: "pdo_sqlite", port: "", wantDriver: common.DbDriverMysql, wantPort: 3306},
		{name: "explicit port overrides the postgres default", driver: "pdo_pgsql", port: "6543", wantDriver: common.DbDriverPostgres, wantPort: 6543},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := flowPersistenceBackendOptions{Driver: tt.driver, Port: tt.port}
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

func TestGenerateS3ResourcesPaths(t *testing.T) {
	for _, tt := range generateS3ResourcesPathTests {
		path := generateS3ResourcePublicPath(&tt.targetConfiguration, tt.resourceSha1, tt.filename)
		if path != tt.wantedPublicUri {
			t.Errorf("publicUri %q does not match %q", path, tt.wantedPublicUri)
		}
	}
}
