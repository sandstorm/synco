package main

import (
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/v2/cmd"
	"os"
	"os/signal"
)

func main() {
	//kubernetes.KubernetesInit()

	// Fetch user interrupt - WORKAROUND: Somehow, this only works when in main.go - if we move it somewhere else, it will break.
	// weird stuff :D I don't understand signals yet.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		pterm.Warning.Println("user interrupt")
		os.Exit(0)
	}()

	cmd.Execute()
}
