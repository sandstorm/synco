package types

import "github.com/sandstorm/synco/pkg/serve"

type Framework interface {
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
