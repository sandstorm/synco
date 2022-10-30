package cmd

import (
	cmd2 "github.com/sandstorm/synco/pkg/receive/cmd"
	"github.com/sandstorm/synco/pkg/serve/cmd"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
)

// !! NOTE: we are not allowed to move this file, as this is needed by the build system at https://github.com/pterm/tag-action/blob/main/entrypoint.sh

import (
	"github.com/pterm/pterm"
)

var rootCmd = &cobra.Command{
	Use:   "synco",
	Short: "an Database and File Dump Downloader for synchronizing production, staging, and local development",
	Long: `This is a template CLI application, which can be used as a boilerplate for awesome CLI tools written in Go.
This template prints the date or time to the terminal.`,
	Example: `synco  date
cli-template date --format 20060102
cli-template time
cli-template time --live`,
	// Uncomment the following lines if your bare application has an action associated with it:
	// RunE: func(cmd *cobra.ReceiveCmd, args []string) error {
	// 	// Your code here
	//
	// 	return nil
	// },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.AddCommand(cmd.ServeCmd)
	rootCmd.AddCommand(cmd2.ReceiveCmd)

	// Fetch user interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		pterm.Warning.Println("user interrupt")
		os.Exit(0)
	}()

	// Execute cobra
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Adds global flags for PTerm settings.
	// Fill the empty strings with the shorthand variant (if you like to have one).
	rootCmd.PersistentFlags().BoolVarP(&pterm.PrintDebugMessages, "debug", "", false, "enable debug messages")
	rootCmd.PersistentFlags().BoolVarP(&pterm.RawOutput, "raw", "", false, "print unstyled raw output (set it if output is written to a file)")

	// Change global PTerm theme
	pterm.ThemeDefault.SectionStyle = *pterm.NewStyle(pterm.FgCyan)
}
