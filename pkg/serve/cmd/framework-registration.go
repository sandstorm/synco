package cmd

import (
	"github.com/sandstorm/synco/pkg/common"
	"github.com/sandstorm/synco/pkg/frameworks/flowServe"
	"github.com/sandstorm/synco/pkg/frameworks/laravelServe"
)

var RegisteredFrameworks = [...]common.ServeFramework{
	flowServe.NewFlowFramework(),
	laravelServe.NewLaravel(),
}
