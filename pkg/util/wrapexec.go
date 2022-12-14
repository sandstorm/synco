package util

import (
	"bufio"
	"github.com/pterm/pterm"
	"os/exec"
	"strings"
)

var mocks map[string]string = make(map[string]string)

func ResetCommandMocks() {
	mocks = make(map[string]string)
}

func RegisterCommandMock(commandString string, output string) {
	mocks[commandString] = output
}

func RunWrappedCommand(cmd *exec.Cmd) (output string, err error) {
	commandString := strings.Join(cmd.Args, " ")
	if output, ok := mocks[commandString]; ok {
		pterm.Debug.Printfln("Mocking command: %s", commandString)
		return output, nil
	}

	pterm.Debug.Printfln("Executing command: %s", commandString)

	// Get a pipe to read from standard out
	r, _ := cmd.StdoutPipe()

	// Use the same pipe for standard error
	cmd.Stderr = cmd.Stdout

	// Make a new channel which will be used to ensure we get all output
	done := make(chan struct{})

	// Create a scanner which scans r in a line-by-line fashion
	scanner := bufio.NewScanner(r)

	// Use the scanner to scan the output line by line and log it
	// It's running in a goroutine so that it doesn't block

	var outputString strings.Builder

	go func() {
		// Read line by line and process it
		for scanner.Scan() {
			line := scanner.Text()
			outputString.WriteString(line)
			outputString.WriteString("\n")
			pterm.Debug.Printfln("Output: %s", line)
		}

		// We're all done, unblock the channel
		done <- struct{}{}
	}()

	// Start the command and check for errors
	err = cmd.Run()

	// Wait for all output to be processed
	<-done

	return outputString.String(), err
}
