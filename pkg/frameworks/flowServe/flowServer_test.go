package flowServe

import "testing"

func TestGenerateResourcePathForS3(t *testing.T) {
	persistentTargetConfig := flowResourceTarget{
		Target: "Flownative\\Aws\\S3\\S3Target",
		TargetOptions: struct {
			Path      string `yaml:"path"`
			BaseUri   string `yaml:"baseUri"`
			Bucket    string `yaml:"bucket"`
			KeyPrefix string `yaml:"keyPrefix"`
		}{Path: "", BaseUri: "https://cdn.vendor.com/r/", Bucket: "project-prod-cdn-export", KeyPrefix: "r/"},
	}

	path, entry := generateResourcePathForS3(
		"sample-image.jpg", "3233621371f429bfc0e36b47c12b116b688055bf", 256, &persistentTargetConfig)

	wantedPath := "Resources/3/2/3/3/3233621371f429bfc0e36b47c12b116b688055bf"
	wantedPublicUri := "https://cdn.vendor.com/r/3233621371f429bfc0e36b47c12b116b688055bf/sample-image.jpg"

	if wantedPath != path {
		t.Errorf("path %q does not match %q", path, wantedPath)
	}
	if entry.PublicUri != wantedPublicUri {
		t.Errorf("publicUri %q does not match %q", entry.PublicUri, "TODO")
	}
}
