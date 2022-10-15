package flow

import (
	"fmt"
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/framework_detection/core"
	"os"
)

type flowDetector struct {
}

var emptyDetectionResult = &core.DetectionResult{}

func (f flowDetector) Run() (*core.DetectionResult, error) {
	if _, err := os.Stat("flow"); err == nil {
		pterm.Info.Println("Flow framework found")
		return &core.DetectionResult{
			PublicWebFolder: "Web/",
		}, nil
	} else {
		pterm.Debug.Println("Flow framework not found")
	}
	return emptyDetectionResult, fmt.Errorf("error")
}

func NewFlowDetector() core.FrameworkDetector {
	return &flowDetector{}
}
