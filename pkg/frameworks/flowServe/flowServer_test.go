package flowServe

import "testing"

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

func TestGenerateS3ResourcesPaths(t *testing.T) {
	for _, tt := range generateS3ResourcesPathTests {
		path := generateS3ResourcePublicPath(&tt.targetConfiguration, tt.resourceSha1, tt.filename)
		if path != tt.wantedPublicUri {
			t.Errorf("publicUri %q does not match %q", path, tt.wantedPublicUri)
		}
	}
}
