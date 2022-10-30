package flowReceive

import (
	"github.com/sandstorm/synco/pkg/common"
	"github.com/sandstorm/synco/pkg/receive"
)

type flowReceive struct {
}

func (f flowReceive) Name() string {
	return "Neos/Flow"
}

func (f flowReceive) Receive(receiveSession *receive.ReceiveSession) {
}

func NewFlowFramework() common.ReceiveFramework {
	return &flowReceive{}
}
