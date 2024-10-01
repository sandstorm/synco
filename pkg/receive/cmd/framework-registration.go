package cmd

import (
	"github.com/sandstorm/synco/v2/pkg/common"
	"github.com/sandstorm/synco/v2/pkg/frameworks/flowReceive"
)

var RegisteredFrameworks = [...]common.ReceiveFramework{
	flowReceive.NewFlowFramework(),
}
