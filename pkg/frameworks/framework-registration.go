package frameworks

import (
	"github.com/sandstorm/synco/pkg/frameworks/flow"
	"github.com/sandstorm/synco/pkg/frameworks/types"
)

var RegisteredFrameworks = [...]types.Framework{
	flow.NewFlowFramework(),
}
