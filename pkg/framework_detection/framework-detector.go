package framework_detection

import (
	"github.com/sandstorm/synco/pkg/framework_detection/core"
	"github.com/sandstorm/synco/pkg/framework_detection/flow"
)

var detectors = [...]core.FrameworkDetector{
	flow.NewFlowDetector(),
}

type frameworkDetector struct {
}

var emptyDetectionResult = &core.DetectionResult{}

func (f frameworkDetector) Run() (*core.DetectionResult, error) {
	var lastError error = nil
	for _, detector := range detectors {
		println("!!!! DETECTOR")
		detectionResult, err := detector.Run()
		if err != nil {
			lastError = err
		} else {
			return detectionResult, nil
		}
	}

	return emptyDetectionResult, lastError
}

func NewFrameworkDetector() core.FrameworkDetector {
	return frameworkDetector{}
}
