package common

import (
	"github.com/sandstorm/synco/v2/pkg/receive"
	"github.com/sandstorm/synco/v2/pkg/serve"
)

type ReceiveFramework interface {
	Name() string
	Receive(receiveSession *receive.ReceiveSession)
}

type ServeFramework interface {
	Name() string
	Detect() bool
	Serve(metadata *serve.TransferSession)
}

type DbCredentials struct {
	Host     string
	Port     int
	User     string
	Password string
	DbName   string
}
