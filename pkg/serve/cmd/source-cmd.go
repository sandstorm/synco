package cmd

import (
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/serve"
	"github.com/sandstorm/synco/pkg/util"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var identifier string
var password string
var listen string

var ServeCmd = &cobra.Command{
	Use:     "serve",
	Short:   "Wizard to be executed in source",
	Long:    `...`,
	Example: `synco serve `,
	// Uncomment the following lines if your bare application has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)
		done := make(chan bool, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		var err error
		if len(password) == 0 {
			password, err = util.GenerateRandomString(30)
			if err != nil {
				pterm.Fatal.Printfln("Error generating random password: %s", err)
			}
		}

		if len(identifier) == 0 {
			identifier, err = util.GenerateRandomString(7)
			if err != nil {
				pterm.Fatal.Printfln("Error generating identifier password: %s", err)
			}
		}

		progressbar, err := pterm.DefaultProgressbar.WithTotal(3).Start()
		pterm.PrintOnErrorf("Error initializing progress bar: %e", err)

		pterm.Info.Printfln("Detecting Frameworks")

		for _, framework := range RegisteredFrameworks {
			progressbar.Add(1)
			pterm.Debug.Printfln("Checking for %s framework", framework.Name())
			if framework.Detect() {
				pterm.Success.Printfln("Found %s framework.", framework.Name())
				transferSession, err := serve.NewSession(identifier, password, listen, sigs, done)
				if err != nil {
					pterm.Fatal.Printfln("Error creating transfer session: %s", err)
				}

				framework.Serve(transferSession)
				pterm.Debug.Printfln("Waiting for ctrl-c")
				<-done
				pterm.Debug.Printfln("Exiting")
				return
			}
		}

		pterm.Error.Printfln("No frameworks could be detected. Aborting.")
		//pterm.Error.Printfln("No frameworks could be detected. You can manually create a syncro.yaml config file with %s.", pterm.ThemeDefault.HighlightStyle.Sprint("syncro config init"))
		os.Exit(1)
	},
}

func init() {
	ServeCmd.Flags().StringVar(&identifier, "id", "", "identifier for the decryption")
	ServeCmd.Flags().StringVar(&password, "password", "", "password to encrypt the files for")
	ServeCmd.Flags().StringVar(&listen, "listen", "", "port to create a HTTP server on, if any")
}
