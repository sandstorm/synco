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
var all bool

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Wizard to be executed in source",
	Long: `The server part is run on the source / production system, where it automatically discovers used frameworks
and figures out what to extract. Depending on the framework, the system might NOT return the all dataset
- if you want to dump EVERYTHING, use the "--all" arg.`,
	Example: `synco serve`,
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

		pterm.Debug.Printfln("Detecting Frameworks")

		for _, framework := range RegisteredFrameworks {
			progressbar.Add(1)
			pterm.Debug.Printfln("Checking for %s framework", framework.Name())
			if framework.Detect() {
				pterm.Info.Printfln("Found %s framework.", framework.Name())
				transferSession, err := serve.NewSession(identifier, password, listen, all, sigs)
				if err != nil {
					pterm.Fatal.Printfln("Error creating transfer session: %s", err)
				}

				framework.Serve(transferSession)
				pterm.Debug.Printfln("Waiting for ctrl-c")

				// done will never be fired; we'll wait forever here.
				<-done
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
	ServeCmd.Flags().BoolVar(&all, "all", false, "Should dump EVERYTHING? (depending on framework)")
}
