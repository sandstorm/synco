package main

import (
	"github.com/sandstorm/synco/cmd"
	"github.com/sandstorm/synco/pkg/kubernetes"
)

func main() {
	kubernetes.KubernetesInit()
	cmd.Execute()
}
