package cmd

import (
	"github.com/sandstorm/synco/pkg/common"
	"github.com/sandstorm/synco/pkg/frameworks/flowReceive"
)

var RegisteredFrameworks = [...]common.ReceiveFramework{
	flowReceive.NewFlowFramework(),
}
