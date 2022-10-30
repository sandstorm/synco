package common

import (
	"github.com/sandstorm/synco/pkg/receive"
	"github.com/sandstorm/synco/pkg/serve"
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
