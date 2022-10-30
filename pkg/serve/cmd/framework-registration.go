package cmd

import (
	"github.com/sandstorm/synco/pkg/common"
	"github.com/sandstorm/synco/pkg/frameworks/flowServe"
)

var RegisteredFrameworks = [...]common.ServeFramework{
	flowServe.NewFlowFramework(),
}
