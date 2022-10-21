package source

import (
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/frameworks/flow"
	"github.com/sandstorm/synco/pkg/frameworks/types"
	"github.com/sandstorm/synco/pkg/serve"
	"github.com/spf13/cobra"
	"os"
)

var registeredFrameworks = [...]types.Framework{
	flow.NewFlowFramework(),
}

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
		progressbar, err := pterm.DefaultProgressbar.WithTotal(3).Start()
		pterm.PrintOnErrorf("Error initializing progress bar: %e", err)

		pterm.Info.Printfln("Detecting Frameworks")

		for _, framework := range registeredFrameworks {
			progressbar.Add(1)
			pterm.Debug.Printfln("Checking for %s framework", framework.Name())
			if framework.Detect() {
				pterm.Success.Printfln("Found %s framework.", framework.Name())
				transferSession, err := serve.NewSession(identifier, password, listen)
				if err != nil {
					pterm.Fatal.Printfln("Error creating transfer session: %s", err)
				}

				framework.Serve(transferSession)
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
