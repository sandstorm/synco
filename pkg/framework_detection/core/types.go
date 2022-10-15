package core

type FrameworkDetector interface {
	Run() (*DetectionResult, error)
}

type DetectionResult struct {
	PublicWebFolder string
	DumpFiles       []DumpFileCreator
}

type DumpFileCreator struct {
	Filename string
}
