package cmd

import (
	"github.com/sandstorm/synco/v2/pkg/common"
	"github.com/sandstorm/synco/v2/pkg/frameworks/flowServe"
	"github.com/sandstorm/synco/v2/pkg/frameworks/laravelServe"
)

var RegisteredFrameworks = [...]common.ServeFramework{
	flowServe.NewFlowFramework(),
	laravelServe.NewLaravel(),
}
